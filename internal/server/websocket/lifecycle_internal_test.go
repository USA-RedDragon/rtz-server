package websocket

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/USA-RedDragon/rtz-server/internal/config"
	"github.com/USA-RedDragon/rtz-server/internal/db/models"
	"github.com/USA-RedDragon/rtz-server/internal/server/apimodels"
	"github.com/USA-RedDragon/rtz-server/internal/websocket"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	gorillaWebsocket "github.com/gorilla/websocket"
	"gorm.io/gorm"
)

var databases atomic.Int64

// tinyBufferListener shrinks the kernel send buffer of every accepted
// connection so that a client which stops reading stalls the server quickly.
type tinyBufferListener struct {
	net.Listener
}

func (l tinyBufferListener) Accept() (net.Conn, error) {
	conn, err := l.Listener.Accept()
	if tcp, ok := conn.(*net.TCPConn); ok {
		_ = tcp.SetWriteBuffer(1024)
	}
	return conn, err
}

func newRPCServer(t *testing.T, tinyBuffers bool) (*RPCWebsocket, *httptest.Server) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s-%d?mode=memory&cache=shared", t.Name(), databases.Add(1))), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.Device{}); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"a", "b"} {
		if err := db.Create(&models.Device{DongleID: id, Serial: id, PublicKey: id}).Error; err != nil {
			t.Fatal(err)
		}
	}

	cfg := &config.Config{}
	ws := CreateRPCWebsocket(cfg, testMetrics())
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("db", db)
		c.Set("metrics", testMetrics())
	})
	r.GET("/ws/:dongle_id", websocket.CreateHandler(ws, cfg))
	srv := httptest.NewUnstartedServer(r)
	if tinyBuffers {
		srv.Listener = tinyBufferListener{srv.Listener}
	}
	srv.Start()
	t.Cleanup(srv.Close)
	return ws, srv
}

func dialDevice(ctx context.Context, srv *httptest.Server, dongleID string, tinyBuffers bool) (*gorillaWebsocket.Conn, error) {
	dialer := *gorillaWebsocket.DefaultDialer
	if tinyBuffers {
		dialer.NetDialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			conn, err := (&net.Dialer{}).DialContext(ctx, network, addr)
			if tcp, ok := conn.(*net.TCPConn); ok {
				_ = tcp.SetReadBuffer(1024)
			}
			return conn, err
		}
	}
	url := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws/" + dongleID
	conn, resp, err := dialer.DialContext(ctx, url, nil)
	if resp != nil && resp.Body != nil {
		_ = resp.Body.Close()
	}
	return conn, err
}

func connectDevice(t *testing.T, ws *RPCWebsocket, srv *httptest.Server, tinyBuffers bool) *gorillaWebsocket.Conn {
	t.Helper()
	before, _ := ws.dongles.Load("a")
	conn, err := dialDevice(t.Context(), srv, "a", tinyBuffers)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	waitUntil(t, func() bool {
		d, ok := ws.dongles.Load("a")
		return ok && d != before
	})
	return conn
}

func waitUntil(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatal("timed out waiting for condition")
		}
		time.Sleep(5 * time.Millisecond)
	}
}

type deviceMessage struct {
	ID     string `json:"id"`
	Method string `json:"method"`
	Result any    `json:"result"`
}

func readDeviceMessage(conn *gorillaWebsocket.Conn) (deviceMessage, error) {
	var msg deviceMessage
	_, data, err := conn.ReadMessage()
	if err != nil {
		return msg, err
	}
	return msg, json.Unmarshal(data, &msg)
}

// answerCalls replies to every call the device receives with result.
func answerCalls(conn *gorillaWebsocket.Conn, result any) {
	for {
		msg, err := readDeviceMessage(conn)
		if err != nil {
			return
		}
		if msg.Method == "" {
			continue
		}
		data, err := json.Marshal(map[string]any{"id": msg.ID, "jsonrpc": "2.0", "result": result})
		if err != nil {
			return
		}
		if err := conn.WriteMessage(gorillaWebsocket.TextMessage, data); err != nil {
			return
		}
	}
}

func TestDisconnectWhileHandlersSend(t *testing.T) {
	t.Parallel()
	ws, srv := newRPCServer(t, false)

	for i := range 20 {
		conn := connectDevice(t, ws, srv, false)
		for j := range 100 {
			for _, msg := range []string{
				fmt.Sprintf(`{"method":"forwardLogs","id":"log-%d-%d","jsonrpc":"2.0","params":{}}`, i, j),
				`{"result":{},"id":"unknown","jsonrpc":"2.0"}`,
			} {
				if err := conn.WriteMessage(gorillaWebsocket.TextMessage, []byte(msg)); err != nil {
					t.Fatal(err)
				}
			}
		}
		_ = conn.Close()
		waitUntil(t, func() bool { return ws.dongles.Size() == 0 })
	}
}

