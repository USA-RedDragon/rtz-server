package websocket

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
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

type Websocket interface {
	OnMessage(r *http.Request, w Writer, msg []byte, t int, device *models.Device, db *gorm.DB, metrics *metrics.Metrics)
	OnConnect(ctx context.Context, r *http.Request, w Writer, device *models.Device, db *gorm.DB, nats *nats.Conn, metrics *metrics.Metrics, conn *websocket.Conn)
	OnDisconnect(r *http.Request, device *models.Device, db *gorm.DB, metrics *metrics.Metrics)
}

type WSHandler struct {
	wsUpgrader websocket.Upgrader
	handler    Websocket
	conn       *websocket.Conn
}

func CreateHandler(ws Websocket, config *config.Config) func(*gin.Context) {
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
				origin = strings.ToLower(origin)
				for _, host := range config.HTTP.CORSHosts {
					host = strings.ToLower(host)
					if strings.HasSuffix(host, ":443") && strings.HasPrefix(origin, "https://") {
						host = strings.TrimSuffix(host, ":443")
					}
					if strings.HasSuffix(host, ":80") && strings.HasPrefix(origin, "http://") {
						host = strings.TrimSuffix(host, ":80")
					}
					if strings.Contains(origin, host) {
						return true
					}
				}
				return false
			},
			EnableCompression: true,
		},
		handler: ws,
	}

	return func(c *gin.Context) {
		dongleID, ok := c.Params.Get("dongle_id")
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "dongle_id is required"})
			return
		}
		maybeNats, ok := c.Get("nats")
		if !ok && config.NATS.Enabled {
			slog.Error("Failed to get NATS from context")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
			return
		}
		nats, ok := maybeNats.(*nats.Conn)
		if !ok {
			nats = nil
		}
		db, ok := c.MustGet("db").(*gorm.DB)
		if !ok {
			slog.Error("Failed to get db from context")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
			return
		}
		metrics, ok := c.MustGet("metrics").(*metrics.Metrics)
		if !ok {
			slog.Error("Failed to get metrics from context")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
			return
		}
		device, err := models.FindDeviceByDongleID(db, dongleID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
			return
		}
		conn, err := handler.wsUpgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			slog.Error("Failed to set websocket upgrade", "error", err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		handler.conn = conn
		handler.conn.SetPongHandler(func(string) error {
			err := handler.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
			if err != nil {
				slog.Warn("Failed to set read deadline", "error", err)
			}
			err = models.UpdateAthenaPingTimestamp(db, device.ID)
			if err != nil {
				slog.Warn("Error updating athena ping timestamp", "error", err)
			}
			return nil
		})

		handler.handle(c.Request.Context(), c.Request, &device, db, nats, metrics)
	}
}

func (h *WSHandler) handle(c context.Context, r *http.Request, device *models.Device, db *gorm.DB, nats *nats.Conn, metrics *metrics.Metrics) {
	defer func() {
		h.handler.OnDisconnect(r, device, db, metrics)
		_ = h.conn.Close()
	}()
	writer := wsWriter{
		writer: make(chan Message, bufferSize),
		error:  make(chan string),
	}

	h.handler.OnConnect(c, r, writer, device, db, nats, metrics, h.conn)

	go func() {
		for {
			select {
			case <-c.Done():
				return
			case <-writer.error:
				return
			case msg := <-writer.writer:
				err := h.conn.WriteMessage(msg.Type, msg.Data)
				if err != nil {
					return
				}
			}
		}
	}()

	err := h.conn.WriteMessage(websocket.PingMessage, []byte{})
	if err != nil {
		slog.Error("Failed to send ping", "error", err, "device_id", device.ID)
		return
	}

	for {
		t, msg, err := h.conn.ReadMessage()
		if err != nil {
			break
		}
		go h.handler.OnMessage(r, writer, msg, t, device, db, metrics)
	}
}
