package websocket

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/USA-RedDragon/rtz-server/internal/config"
	"github.com/USA-RedDragon/rtz-server/internal/metrics"
	"github.com/USA-RedDragon/rtz-server/internal/server/apimodels"
	"github.com/USA-RedDragon/rtz-server/internal/utils"
)

//nolint:gochecknoglobals
var testMetrics = sync.OnceValue(metrics.NewMetrics)

func TestLateResponseAfterTimeout(t *testing.T) {
	t.Parallel()
	m := testMetrics()
	ws := CreateRPCWebsocket(&config.Config{}, m)

	bidi := &bidiChannel{
		open:     true,
		inbound:  make(chan apimodels.RPCCall),
		outbound: make(chan apimodels.RPCResponse),
	}
	d := &dongle{bidiChannel: bidi, channelWatcher: utils.NewChannelWatcher(bidi.outbound)}
	go d.channelWatcher.WatchChannel(func(resp apimodels.RPCResponse) string { return resp.ID })
	ws.dongles.Store("abc", d)

	go func() {
		for call := range bidi.inbound {
			if call.ID == "fast" {
				bidi.outbound <- apimodels.RPCResponse{ID: call.ID}
			}
		}
	}()

	ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()
	if _, err := ws.Call(ctx, nil, m, "abc", apimodels.RPCCall{ID: "slow"}); err == nil {
		t.Fatal("expected a timeout")
	}

	bidi.outbound <- apimodels.RPCResponse{ID: "slow"}

	ctx, cancel = context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()
	resp, err := ws.Call(ctx, nil, m, "abc", apimodels.RPCCall{ID: "fast"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ID != "fast" {
		t.Errorf("got response %q, want fast", resp.ID)
	}
}
