package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http/httptest"
	"strings"
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

func connectDevice(t *testing.T, ws *RPCWebsocket, srv *httptest.Server, dongleID string, tinyBuffers bool) *gorillaWebsocket.Conn {
	t.Helper()
	before, _ := ws.dongles.Load(dongleID)
	conn, err := dialDevice(t.Context(), srv, dongleID, tinyBuffers)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	waitUntil(t, func() bool {
		d, ok := ws.dongles.Load(dongleID)
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

func TestDeviceCallsAreAnsweredOnce(t *testing.T) {
	t.Parallel()
	for _, method := range []string{"forwardLogs", "storeStats"} {
		t.Run(method, func(t *testing.T) {
			t.Parallel()
			ws, srv := newRPCServer(t, false)
			conn := connectDevice(t, ws, srv, "a", false)

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

