package websocket

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/USA-RedDragon/connect-server/internal/db/models"
	"github.com/USA-RedDragon/connect-server/internal/server/apimodels"
	"github.com/USA-RedDragon/connect-server/internal/websocket"
	gorillaWebsocket "github.com/gorilla/websocket"
	"github.com/puzpuzpuz/xsync/v3"
	"gorm.io/gorm"
)

var (
	ErrorNotConnected = errors.New("dongle not connected")
)

type bidiChannel struct {
	open     bool
	inbound  chan apimodels.RPCCall
	outbound chan apimodels.RPCResponse
}

type RPCWebsocket struct {
	websocket.Websocket
	connectedClients *xsync.Counter
	dongles          *xsync.MapOf[string, *bidiChannel]
}

func CreateRPCWebsocket() *RPCWebsocket {
	return &RPCWebsocket{
		connectedClients: xsync.NewCounter(),
		dongles:          xsync.NewMapOf[string, *bidiChannel](),
	}
}

func (c *RPCWebsocket) Call(dongleID string, call apimodels.RPCCall) (apimodels.RPCResponse, error) {
	dongle, loaded := c.dongles.Load(dongleID)
	if !loaded {
		return apimodels.RPCResponse{}, ErrorNotConnected
	}

	responseChan := make(chan apimodels.RPCResponse)
	defer close(responseChan)
	if !dongle.open {
		return apimodels.RPCResponse{}, ErrorNotConnected
	}
	dongle.inbound <- call
	go func() {
		resp, err := waitForResponse(call.ID, dongle.outbound, 120*time.Second)
		if err != nil {
			slog.Warn("Error waiting for response", "error", err)
			return
		}
		responseChan <- resp
	}()
	return <-responseChan, nil
}

func waitForResponse(callID string, ch chan apimodels.RPCResponse, timeout time.Duration) (apimodels.RPCResponse, error) {
	context, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	for {
		select {
		case resp := <-ch:
			if resp.ID == callID {
				return resp, nil
			}
			ch <- resp
		case <-context.Done():
			return apimodels.RPCResponse{}, fmt.Errorf("timeout")
		}
	}
}

func (c *RPCWebsocket) OnMessage(_ context.Context, _ *http.Request, _ websocket.Writer, msg []byte, msgType int, device *models.Device, db *gorm.DB) {
	err := models.UpdateAthenaPingTimestamp(db, device.ID)
	if err != nil {
		slog.Warn("Error updating athena ping timestamp", "error", err)
	}

	jsonRPC := apimodels.RPCResponse{}
	err = json.Unmarshal(msg, &jsonRPC)
	if err != nil {
		slog.Warn("Error unmarshalling RPC response:", "error", err)
		return
	}

	dongle, loaded := c.dongles.Load(device.DongleID)
	if loaded && dongle.open {
		dongle.outbound <- jsonRPC
		return
	}
}

func (c *RPCWebsocket) OnConnect(ctx context.Context, _ *http.Request, w websocket.Writer, device *models.Device, db *gorm.DB) {
	dongle := bidiChannel{
		open:     true,
		inbound:  make(chan apimodels.RPCCall),
		outbound: make(chan apimodels.RPCResponse),
	}
	c.dongles.Store(device.DongleID, &dongle)
	c.connectedClients.Inc()

	slog.Info("RPC websocket connected", "device", device.DongleID)

	err := models.UpdateAthenaPingTimestamp(db, device.ID)
	if err != nil {
		slog.Warn("Error updating athena ping timestamp", "error", err)
	}

	w.WriteMessage(websocket.Message{
		Type: gorillaWebsocket.PingMessage,
		Data: []byte{},
	})

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case call := <-dongle.inbound:
				// Received a call from the site
				jsonData, err := json.Marshal(call)
				if err != nil {
					slog.Warn("Error marshalling call data:", "error", err)
					continue
				}
				w.WriteMessage(websocket.Message{
					Type: gorillaWebsocket.TextMessage,
					Data: jsonData,
				})
			}
		}
	}()
}

func (c *RPCWebsocket) OnDisconnect(ctx context.Context, _ *http.Request, device *models.Device, db *gorm.DB) {
	c.connectedClients.Dec()
	slog.Info("RPC websocket disconnected", "device", device.DongleID)
	dongle, loaded := c.dongles.LoadAndDelete(device.DongleID)
	if !loaded {
		return
	}
	dongle.open = false
	close(dongle.inbound)
	close(dongle.outbound)
}
