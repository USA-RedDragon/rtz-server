package websocket

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/USA-RedDragon/rtz-server/internal/config"
	"github.com/USA-RedDragon/rtz-server/internal/db/models"
	"github.com/USA-RedDragon/rtz-server/internal/metrics"
	"github.com/USA-RedDragon/rtz-server/internal/server/apimodels"
	"github.com/USA-RedDragon/rtz-server/internal/websocket"
	gorillaWebsocket "github.com/gorilla/websocket"
	"github.com/nats-io/nats.go"
	"github.com/puzpuzpuz/xsync/v3"
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
	natsSub        *nats.Subscription
}

type RPCWebsocket struct {
	websocket.Websocket
	dongles *xsync.MapOf[string, *dongle]
	metrics *metrics.Metrics
	config  *config.Config
}

func CreateRPCWebsocket(ctx context.Context, config *config.Config, nats *nats.Conn, metrics *metrics.Metrics) *RPCWebsocket {
	socket := &RPCWebsocket{
		dongles: xsync.NewMapOf[string, *dongle](),
		metrics: metrics,
		config:  config,
	}
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
			closedChan := make(chan any)
			value.conn.SetCloseHandler(func(code int, text string) error {
				close(closedChan)
				return nil
			})

			// Close the socket
			err := value.conn.WriteControl(
				gorillaWebsocket.CloseMessage,
				gorillaWebsocket.FormatCloseMessage(gorillaWebsocket.CloseServiceRestart, "Server is restarting"),
				time.Now().Add(5*time.Second))
			if err != nil {
				slog.Warn("Error sending close message to websocket", "error", err)
				return err
			}
			ctx, cancel := context.WithTimeout(ctx, 6*time.Second)
			defer cancel()
			select {
			case <-ctx.Done():
				slog.Warn("Timeout waiting for websocket to close")
			case <-closedChan:
			}

			return nil
		})
		return true
	})

	return errGrp.Wait()
}

func (c *RPCWebsocket) Call(ctx context.Context, nats *nats.Conn, metrics *metrics.Metrics, dongleID string, call apimodels.RPCCall) (apimodels.RPCResponse, error) {
	dongle, loaded := c.dongles.Load(dongleID)
	if !loaded {
		// Dongle is not here, send to NATS if enabled
		if nats != nil {
			msg, err := json.Marshal(call)
			if err != nil {
				return apimodels.RPCResponse{}, err
			}

			resp, err := nats.Request("rpc:call:"+dongleID, msg, 5*time.Second)
			if err != nil {
				metrics.IncrementAthenaErrors(dongleID, "rpc_call_nats_request")
				slog.Warn("Error sending RPC to NATS", "error", err)
				return apimodels.RPCResponse{}, err
			}

			var rpcResp apimodels.RPCResponse
			err = json.Unmarshal(resp.Data, &rpcResp)
			if err != nil {
				metrics.IncrementAthenaErrors(dongleID, "rpc_call_nats_unmarshal")
				slog.Warn("Error unmarshalling RPC response", "error", err)
				return apimodels.RPCResponse{}, err
			}

			return rpcResp, nil
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

func (c *RPCWebsocket) OnMessage(ctx context.Context, _ *http.Request, _ websocket.Writer, msg []byte, msgType int, device *models.Device, _ *gorm.DB, nats *nats.Conn, metrics *metrics.Metrics) {
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
				slog.Debug("RPC: forwardLogs", "device", device.DongleID, "logs", jsonRPC.Params)
			case "storeStats":
				slog.Debug("RPC: storeStats", "device", device.DongleID, "stats", jsonRPC.Params)
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

func (c *RPCWebsocket) OnConnect(ctx context.Context, _ *http.Request, w websocket.Writer, device *models.Device, _ *gorm.DB, nc *nats.Conn, metrics *metrics.Metrics, conn *gorillaWebsocket.Conn) {
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
	if c.config.NATS.Enabled {
		sub, err := nc.Subscribe("rpc:call:"+device.DongleID, func(msg *nats.Msg) {
			var call apimodels.RPCCall
			err := json.Unmarshal([]byte(msg.Data), &call)
			if err != nil {
				metrics.IncrementAthenaErrors(device.DongleID, "unmarshal_nats_rpc_call")
				slog.Warn("Error unmarshalling RPC call", "error", err)
			}

			responseChan := make(chan apimodels.RPCResponse)
			defer close(responseChan)
			dongle.channelWatcher.Subscribe(call.ID, func(response apimodels.RPCResponse) {
				responseChan <- response
			})

			if dongle.bidiChannel.open {
				dongle.bidiChannel.inbound <- call
			}

			context, cancel := context.WithTimeout(context.Background(), 120*time.Second)
			defer cancel()
			select {
			case <-context.Done():
				metrics.IncrementAthenaErrors(device.DongleID, "rpc_call_timeout")
				err := msg.NakWithDelay(2 * time.Second)
				if err != nil {
					metrics.IncrementAthenaErrors(device.DongleID, "nats_rpc_nak")
					slog.Warn("Error sending NAK to NATS", "error", err)
				}
			case resp := <-responseChan:
				jsonData, err := json.Marshal(resp)
				if err != nil {
					metrics.IncrementAthenaErrors(device.DongleID, "marshal_rpc_response")
					slog.Warn("Error marshalling response data:", "error", err)
					msg.NakWithDelay(2 * time.Second)
				}
				err = msg.Respond(jsonData)
				if err != nil {
					metrics.IncrementAthenaErrors(device.DongleID, "nats_rpc_respond")
					slog.Warn("Error responding to NATS", "error", err)
				}
			}
		})
		if err != nil {
			metrics.IncrementAthenaErrors(device.DongleID, "nats_rpc_subscribe")
			slog.Warn("Error subscribing to NATS", "error", err)
			return
		}
		dongle.natsSub = sub
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

func (c *RPCWebsocket) OnDisconnect(ctx context.Context, _ *http.Request, device *models.Device, _ *gorm.DB, metrics *metrics.Metrics) {
	metrics.DecrementAthenaConnections(device.DongleID)
	dongle, loaded := c.dongles.LoadAndDelete(device.DongleID)
	if !loaded {
		return
	}
	if c.config.NATS.Enabled {
		err := dongle.natsSub.Unsubscribe()
		if err != nil {
			slog.Warn("Error unsubscribing from NATS", "error", err)
		}
	}
	dongle.bidiChannel.open = false
	close(dongle.bidiChannel.inbound)
	close(dongle.bidiChannel.outbound)
}
