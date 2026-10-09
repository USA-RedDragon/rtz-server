package websocket

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/USA-RedDragon/rtz-server/internal/db/models"
	"github.com/USA-RedDragon/rtz-server/internal/metrics"
	"github.com/USA-RedDragon/rtz-server/internal/server/apimodels"
	"github.com/USA-RedDragon/rtz-server/internal/websocket"
	gorillaWebsocket "github.com/gorilla/websocket"
	"github.com/nats-io/nats.go"
	"github.com/puzpuzpuz/xsync/v3"
	"gorm.io/gorm"
)

func (c *RPCWebsocket) OnConnect(ctx context.Context, _ *http.Request, w websocket.Writer, device *models.Device, _ *gorm.DB, nc *nats.Conn, metrics *metrics.Metrics, conn *gorillaWebsocket.Conn) websocket.Connection {
	d := &dongle{
		rpc:     c,
		device:  device,
		metrics: metrics,
		conn:    conn,
		writer:  w,
		done:    ctx.Done(),
		pending: xsync.NewMapOf[string, chan apimodels.RPCResponse](),
	}
	c.dongles.Store(device.DongleID, d)
	metrics.IncrementAthenaConnections(device.DongleID)
	if c.config.NATS.Enabled {
		sub, err := nc.Subscribe("rpc:call:"+device.DongleID, func(msg *nats.Msg) {
			d.handleNATSCall(ctx, msg)
		})
		if err != nil {
			metrics.IncrementAthenaErrors(device.DongleID, "nats_rpc_subscribe")
			slog.Warn("Error subscribing to NATS", "error", err)
			return d
		}
		d.natsSub = sub
	}
	return d
}

func (d *dongle) OnMessage(ctx context.Context, msg []byte, msgType int) {
	var rawJSON map[string]interface{}
	err := json.Unmarshal(msg, &rawJSON)
	if err != nil {
		d.metrics.IncrementAthenaErrors(d.device.DongleID, "unmarshal_rpc_json")
		slog.Warn("Error unmarshalling JSON:", "error", err)
		return
	}
	if _, ok := rawJSON["method"]; ok {
		d.handleDeviceCall(ctx, msg, msgType)
	} else if _, ok := rawJSON["result"]; ok {
		jsonRPC := apimodels.RPCResponse{}
		err := json.Unmarshal(msg, &jsonRPC)
		if err != nil {
			d.metrics.IncrementAthenaErrors(d.device.DongleID, "unmarshal_rpc_response")
			slog.Warn("Error unmarshalling RPC call:", "error", err)
			return
		}
		if waiter, ok := d.pending.LoadAndDelete(jsonRPC.ID); ok {
			waiter <- jsonRPC
		}
	} else {
		d.metrics.IncrementAthenaErrors(d.device.DongleID, "unknown_rpc_message_type")
		slog.Warn("Unknown message type")
		slog.Info("Message", "type", msgType, "msg", msg)
	}
}

// handleDeviceCall answers a call the device made to the server.
func (d *dongle) handleDeviceCall(ctx context.Context, msg []byte, msgType int) {
	jsonRPC := apimodels.RPCCall{}
	err := json.Unmarshal(msg, &jsonRPC)
	if err != nil {
		d.metrics.IncrementAthenaErrors(d.device.DongleID, "unmarshal_rpc_call")
		slog.Warn("Error unmarshalling RPC call:", "error", err)
		return
	}

	switch jsonRPC.Method {
	case "forwardLogs":
		slog.Debug("RPC: forwardLogs", "device", d.device.DongleID, "logs", jsonRPC.Params)
	case "storeStats":
		slog.Debug("RPC: storeStats", "device", d.device.DongleID, "stats", jsonRPC.Params)
	default:
		d.metrics.IncrementAthenaErrors(d.device.DongleID, "unknown_rpc_method")
		slog.Warn("Unknown RPC method", "method", jsonRPC.Method)
		slog.Info("Message", "type", msgType, "msg", msg)
		return
	}
	reply, err := json.Marshal(apimodels.RPCResponse{
		ID:             jsonRPC.ID,
		JSONRPCVersion: jsonRPC.JSONRPCVersion,
		Result: map[string]bool{
			"success": true,
		},
	})
	if err != nil {
		d.metrics.IncrementAthenaErrors(d.device.DongleID, "marshal_rpc_response")
		slog.Warn("Error marshalling RPC response", "error", err)
		return
	}
	err = d.writer.WriteMessage(ctx, websocket.Message{Type: gorillaWebsocket.TextMessage, Data: reply})
	if err != nil {
		slog.Debug("Dropped RPC response to a closed connection", "device", d.device.DongleID, "error", err)
	}
}

// call sends call to the device and waits for its response, until ctx is done
// or the connection ends.
func (d *dongle) call(ctx context.Context, call apimodels.RPCCall) (apimodels.RPCResponse, error) {
	data, err := json.Marshal(call)
	if err != nil {
		d.metrics.IncrementAthenaErrors(d.device.DongleID, "marshal_rpc_call")
		return apimodels.RPCResponse{}, err
	}

	response := make(chan apimodels.RPCResponse, 1)
	d.pending.Store(call.ID, response)
	defer d.pending.Delete(call.ID)

	err = d.writer.WriteMessage(ctx, websocket.Message{Type: gorillaWebsocket.TextMessage, Data: data})
	if errors.Is(err, websocket.ErrConnectionClosed) {
		return apimodels.RPCResponse{}, ErrNotConnected
	} else if err != nil {
		return apimodels.RPCResponse{}, err
	}

	select {
	case resp := <-response:
		return resp, nil
	case <-d.done:
		return apimodels.RPCResponse{}, ErrNotConnected
	case <-ctx.Done():
		return apimodels.RPCResponse{}, ctx.Err()
	}
}

// handleNATSCall forwards a call another server received for this device.
func (d *dongle) handleNATSCall(ctx context.Context, msg *nats.Msg) {
	nak := func() {
		if err := msg.Respond([]byte{}); err != nil {
			d.metrics.IncrementAthenaErrors(d.device.DongleID, "nats_rpc_nak")
			slog.Warn("Error sending NAK to NATS", "error", err)
		}
	}

	var call apimodels.RPCCall
	err := json.Unmarshal(msg.Data, &call)
	if err != nil {
		d.metrics.IncrementAthenaErrors(d.device.DongleID, "unmarshal_nats_rpc_call")
		slog.Warn("Error unmarshalling RPC call", "error", err)
		nak()
		return
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	resp, err := d.call(ctx, call)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			d.metrics.IncrementAthenaErrors(d.device.DongleID, "rpc_call_timeout")
		}
		nak()
		return
	}
	jsonData, err := json.Marshal(resp)
	if err != nil {
		d.metrics.IncrementAthenaErrors(d.device.DongleID, "marshal_rpc_response")
		slog.Warn("Error marshalling response data:", "error", err)
		nak()
		return
	}
	if err := msg.Respond(jsonData); err != nil {
		d.metrics.IncrementAthenaErrors(d.device.DongleID, "nats_rpc_respond")
		slog.Warn("Error responding to NATS", "error", err)
	}
}

func (d *dongle) OnDisconnect() {
	d.metrics.DecrementAthenaConnections(d.device.DongleID)
	d.rpc.dongles.Delete(d.device.DongleID)
	if d.natsSub != nil {
		err := d.natsSub.Unsubscribe()
		if err != nil && !errors.Is(err, nats.ErrConnectionDraining) && !errors.Is(err, nats.ErrConnectionClosed) {
			slog.Warn("Error unsubscribing from NATS", "error", err)
		}
	}
}
