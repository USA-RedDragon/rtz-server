package websocket

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/USA-RedDragon/rtz-server/internal/metrics"
	"github.com/USA-RedDragon/rtz-server/internal/server/apimodels"
	gorillaWebsocket "github.com/gorilla/websocket"
	"github.com/nats-io/nats.go"
)

//nolint:gochecknoglobals
var testMetrics = sync.OnceValue(metrics.NewMetrics)

func TestLateResponseAfterTimeout(t *testing.T) {
	t.Parallel()
	m := testMetrics()
	ws, srv := newRPCServer(t, false)
	conn := connectDevice(t, ws, srv, false)

	ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()
	if _, err := ws.Call(ctx, nil, m, "a", apimodels.RPCCall{ID: "slow", Method: "echo"}); err == nil {
		t.Fatal("expected a timeout")
	}
	if msg, err := readDeviceMessage(conn); err != nil || msg.ID != "slow" {
		t.Fatalf("got %+v, %v, want the slow call", msg, err)
	}
	if err := conn.WriteMessage(gorillaWebsocket.TextMessage, []byte(`{"id":"slow","jsonrpc":"2.0","result":"late"}`)); err != nil {
		t.Fatal(err)
	}
	go answerCalls(conn, "fast")

	ctx, cancel = context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()
	resp, err := ws.Call(ctx, nil, m, "a", apimodels.RPCCall{ID: "fast", Method: "echo"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ID != "fast" || resp.Result != "fast" {
		t.Errorf("got response %+v, want fast", resp)
	}
}

type fakeNATS struct {
	reply []byte
}

func (f fakeNATS) RequestWithContext(context.Context, string, []byte) (*nats.Msg, error) {
	return &nats.Msg{Data: f.reply}, nil
}

func TestCallNATS(t *testing.T) {
	t.Parallel()
	m := testMetrics()

	resp, err := callNATS(t.Context(), fakeNATS{reply: []byte(`{"id":"1"}`)}, m, "abc", apimodels.RPCCall{ID: "1"})
	if err != nil || resp.ID != "1" {
		t.Errorf("got %v, %v, want response 1", resp, err)
	}

	ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
	defer cancel()
	done := make(chan error, 1)
	go func() {
		_, err := callNATS(ctx, fakeNATS{}, m, "abc", apimodels.RPCCall{ID: "2"})
		done <- err
	}()
	select {
	case err := <-done:
		if err == nil {
			t.Error("expected an error while the dongle keeps refusing")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("callNATS kept retrying after the context was done")
	}
}

// silentNATS never answers, like a server whose dongle is busy.
type silentNATS struct{}

func (silentNATS) RequestWithContext(ctx context.Context, _ string, _ []byte) (*nats.Msg, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}

func TestCallNATSReturnsWhenContextIsCancelled(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err := callNATS(ctx, silentNATS{}, testMetrics(), "abc", apimodels.RPCCall{ID: "1", Method: "takeSnapshot"})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("got %v, want the context's error", err)
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Errorf("callNATS took %v after its context was done", elapsed)
	}
}
