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
	"github.com/USA-RedDragon/rtz-server/internal/metrics"
	"github.com/USA-RedDragon/rtz-server/internal/server/apimodels"
	"github.com/USA-RedDragon/rtz-server/internal/websocket"
	gorillaWebsocket "github.com/gorilla/websocket"
	"github.com/puzpuzpuz/xsync/v3"
	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/errgroup"
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
	conn           *gorillaWebsocket.Conn
}

type RPCWebsocket struct {
	websocket.Websocket
	dongles *xsync.MapOf[string, *dongle]
	metrics *metrics.Metrics
}

func CreateRPCWebsocket(ctx context.Context, redis *redis.Client, metrics *metrics.Metrics) *RPCWebsocket {
	socket := &RPCWebsocket{
		dongles: xsync.NewMapOf[string, *dongle](),
		metrics: metrics,
	}
	go socket.watchRedis(ctx, redis, metrics)
	return socket
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

func (c *RPCWebsocket) Stop(ctx context.Context) error {
	errGrp := errgroup.Group{}

	c.dongles.Range(func(key string, value *dongle) bool {
		errGrp.Go(func() error {
			c.metrics.DecrementAthenaConnections(key)
			// Close the socket
			err := value.conn.WriteControl(
				gorillaWebsocket.CloseMessage,
				gorillaWebsocket.FormatCloseMessage(gorillaWebsocket.CloseServiceRestart, "Server is restarting"),
				time.Now().Add(5*time.Second))
			if err != nil {
				slog.Warn("Error sending close message to websocket", "error", err)
				return err
			}

			value.bidiChannel.open = false
			close(value.bidiChannel.inbound)
			close(value.bidiChannel.outbound)
			return value.conn.Close()
		})
		return true
	})

	return errGrp.Wait()
}

func (c *RPCWebsocket) watchRedis(ctx context.Context, redis *redis.Client, metrics *metrics.Metrics) {
	if redis == nil {
		return
	}
	sub := redis.Subscribe(ctx, "rpc:call:*")
	defer sub.Close()
	subChan := sub.Channel()
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-subChan:
			callChannel := msg.Channel
			deviceID := callChannel[len("rpc:call:"):]
			device, loaded := c.dongles.Load(deviceID)
			if !loaded {
				// Push it back onto the channel
				redis.Publish(ctx, callChannel, msg.Payload)
				continue
			}
			var call apimodels.RPCCall
			err := json.Unmarshal([]byte(msg.Payload), &call)
			if err != nil {
				metrics.IncrementAthenaErrors(deviceID, "unmarshal_redis_rpc_call")
				slog.Warn("Error unmarshalling RPC call", "error", err)
				continue
			}
			if device.bidiChannel.open {
				device.bidiChannel.inbound <- call
			}
		}
	}
}