func TestCallReturnsWhenConnectionDrops(t *testing.T) {
	t.Parallel()
	ws, srv := newRPCServer(t, false)
	conn := connectDevice(t, ws, srv, false)

	errc := make(chan error, 1)
	go func() {
		_, err := ws.Call(t.Context(), nil, testMetrics(), "a", apimodels.RPCCall{ID: "1", Method: "getVersion"})
		errc <- err
	}()
	for {
		msg, err := readDeviceMessage(conn)
		if err != nil {
			t.Fatal(err)
		}
		if msg.Method != "" {
			break
		}
	}
	_ = conn.Close()

	select {
	case err := <-errc:
		if !errors.Is(err, ErrNotConnected) {
			t.Errorf("got %v, want ErrNotConnected", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Call kept waiting after the device disconnected")
	}
}

func TestCallReturnsWhenContextIsCancelled(t *testing.T) {
	t.Parallel()
	ws, srv := newRPCServer(t, true)
	connectDevice(t, ws, srv, true)

	// The device never reads, so once the socket and the write queue are
	// full every further Call can only return through its context.
	params := strings.Repeat("x", 4096)
	var wg sync.WaitGroup
	for i := range 1500 {
		wg.Go(func() {
			ctx, cancel := context.WithTimeout(t.Context(), 500*time.Millisecond)
			defer cancel()
			_, err := ws.Call(ctx, nil, testMetrics(), "a", apimodels.RPCCall{ID: fmt.Sprint(i), Method: "echo", Params: params})
			if err == nil {
				t.Error("expected an error from a device that never answers")
			}
		})
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("Call ignored its context while the connection was stalled")
	}
}

func TestConnectDisconnectStress(t *testing.T) {
	t.Parallel()
	ws, srv := newRPCServer(t, false)

	var wg sync.WaitGroup
	for g := range 8 {
		wg.Go(func() {
			dongleID := []string{"a", "b"}[g%2]
			for i := range 10 {
				conn, err := dialDevice(t.Context(), srv, dongleID, false)
				if err != nil {
					t.Error(err)
					return
				}
				log := fmt.Sprintf(`{"method":"forwardLogs","id":"log-%d-%d","jsonrpc":"2.0","params":{}}`, g, i)
				_ = conn.WriteMessage(gorillaWebsocket.TextMessage, []byte(log))
				go answerCalls(conn, "ok")

				var calls sync.WaitGroup
				for k := range 3 {
					calls.Go(func() {
						ctx, cancel := context.WithTimeout(t.Context(), 300*time.Millisecond)
						defer cancel()
						start := time.Now()
						_, _ = ws.Call(ctx, nil, testMetrics(), dongleID, apimodels.RPCCall{ID: fmt.Sprintf("%d-%d-%d", g, i, k), Method: "echo"})
						if elapsed := time.Since(start); elapsed > 2*time.Second {
							t.Errorf("Call took %v", elapsed)
						}
					})
				}
				calls.Wait()
				_ = conn.Close()
			}
		})
	}
	wg.Wait()
	waitUntil(t, func() bool { return ws.dongles.Size() == 0 })
}

func TestDeviceCallsAreAnsweredOnce(t *testing.T) {
	t.Parallel()
	for _, method := range []string{"forwardLogs", "storeStats"} {
		t.Run(method, func(t *testing.T) {
			t.Parallel()
			ws, srv := newRPCServer(t, false)
			conn := connectDevice(t, ws, srv, false)

			// The server has a call of its own in flight with the same id, which
			// the reply to the device must not resolve.
			type result struct {
				resp apimodels.RPCResponse
				err  error
			}
			callDone := make(chan result, 1)
			go func() {
				resp, err := ws.Call(t.Context(), nil, testMetrics(), "a", apimodels.RPCCall{ID: "7", Method: "getVersion"})
				callDone <- result{resp, err}
			}()
			if msg, err := readDeviceMessage(conn); err != nil || msg.Method != "getVersion" {
				t.Fatalf("got %+v, %v, want the server's call", msg, err)
			}

			req := `{"method":"` + method + `","id":"7","jsonrpc":"2.0","params":{}}`
			if err := conn.WriteMessage(gorillaWebsocket.TextMessage, []byte(req)); err != nil {
				t.Fatal(err)
			}

			var replies []deviceMessage
			_ = conn.SetReadDeadline(time.Now().Add(time.Second))
			for {
				msg, err := readDeviceMessage(conn)
				if err != nil {
					break
				}
				replies = append(replies, msg)
			}
			if len(replies) != 1 {
				t.Fatalf("got %d replies, want exactly one: %+v", len(replies), replies)
			}
			if result, ok := replies[0].Result.(map[string]any); replies[0].ID != "7" || !ok || result["success"] != true {
				t.Errorf("got reply %+v, want success for id 7", replies[0])
			}

			select {
			case r := <-callDone:
				t.Fatalf("the reply to the device answered the server's call: %+v, %v", r.resp, r.err)
			default:
			}
			_ = conn.SetReadDeadline(time.Time{})
			if err := conn.WriteMessage(gorillaWebsocket.TextMessage, []byte(`{"id":"7","jsonrpc":"2.0","result":"1.0"}`)); err != nil {
				t.Fatal(err)
			}
			select {
			case r := <-callDone:
				if r.err != nil || r.resp.Result != "1.0" {
					t.Errorf("got %+v, %v, want the device's answer", r.resp, r.err)
				}
			case <-time.After(3 * time.Second):
				t.Fatal("the server's call was never answered")
			}
		})
	}
}

func TestReconnectKeepsTheNewConnection(t *testing.T) {
	t.Parallel()
	ws, srv := newRPCServer(t, false)

	first := connectDevice(t, ws, srv, false)
	second := connectDevice(t, ws, srv, false)
	go answerCalls(second, "second")
	_ = first.Close()

	for i := range 50 {
		ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
		resp, err := ws.Call(ctx, nil, testMetrics(), "a", apimodels.RPCCall{ID: fmt.Sprint(i), Method: "echo"})
		cancel()
		if err != nil {
			t.Fatalf("call %d: %v", i, err)
		}
		if resp.Result != "second" {
			t.Fatalf("call %d: got %v, want an answer from the second connection", i, resp.Result)
		}
		time.Sleep(10 * time.Millisecond)
	}
}
