package websocket

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/USA-RedDragon/rtz-server/internal/db/models"
	"github.com/USA-RedDragon/rtz-server/internal/metrics"
	"github.com/USA-RedDragon/rtz-server/internal/server/apimodels"
	"github.com/USA-RedDragon/rtz-server/internal/utils"
	"github.com/USA-RedDragon/rtz-server/internal/websocket"
	gorillaWebsocket "github.com/gorilla/websocket"
	"github.com/nats-io/nats.go"
	"gorm.io/gorm"
)

func (c *RPCWebsocket) OnMessage(_ *http.Request, _ websocket.Writer, msg []byte, msgType int, device *models.Device, _ *gorm.DB, metrics *metrics.Metrics) {
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
				dongle.bidiChannel.outbound <- apimodels.RPCResponse{
					ID:             jsonRPC.ID,
					JSONRPCVersion: jsonRPC.JSONRPCVersion,
					Result: map[string]bool{
						"success": true,
					},
				}
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
		bidiChannel:    &bidi,
		channelWatcher: utils.NewChannelWatcher(bidi.outbound),
		conn:           conn,
	}
	go dongle.channelWatcher.WatchChannel(func(resp apimodels.RPCResponse) string {
		return resp.ID
	})
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
	c.dongles.Store(device.DongleID, &dongle)
	metrics.IncrementAthenaConnections(device.DongleID)
	if c.config.NATS.Enabled {
		sub, err := nc.Subscribe("rpc:call:"+device.DongleID, func(msg *nats.Msg) {
			var call apimodels.RPCCall
			err := json.Unmarshal(msg.Data, &call)
			if err != nil {
				metrics.IncrementAthenaErrors(device.DongleID, "unmarshal_nats_rpc_call")
				slog.Warn("Error unmarshalling RPC call", "error", err)
				err := msg.Respond([]byte{})
				if err != nil {
					metrics.IncrementAthenaErrors(device.DongleID, "nats_rpc_nak")
					slog.Warn("Error sending NAK to NATS", "error", err)
				}
				return
			}

			responseChan := make(chan apimodels.RPCResponse)
			defer close(responseChan)
			dongle.channelWatcher.Subscribe(call.ID, func(response apimodels.RPCResponse) {
				responseChan <- response
			})

			if !dongle.bidiChannel.open {
				err := msg.Respond([]byte{})
				if err != nil {
					metrics.IncrementAthenaErrors(device.DongleID, "nats_rpc_nak")
					slog.Warn("Error sending NAK to NATS", "error", err)
				}
				return
			}

			dongle.bidiChannel.inbound <- call

			context, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()
			select {
			case <-context.Done():
				metrics.IncrementAthenaErrors(device.DongleID, "rpc_call_timeout")
				err := msg.Respond([]byte{})
				if err != nil {
					metrics.IncrementAthenaErrors(device.DongleID, "nats_rpc_nak")
					slog.Warn("Error sending NAK to NATS", "error", err)
				}
			case resp := <-responseChan:
				jsonData, err := json.Marshal(resp)
				if err != nil {
					metrics.IncrementAthenaErrors(device.DongleID, "marshal_rpc_response")
					slog.Warn("Error marshalling response data:", "error", err)
					err := msg.Respond([]byte{})
					if err != nil {
						metrics.IncrementAthenaErrors(device.DongleID, "nats_rpc_nak")
						slog.Warn("Error sending NAK to NATS", "error", err)
					}
					return
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
}

func (c *RPCWebsocket) OnDisconnect(_ *http.Request, device *models.Device, _ *gorm.DB, metrics *metrics.Metrics) {
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