func (c *RPCWebsocket) Call(ctx context.Context, redis *redis.Client, metrics *metrics.Metrics, dongleID string, call apimodels.RPCCall) (apimodels.RPCResponse, error) {
	dongle, loaded := c.dongles.Load(dongleID)
	if !loaded {
		// Dongle is not here, send to redis if enabled
		if redis != nil {
			msg, err := json.Marshal(call)
			if err != nil {
				return apimodels.RPCResponse{}, err
			}
			sub := redis.Subscribe(ctx, "rpc:response:"+dongleID+":"+call.ID)
			defer sub.Close()
			respChannel := make(chan apimodels.RPCResponse)
			subChan := sub.Channel()
			tCtx, cancel := context.WithTimeout(ctx, 120*time.Second)
			defer cancel()
			go func() {
				for {
					select {
					case <-tCtx.Done():
						return
					case msg := <-subChan:
						var response apimodels.RPCResponse
						err := json.Unmarshal([]byte(msg.Payload), &response)
						if err != nil {
							metrics.IncrementAthenaErrors(dongleID, "unmarshal_redis_rpc_response")
							slog.Warn("Error unmarshalling RPC response", "error", err)
							continue
						}
						respChannel <- response
						return
					}
				}
			}()

			ctx, cancel = context.WithTimeout(ctx, 120*time.Second)
			defer cancel()

			err = redis.Publish(ctx, "rpc:call:"+dongleID, msg).Err()
			if err != nil {
				metrics.IncrementAthenaErrors(dongleID, "rpc_call_redis_publish")
				slog.Warn("Error sending RPC to redis", "error", err)
			}

			select {
			case <-ctx.Done():
				metrics.IncrementAthenaErrors(dongleID, "rpc_call_redis_timeout")
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
		metrics.IncrementAthenaErrors(dongleID, "rpc_call_timeout")
		return apimodels.RPCResponse{}, fmt.Errorf("timeout")
	case resp := <-responseChan:
		return resp, nil
	}
}

func (c *RPCWebsocket) OnMessage(ctx context.Context, _ *http.Request, _ websocket.Writer, msg []byte, msgType int, device *models.Device, _ *gorm.DB, redis *redis.Client, metrics *metrics.Metrics) {
	var rawJSON map[string]interface{}
	err := json.Unmarshal(msg, &rawJSON)
	if err != nil {
		metrics.IncrementAthenaErrors(device.DongleID, "unmarshal_rpc_json")
		slog.Warn("Error unmarshalling JSON:", "error", err)
		return
	}
	dongle, loaded := c.dongles.Load(device.DongleID)
	if !loaded {
		slog.Warn("Dongle not connected", "dongle", device.DongleID)
		return
	}
	if _, ok := rawJSON["method"]; ok {
		// This is a call
		jsonRPC := apimodels.RPCCall{}
		err := json.Unmarshal(msg, &jsonRPC)
		if err != nil {
			metrics.IncrementAthenaErrors(device.DongleID, "unmarshal_rpc_call")
			slog.Warn("Error unmarshalling RPC call:", "error", err)
			return
		}

		go func() {
			switch jsonRPC.Method {
			case "forwardLogs":
			case "storeStats":
			default:
				metrics.IncrementAthenaErrors(device.DongleID, "unknown_rpc_method")
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
		if redis != nil {
			// This is a response
			maybeID, ok := rawJSON["id"]
			if !ok {
				metrics.IncrementAthenaErrors(device.DongleID, "rpc_response_bad_id")
				slog.Warn("Invalid response ID")
				return
			}
			id, ok := maybeID.(string)
			if !ok {
				metrics.IncrementAthenaErrors(device.DongleID, "rpc_response_bad_id_cast")
				slog.Warn("Invalid response ID")
				return
			}
			err := redis.Publish(ctx, "rpc:response:"+device.DongleID+":"+id, msg).Err()
			if err != nil {
				metrics.IncrementAthenaErrors(device.DongleID, "rpc_response_redis_publish")
				slog.Warn("Error sending RPC to redis", "error", err)
			}
		}

		jsonRPC := apimodels.RPCResponse{}
		err := json.Unmarshal(msg, &jsonRPC)
		if err != nil {
			metrics.IncrementAthenaErrors(device.DongleID, "unmarshal_rpc_response")
			slog.Warn("Error unmarshalling RPC call:", "error", err)
			return
		}

		if dongle.bidiChannel.open {
			dongle.bidiChannel.outbound <- jsonRPC
			return
		}
	} else {
		metrics.IncrementAthenaErrors(device.DongleID, "unknown_rpc_message_type")
		slog.Warn("Unknown message type")
		slog.Info("Message", "type", msgType, "msg", msg)
		return
	}
}

func (c *RPCWebsocket) OnConnect(ctx context.Context, _ *http.Request, w websocket.Writer, device *models.Device, _ *gorm.DB, redis *redis.Client, metrics *metrics.Metrics, conn *gorillaWebsocket.Conn) {
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
		conn: conn,
	}
	go dongle.channelWatcher.WatchChannel()
	c.dongles.Store(device.DongleID, &dongle)
	metrics.IncrementAthenaConnections(device.DongleID)

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
					metrics.IncrementAthenaErrors(device.DongleID, "marshal_rpc_call")
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

func (c *RPCWebsocket) OnDisconnect(_ context.Context, _ *http.Request, device *models.Device, _ *gorm.DB, metrics *metrics.Metrics) {
	metrics.DecrementAthenaConnections(device.DongleID)
	dongle, loaded := c.dongles.LoadAndDelete(device.DongleID)
	if !loaded {
		return
	}
	dongle.bidiChannel.open = false
	close(dongle.bidiChannel.inbound)
	close(dongle.bidiChannel.outbound)
}
