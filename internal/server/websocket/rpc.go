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

type dongle struct {
	bidiChannel    *bidiChannel
	channelWatcher *channelWatcher
}

type RPCWebsocket struct {
	websocket.Websocket
	connectedClients *xsync.Counter
	dongles          *xsync.MapOf[string, *dongle]
}

func CreateRPCWebsocket() *RPCWebsocket {
	return &RPCWebsocket{
		connectedClients: xsync.NewCounter(),
		dongles:          xsync.NewMapOf[string, *dongle](),
	}
}

type channelWatcher struct {
	ch          chan apimodels.RPCResponse
	subscribers *xsync.MapOf[string, func(apimodels.RPCResponse)]
}

func (cw *channelWatcher) WatchChannel() {
	for {
		response, more := <-cw.ch
		if !more {
			return
		}
		if response.ID == "" {
			continue
		}
		if subscriber, loaded := cw.subscribers.LoadAndDelete(response.ID); loaded {
			subscriber(response)
		}
	}
}

func (cw *channelWatcher) Subscribe(callID string, subscriber func(apimodels.RPCResponse)) {
	cw.subscribers.Store(callID, subscriber)
}

func (c *RPCWebsocket) Call(dongleID string, call apimodels.RPCCall) (apimodels.RPCResponse, error) {
	dongle, loaded := c.dongles.Load(dongleID)
	if !loaded {
		return apimodels.RPCResponse{}, ErrorNotConnected
	}

	if !dongle.bidiChannel.open {
		return apimodels.RPCResponse{}, ErrorNotConnected
	}

	responseChan := make(chan apimodels.RPCResponse)
	defer close(responseChan)
	dongle.channelWatcher.Subscribe(call.ID, func(response apimodels.RPCResponse) {
		responseChan <- response
	})

	dongle.bidiChannel.inbound <- call

	context, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	select {
	case <-context.Done():
		return apimodels.RPCResponse{}, fmt.Errorf("timeout")
	case resp := <-responseChan:
		return resp, nil
	}
}

func (c *RPCWebsocket) OnMessage(_ context.Context, _ *http.Request, _ websocket.Writer, msg []byte, msgType int, device *models.Device, db *gorm.DB) {
	jsonRPC := apimodels.RPCResponse{}
	err := json.Unmarshal(msg, &jsonRPC)
	if err != nil {
		slog.Warn("Error unmarshalling RPC response:", "error", err)
		return
	}

	dongle, loaded := c.dongles.Load(device.DongleID)
	if loaded && dongle.bidiChannel.open {
		dongle.bidiChannel.outbound <- jsonRPC
		return
	}
}

func (c *RPCWebsocket) OnConnect(ctx context.Context, _ *http.Request, w websocket.Writer, device *models.Device, db *gorm.DB) {
	bidi := bidiChannel{
		open:     true,
		inbound:  make(chan apimodels.RPCCall),
		outbound: make(chan apimodels.RPCResponse),
	}

	dongle := dongle{
		bidiChannel: &bidi,
		channelWatcher: &channelWatcher{
			ch:          bidi.outbound,
			subscribers: xsync.NewMapOf[string, func(apimodels.RPCResponse)](),
		},
	}
	go dongle.channelWatcher.WatchChannel()
	c.dongles.Store(device.DongleID, &dongle)
	c.connectedClients.Inc()

	slog.Info("RPC websocket connected", "device", device.DongleID)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case call, more := <-dongle.bidiChannel.inbound:
				if !more {
					return
				}
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
	dongle.bidiChannel.open = false
	close(dongle.bidiChannel.inbound)
	close(dongle.bidiChannel.outbound)
}
