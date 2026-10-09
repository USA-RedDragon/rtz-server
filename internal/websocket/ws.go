package websocket

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/USA-RedDragon/rtz-server/internal/config"
	"github.com/USA-RedDragon/rtz-server/internal/db/models"
	"github.com/USA-RedDragon/rtz-server/internal/metrics"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/nats-io/nats.go"
	"gorm.io/gorm"
)

const bufferSize = 1024

// handlerGrace bounds how long a closed connection waits for its OnMessage
// calls to return before OnDisconnect runs anyway.
const handlerGrace = 5 * time.Second

// Websocket handles the connections to one websocket endpoint.
type Websocket interface {
	// OnConnect is called once a connection is established and returns the
	// state for that connection. ctx is cancelled as soon as the connection
	// ends, and from then on w refuses new messages.
	OnConnect(ctx context.Context, r *http.Request, w Writer, device *models.Device, db *gorm.DB, nats *nats.Conn, metrics *metrics.Metrics, conn *websocket.Conn) Connection
}

// Connection is the state of a single websocket connection.
type Connection interface {
	// OnMessage is called in its own goroutine for every message received.
	// ctx is cancelled when the connection ends.
	OnMessage(ctx context.Context, msg []byte, msgType int)
	// OnDisconnect is called exactly once after the connection has ended and
	// its OnMessage calls have returned, or handlerGrace has passed.
	OnDisconnect()
}

type WSHandler struct {
	wsUpgrader websocket.Upgrader
	handler    Websocket
	conn       *websocket.Conn
	// pongWait is how long a connection may go without a pong before it is closed.
	pongWait time.Duration
	// pingPeriod is how often a ping is sent, and must be less than pongWait.
	pingPeriod time.Duration
}

func CreateHandler(ws Websocket, config *config.Config) func(*gin.Context) {
	return createHandler(ws, config, pongWait, pingPeriod)
}

func createHandler(ws Websocket, config *config.Config, pongWait, pingPeriod time.Duration) func(*gin.Context) {
	handler := &WSHandler{
		wsUpgrader: websocket.Upgrader{
			HandshakeTimeout: 0,
			ReadBufferSize:   bufferSize,
			WriteBufferSize:  bufferSize,
			WriteBufferPool:  nil,
			Subprotocols:     []string{},
			CheckOrigin: func(r *http.Request) bool {
				origin := r.Header.Get("Origin")
				if origin == "" {
					return true
				}
				return originAllowed(origin, config.HTTP.CORSHosts)
			},
			EnableCompression: true,
		},
		handler:    ws,
		pongWait:   pongWait,
		pingPeriod: pingPeriod,
	}

	return func(c *gin.Context) {
		dongleID, ok := c.Params.Get("dongle_id")
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{errorKey: "dongle_id is required"})
			return
		}
		maybeNats, ok := c.Get("nats")
		if !ok && config.NATS.Enabled {
			slog.Error("Failed to get NATS from context")
			c.JSON(http.StatusInternalServerError, gin.H{errorKey: msgTryAgainLater})
			return
		}
		nats, ok := maybeNats.(*nats.Conn)
		if !ok {
			nats = nil
		}
		db, ok := c.MustGet("db").(*gorm.DB)
		if !ok {
			slog.Error("Failed to get db from context")
			c.JSON(http.StatusInternalServerError, gin.H{errorKey: msgTryAgainLater})
			return
		}
		metrics, ok := c.MustGet("metrics").(*metrics.Metrics)
		if !ok {
			slog.Error("Failed to get metrics from context")
			c.JSON(http.StatusInternalServerError, gin.H{errorKey: msgTryAgainLater})
			return
		}
		device, err := models.FindDeviceByDongleID(db, dongleID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{errorKey: msgTryAgainLater})
			return
		}
		conn, err := handler.wsUpgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			slog.Error("Failed to set websocket upgrade", errorKey, err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		conn.SetPongHandler(func(string) error {
			err := conn.SetReadDeadline(time.Now().Add(handler.pongWait))
			if err != nil {
				slog.Warn("Failed to set read deadline", errorKey, err)
			}
			err = models.UpdateAthenaPingTimestamp(db, device.ID)
			if err != nil {
				slog.Warn("Error updating athena ping timestamp", errorKey, err)
			}
			return nil
		})

		connHandler := &WSHandler{handler: handler.handler, conn: conn, pongWait: handler.pongWait, pingPeriod: handler.pingPeriod}
		connHandler.handle(c.Request.Context(), c.Request, &device, db, nats, metrics)
	}
}

// originAllowed reports whether origin matches one of the CORS hosts, which
// are origins such as https://example.com and may contain one * wildcard.
func originAllowed(origin string, hosts []string) bool {
	origin = strings.ToLower(origin)
	for _, host := range hosts {
		host = strings.ToLower(host)
		if strings.HasPrefix(host, "https://") {
			host = strings.TrimSuffix(host, ":443")
		}
		if strings.HasPrefix(host, "http://") {
			host = strings.TrimSuffix(host, ":80")
		}
		if prefix, suffix, wildcard := strings.Cut(host, "*"); wildcard {
			if len(origin) >= len(prefix)+len(suffix) && strings.HasPrefix(origin, prefix) && strings.HasSuffix(origin, suffix) {
				return true
			}
		} else if origin == host {
			return true
		}
	}
	return false
}

func (h *WSHandler) handle(parent context.Context, r *http.Request, device *models.Device, db *gorm.DB, nats *nats.Conn, metrics *metrics.Metrics) {
	defer func() { _ = h.conn.Close() }()

	err := h.conn.SetReadDeadline(time.Now().Add(h.pongWait))
	if err != nil {
		slog.Error("Failed to set read deadline", errorKey, err, "device_id", device.ID)
		return
	}
	err = h.conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(writeWait))
	if err != nil {
		slog.Error("Failed to send ping", errorKey, err, "device_id", device.ID)
		return
	}

	ctx, cancel := context.WithCancel(parent)
	defer cancel()
	writer := wsWriter{
		messages: make(chan Message, bufferSize),
		done:     ctx.Done(),
	}
	conn := h.handler.OnConnect(ctx, r, writer, device, db, nats, metrics, h.conn)

	writerDone := make(chan struct{})
	go func() {
		defer close(writerDone)
		ticker := time.NewTicker(h.pingPeriod)
		defer ticker.Stop()
		for {
			var err error
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				err = h.conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(writeWait))
			case msg := <-writer.messages:
				err = h.conn.WriteMessage(msg.Type, msg.Data)
			}
			if err != nil {
				// Closing the socket ends the read loop below.
				_ = h.conn.Close()
				return
			}
		}
	}()

	var handlers sync.WaitGroup
	for {
		t, msg, err := h.conn.ReadMessage()
		if err != nil {
			break
		}
		handlers.Go(func() { conn.OnMessage(ctx, msg, t) })
	}

	cancel()
	_ = h.conn.Close()
	<-writerDone
	waitFor(&handlers, handlerGrace, device)
	conn.OnDisconnect()
}

func waitFor(wg *sync.WaitGroup, timeout time.Duration, device *models.Device) {
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-done:
	case <-timer.C:
		slog.Warn("Message handlers still running after disconnect", "device_id", device.ID)
	}
}
