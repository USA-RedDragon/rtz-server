package websocket_test

import (
	"net/http/httptest"
	"runtime"
	"testing"
	"time"

	"github.com/USA-RedDragon/rtz-server/internal/config"
	"github.com/USA-RedDragon/rtz-server/internal/websocket"
	gorillaWebsocket "github.com/gorilla/websocket"
)

const (
	testPongWait   = 300 * time.Millisecond
	testPingPeriod = 50 * time.Millisecond
)

func serveTimed(t *testing.T, ws websocket.Websocket) *httptest.Server {
	t.Helper()
	return serveHandler(t, websocket.CreateHandlerWithTimings(ws, &config.Config{}, testPongWait, testPingPeriod))
}

// readUntilClosed reads from conn, answering pings unless the ping handler
// was replaced, and reports when the connection ends.
func readUntilClosed(conn *gorillaWebsocket.Conn) <-chan error {
	closed := make(chan error, 1)
	go func() {
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				closed <- err
				return
			}
		}
	}()
	return closed
}

func TestConnectionAnsweringPingsStaysOpen(t *testing.T) {
	t.Parallel()
	rec := &recorder{connected: make(chan string, 1), messages: make(chan message, 1)}
	srv := serveTimed(t, rec)

	conn, err := dial(t, srv, "a", nil)
	if err != nil {
		t.Fatal(err)
	}
	waitFor(t, rec.connected)
	closed := readUntilClosed(conn)

	select {
	case err := <-closed:
		t.Fatalf("connection closed: %v", err)
	case <-time.After(4 * testPongWait):
	}
	if err := conn.WriteMessage(gorillaWebsocket.TextMessage, []byte("still here")); err != nil {
		t.Fatal(err)
	}
	if got := waitFor(t, rec.messages); got.data != "still here" {
		t.Errorf("got %q", got.data)
	}
}

func TestConnectionIgnoringPingsIsClosed(t *testing.T) {
	t.Parallel()
	rec := &recorder{connected: make(chan string, 1)}
	srv := serveTimed(t, rec)

	conn, err := dial(t, srv, "a", nil)
	if err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	conn.SetPingHandler(func(string) error { return nil })
	waitFor(t, rec.connected)
	closed := readUntilClosed(conn)

	select {
	case <-closed:
	case <-time.After(10 * testPongWait):
		t.Fatal("connection still open")
	}
	if elapsed := time.Since(start); elapsed < testPongWait/2 {
		t.Errorf("closed after %v, before the pong wait of %v", elapsed, testPongWait)
	}
}

//nolint:paralleltest // counts goroutines, so it must not share the process with other tests
func TestDisconnectLeavesNoGoroutines(t *testing.T) {
	rec := &recorder{connected: make(chan string, 1), messages: make(chan message, 1)}
	srv := serveTimed(t, rec)
	before := runtime.NumGoroutine()

	conn, err := dial(t, srv, "a", nil)
	if err != nil {
		t.Fatal(err)
	}
	waitFor(t, rec.connected)
	closed := readUntilClosed(conn)
	time.Sleep(3 * testPingPeriod)
	_ = conn.Close()
	<-closed

	deadline := time.Now().Add(5 * time.Second)
	for runtime.NumGoroutine() > before {
		if time.Now().After(deadline) {
			buf := make([]byte, 1<<20)
			t.Fatalf("%d goroutines, want at most %d:\n%s", runtime.NumGoroutine(), before, buf[:runtime.Stack(buf, true)])
		}
		time.Sleep(10 * time.Millisecond)
	}
}
