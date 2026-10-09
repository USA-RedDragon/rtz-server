package websocket

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/USA-RedDragon/rtz-server/internal/config"
	"github.com/USA-RedDragon/rtz-server/internal/metrics"
	"github.com/USA-RedDragon/rtz-server/internal/server/apimodels"
	"github.com/USA-RedDragon/rtz-server/internal/websocket"
	gorillaWebsocket "github.com/gorilla/websocket"
	"github.com/nats-io/nats.go"
	"github.com/puzpuzpuz/xsync/v3"
)

var (
	ErrNotConnected = errors.New("dongle not connected")
)

type RPCWebsocket struct {
	websocket.Websocket
	dongles *xsync.MapOf[string, *dongle]
	metrics *metrics.Metrics
	config  *config.Config
}

func CreateRPCWebsocket(config *config.Config, metrics *metrics.Metrics) *RPCWebsocket {
	socket := &RPCWebsocket{
		dongles: xsync.NewMapOf[string, *dongle](),
		metrics: metrics,
		config:  config,
	}
	return socket
}

// Stop asks every connected device to close its connection and waits until
// each connection has been cleaned up, or ctx is done.
func (c *RPCWebsocket) Stop(ctx context.Context) error {
	var wg sync.WaitGroup
	c.dongles.Range(func(_ string, d *dongle) bool {
		wg.Go(func() { d.stop(ctx) })
		return true
	})
	wg.Wait()
	return ctx.Err()
}

func (d *dongle) stop(ctx context.Context) {
	err := d.conn.WriteControl(
		gorillaWebsocket.CloseMessage,
		gorillaWebsocket.FormatCloseMessage(gorillaWebsocket.CloseServiceRestart, "Server is restarting"),
		time.Now().Add(5*time.Second))
	if err != nil {
		slog.Warn("Error sending close message to websocket", "error", err)
		_ = d.conn.Close()
	}

	timer := time.NewTimer(6 * time.Second)
	defer timer.Stop()
	select {
	case <-d.closed:
		return
	case <-ctx.Done():
	case <-timer.C:
	}
	slog.Warn("Timeout waiting for websocket to close")
	_ = d.conn.Close()
	select {
	case <-d.closed:
	case <-ctx.Done():
	}
}

func (c *RPCWebsocket) Call(ctx context.Context, nc *nats.Conn, metrics *metrics.Metrics, dongleID string, call apimodels.RPCCall) (apimodels.RPCResponse, error) {
	dongle, loaded := c.dongles.Load(dongleID)
	if !loaded {
		// Dongle is not here, send to NATS if enabled
		if nc != nil {
			return callNATS(ctx, nc, metrics, dongleID, call)
		}
		return apimodels.RPCResponse{}, ErrNotConnected
	}

	ctx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()
	resp, err := dongle.call(ctx, call)
	if errors.Is(err, context.DeadlineExceeded) {
		metrics.IncrementAthenaErrors(dongleID, "rpc_call_timeout")
	}
	return resp, err
}

// natsRequester is the part of *nats.Conn that callNATS uses.
type natsRequester interface {
	RequestWithContext(ctx context.Context, subj string, data []byte) (*nats.Msg, error)
}

func callNATS(ctx context.Context, nc natsRequester, metrics *metrics.Metrics, dongleID string, call apimodels.RPCCall) (apimodels.RPCResponse, error) {
	msg, err := json.Marshal(call)
	if err != nil {
		return apimodels.RPCResponse{}, err
	}

	ctx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()

	retry := 0
	for {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return apimodels.RPCResponse{}, ctxErr
		}
		if retry > 3 {
			return apimodels.RPCResponse{}, err
		}
		retry++
		timeout := 5 * time.Second
		// Special cases for longer RPC calls
		switch call.Method {
		case "takeSnapshot":
			timeout = 30 * time.Second
		default:
		}
		var resp *nats.Msg
		attemptCtx, cancelAttempt := context.WithTimeout(ctx, timeout)
		resp, err = nc.RequestWithContext(attemptCtx, "rpc:call:"+dongleID, msg)
		cancelAttempt()
		if err != nil {
			switch {
			case ctx.Err() != nil:
				return apimodels.RPCResponse{}, ctx.Err()
			case errors.Is(err, nats.ErrTimeout), errors.Is(err, context.DeadlineExceeded):
				continue
			case errors.Is(err, nats.ErrNoResponders):
				// This could be a dongle reconnecting
				waitForRetry(ctx)
				continue
			default:
				return apimodels.RPCResponse{}, err
			}
		}
		if len(resp.Data) == 0 {
			// This is essentially a NAK
			// We should retry and not count towards the limit
			// because the dongle is probably reconnecting
			retry--
			waitForRetry(ctx)
			continue
		}

		var rpcResp apimodels.RPCResponse
		slog.Debug("Received RPC response from NATS", "response", string(resp.Data))
		err = json.Unmarshal(resp.Data, &rpcResp)
		if err != nil {
			metrics.IncrementAthenaErrors(dongleID, "rpc_call_nats_unmarshal")
			slog.Warn("Error unmarshalling RPC response", "error", err)
			return apimodels.RPCResponse{}, err
		}

		return rpcResp, nil
	}
}

func waitForRetry(ctx context.Context) {
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	select {
	case <-ctx.Done():
	case <-timer.C:
	}
}
