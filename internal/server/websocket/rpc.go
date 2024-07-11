package websocket

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/USA-RedDragon/rtz-server/internal/db/models"
	"github.com/USA-RedDragon/rtz-server/internal/server/apimodels"
	"github.com/USA-RedDragon/rtz-server/internal/websocket"
	gorillaWebsocket "github.com/gorilla/websocket"
	"github.com/puzpuzpuz/xsync/v3"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

var (
	ErrNotConnected = errors.New("dongle not connected")
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

func (d *dongle) watchRedis(ctx context.Context, redis *redis.Client, device *models.Device) {
	if redis == nil {
		return
	}
	slog.Info("Watching redis for RPC calls", "dongle", device.DongleID)
	sub := redis.Subscribe(ctx, "rpc:call:"+device.DongleID)
	defer sub.Close()
	subChan := sub.Channel()
	checkOpen := time.NewTicker(1 * time.Second)
	for {
		select {
		case <-ctx.Done():
			slog.Info("watchRedis Context done")
			return
		case <-checkOpen.C:
			if !d.bidiChannel.open {
				slog.Info("Dongle not connected, stopping redis watch")
				return
			}
		case msg := <-subChan:
			slog.Info("Received RPC call from redis", "key", msg.Channel)
			var call apimodels.RPCCall
			err := json.Unmarshal([]byte(msg.Payload), &call)
			if err != nil {
				slog.Warn("Error unmarshalling RPC call", "error", err)
				continue
			}
			d.bidiChannel.inbound <- call
		}
	}
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

func (c *RPCWebsocket) Call(ctx context.Context, redis *redis.Client, dongleID string, call apimodels.RPCCall) (apimodels.RPCResponse, error) {
	dongle, loaded := c.dongles.Load(dongleID)
	if !loaded {
		// Dongle is not here, send to redis if enabled
		if redis != nil {
			msg, err := json.Marshal(call)
			if err != nil {
				return apimodels.RPCResponse{}, err
			}
			slog.Info("Reading RPC response to redis", "key", "rpc:response:"+dongleID+":"+call.ID)
			sub := redis.Subscribe(ctx, "rpc:response:"+dongleID+":"+call.ID)
			defer sub.Close()
			respChannel := make(chan apimodels.RPCResponse)
			go func() {
				subChan := sub.Channel()
				ctx, cancel := context.WithTimeout(ctx, 120*time.Second)
				defer cancel()
				for {
					select {
					case <-ctx.Done():
						return
					case msg := <-subChan:
						slog.Info("Received RPC response from redis", "key", "rpc:response:"+dongleID+":"+call.ID)
						var response apimodels.RPCResponse
						err := json.Unmarshal([]byte(msg.Payload), &response)
						if err != nil {
							slog.Warn("Error unmarshalling RPC response", "error", err)
							continue
						}
						respChannel <- response
						return
					}
				}
			}()

			slog.Info("Sending RPC to redis", "key", "rpc:call:"+dongleID)
			err = redis.Publish(ctx, "rpc:call:"+dongleID, msg).Err()
			if err != nil {
				slog.Warn("Error sending RPC to redis", "error", err)
			}

			ctx, cancel := context.WithTimeout(ctx, 120*time.Second)
			defer cancel()
			select {
			case <-ctx.Done():
				return apimodels.RPCResponse{}, fmt.Errorf("timeout")
			case resp := <-respChannel:
				return resp, nil
			}
		}
		return apimodels.RPCResponse{}, ErrNotConnected
	}

	if !dongle.bidiChannel.open {
		return apimodels.RPCResponse{}, ErrNotConnected
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

func (c *RPCWebsocket) OnMessage(ctx context.Context, _ *http.Request, _ websocket.Writer, msg []byte, msgType int, device *models.Device, _ *gorm.DB, redis *redis.Client) {
	var rawJSON map[string]interface{}
	err := json.Unmarshal(msg, &rawJSON)
	if err != nil {
		slog.Warn("Error unmarshalling JSON:", "error", err)
		return
	}
	dongle, loaded := c.dongles.Load(device.DongleID)
	if !loaded {
		if redis != nil {
			if _, ok := rawJSON["result"]; ok {
				// This is a response
				maybeID, ok := rawJSON["id"]
				if !ok {
					slog.Warn("Invalid response ID")
					return
				}
				id, ok := maybeID.(string)
				if !ok {
					slog.Warn("Invalid response ID")
					return
				}
				slog.Info("Sending response to redis", "key", "rpc:response:"+device.DongleID+":"+id)
				err := redis.Publish(ctx, "rpc:response:"+device.DongleID+":"+id, msg).Err()
				if err != nil {
					slog.Warn("Error sending RPC to redis", "error", err)
				}
				return
			}
		}
		slog.Warn("Dongle not connected", "dongle", device.DongleID)
		return
	}
	if _, ok := rawJSON["method"]; ok {
		// This is a call
		jsonRPC := apimodels.RPCCall{}
		err := json.Unmarshal(msg, &jsonRPC)
		if err != nil {
			slog.Warn("Error unmarshalling RPC call:", "error", err)
			return
		}

		go func() {
			switch jsonRPC.Method {
			case "forwardLogs":
			case "storeStats":
			default:
				slog.Warn("Unknown RPC method", "method", jsonRPC.Method)
				slog.Info("Message", "type", msgType, "msg", msg)
				return
			}
			if dongle.bidiChannel.open {
				dongle.bidiChannel.outbound <- apimodels.RPCResponse{
					ID:             jsonRPC.ID,
					JSONRPCVersion: jsonRPC.JSONRPCVersion,
					Result: map[string]bool{
						"success": true,
					},
				}
			}
		}()
	} else if _, ok := rawJSON["result"]; ok {
		// This is a response
		jsonRPC := apimodels.RPCResponse{}
		err := json.Unmarshal(msg, &jsonRPC)
		if err != nil {
			slog.Warn("Error unmarshalling RPC call:", "error", err)
			return
		}

		if dongle.bidiChannel.open {
			dongle.bidiChannel.outbound <- jsonRPC
			return
		}
	} else {
		slog.Warn("Unknown message type")
		slog.Info("Message", "type", msgType, "msg", msg)
		return
	}
}

func (c *RPCWebsocket) OnConnect(ctx context.Context, _ *http.Request, w websocket.Writer, device *models.Device, _ *gorm.DB, redis *redis.Client) {
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
	go dongle.watchRedis(ctx, redis, device)
	c.dongles.Store(device.DongleID, &dongle)
	c.connectedClients.Inc()

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

func (c *RPCWebsocket) OnDisconnect(_ context.Context, _ *http.Request, device *models.Device, _ *gorm.DB) {
	c.connectedClients.Dec()
	dongle, loaded := c.dongles.LoadAndDelete(device.DongleID)
	if !loaded {
		return
	}
	dongle.bidiChannel.open = false
	close(dongle.bidiChannel.inbound)
	close(dongle.bidiChannel.outbound)
}
