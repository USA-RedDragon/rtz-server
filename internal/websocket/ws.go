package websocket

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/USA-RedDragon/connect-server/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

const bufferSize = 1024

type Websocket interface {
	OnMessage(ctx context.Context, r *http.Request, w Writer, msg []byte, t int)
	OnConnect(ctx context.Context, r *http.Request, w Writer)
	OnDisconnect(ctx context.Context, r *http.Request)
}

type WSHandler struct {
	wsUpgrader websocket.Upgrader
	handler    Websocket
	conn       *websocket.Conn
}

func CreateHandler(ws Websocket, config *config.HTTP) func(*gin.Context) {
	handler := &WSHandler{
		wsUpgrader: websocket.Upgrader{
			HandshakeTimeout: 0,
			ReadBufferSize:   bufferSize,
			WriteBufferSize:  bufferSize,
			WriteBufferPool:  nil,
			Subprotocols:     []string{},
			Error: func(w http.ResponseWriter, r *http.Request, status int, reason error) {
			},
			CheckOrigin: func(r *http.Request) bool {
				origin := r.Header.Get("Origin")
				if origin == "" {
					return true
				}
				origin = strings.ToLower(origin)
				for _, host := range config.CORSHosts {
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
		conn, err := handler.wsUpgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			slog.Error("Failed to set websocket upgrade", "error", err)
			return
		}
		handler.conn = conn

		defer func() {
			handler.handler.OnDisconnect(c, c.Request)
			_ = handler.conn.Close()
		}()

		handler.handle(c.Request.Context(), c.Request)
	}
}

func (h *WSHandler) handle(c context.Context, r *http.Request) {
	writer := wsWriter{
		writer: make(chan Message, bufferSize),
		error:  make(chan string),
	}
	h.handler.OnConnect(c, r, writer)

	go func() {
		for {
			t, msg, err := h.conn.ReadMessage()
			if err != nil {
				writer.Error("read failed")
				break
			}
			switch {
			case t == websocket.PingMessage:
				writer.WriteMessage(Message{
					Type: websocket.PongMessage,
				})
			case strings.EqualFold(string(msg), "ping"):
				writer.WriteMessage(Message{
					Type: websocket.TextMessage,
					Data: []byte("PONG"),
				})
			default:
				h.handler.OnMessage(c, r, writer, msg, t)
			}
		}
	}()

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
}
