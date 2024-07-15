package websocket

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/USA-RedDragon/rtz-server/internal/config"
	"github.com/USA-RedDragon/rtz-server/internal/metrics"
	"github.com/USA-RedDragon/rtz-server/internal/server/apimodels"
	"github.com/USA-RedDragon/rtz-server/internal/websocket"
	gorillaWebsocket "github.com/gorilla/websocket"
	"github.com/nats-io/nats.go"
	"github.com/puzpuzpuz/xsync/v3"
	"golang.org/x/sync/errgroup"
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

func (c *RPCWebsocket) Call(ctx context.Context, nc *nats.Conn, metrics *metrics.Metrics, dongleID string, call apimodels.RPCCall) (apimodels.RPCResponse, error) {
	dongle, loaded := c.dongles.Load(dongleID)
	if !loaded {
		// Dongle is not here, send to NATS if enabled
		if nc != nil {
			msg, err := json.Marshal(call)
			if err != nil {
				return apimodels.RPCResponse{}, err
			}

			retry := 0
			for {
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
				resp, err = nc.Request("rpc:call:"+dongleID, msg, timeout)
				if err != nil {
					if errors.Is(err, nats.ErrTimeout) {
						continue
					} else {
						return apimodels.RPCResponse{}, err
					}
				}
				if resp.Data == nil {
					// This is essentially a NAK
					// We should retry and not count towards the limit
					// because the dongle is probably reconnecting
					retry--
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

	context, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()
	select {
	case <-context.Done():
		metrics.IncrementAthenaErrors(dongleID, "rpc_call_timeout")
		return apimodels.RPCResponse{}, fmt.Errorf("timeout")
	case resp := <-responseChan:
		return resp, nil
	}
}
