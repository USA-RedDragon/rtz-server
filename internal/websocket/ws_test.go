package websocket_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/USA-RedDragon/rtz-server/internal/config"
	"github.com/USA-RedDragon/rtz-server/internal/db/models"
	"github.com/USA-RedDragon/rtz-server/internal/metrics"
	"github.com/USA-RedDragon/rtz-server/internal/websocket"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	gorillaWebsocket "github.com/gorilla/websocket"
	"github.com/nats-io/nats.go"
	"gorm.io/gorm"
)

type message struct {
	dongleID string
	data     string
}

type recorder struct {
	connected chan string
	messages  chan message
}

func (r *recorder) OnMessage(_ *http.Request, _ websocket.Writer, msg []byte, _ int, device *models.Device, _ *gorm.DB, _ *metrics.Metrics) {
	r.messages <- message{dongleID: device.DongleID, data: string(msg)}
}

func (r *recorder) OnConnect(_ context.Context, _ *http.Request, _ websocket.Writer, device *models.Device, _ *gorm.DB, _ *nats.Conn, _ *metrics.Metrics, _ *gorillaWebsocket.Conn) {
	r.connected <- device.DongleID
}

func (r *recorder) OnDisconnect(_ *http.Request, _ *models.Device, _ *gorm.DB, _ *metrics.Metrics) {}

func newServer(t *testing.T, corsHosts []string) (*httptest.Server, *recorder) {
	t.Helper()
	rec := &recorder{connected: make(chan string, 2), messages: make(chan message, 8)}
	return serve(t, rec, corsHosts), rec
}

func serve(t *testing.T, ws websocket.Websocket, corsHosts []string) *httptest.Server {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory"), &gorm.Config{})
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
	cfg.HTTP.CORSHosts = corsHosts
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("db", db)
		c.Set("metrics", (*metrics.Metrics)(nil))
	})
	r.GET("/ws/:dongle_id", websocket.CreateHandler(ws, cfg))
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return srv
}

func dial(t *testing.T, srv *httptest.Server, dongleID string, header http.Header) (*gorillaWebsocket.Conn, error) {
	t.Helper()
	url := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws/" + dongleID
	conn, resp, err := gorillaWebsocket.DefaultDialer.DialContext(t.Context(), url, header)
	if resp != nil && resp.Body != nil {
		_ = resp.Body.Close()
	}
	if err == nil {
		t.Cleanup(func() { _ = conn.Close() })
	}
	return conn, err
}

func waitFor[T any](t *testing.T, ch chan T) T {
	t.Helper()
	select {
	case v := <-ch:
		return v
	case <-time.After(2 * time.Second):
		t.Fatal("timed out")
	}
	var zero T
	return zero
}

func TestConnectionsDoNotShareSockets(t *testing.T) {
	t.Parallel()
	srv, rec := newServer(t, nil)

	connA, err := dial(t, srv, "a", nil)
	if err != nil {
		t.Fatal(err)
	}
	waitFor(t, rec.connected)
	connB, err := dial(t, srv, "b", nil)
	if err != nil {
		t.Fatal(err)
	}
	waitFor(t, rec.connected)

	for _, send := range []struct {
		conn *gorillaWebsocket.Conn
		msg  message
	}{
		{connA, message{"a", "one"}},
		{connA, message{"a", "two"}},
		{connB, message{"b", "three"}},
	} {
		if err := send.conn.WriteMessage(gorillaWebsocket.TextMessage, []byte(send.msg.data)); err != nil {
			t.Fatal(err)
		}
		if got := waitFor(t, rec.messages); got != send.msg {
			t.Errorf("got %v, want %v", got, send.msg)
		}
	}
}

func TestCheckOrigin(t *testing.T) {
	t.Parallel()
	srv, rec := newServer(t, []string{"https://example.com:443", "https://*.example.org"})

	tests := []struct {
		origin  string
		allowed bool
	}{
		{"", true},
		{"https://example.com", true},
		{"https://EXAMPLE.com", true},
		{"https://example.com.attacker.test", false},
		{"https://notexample.com", false},
		{"http://example.com", false},
		{"https://app.example.org", true},
		{"https://example.org.attacker.test", false},
	}
	for _, tt := range tests {
		header := http.Header{}
		if tt.origin != "" {
			header.Set("Origin", tt.origin)
		}
		_, err := dial(t, srv, "a", header)
		if allowed := err == nil; allowed != tt.allowed {
			t.Errorf("%q: allowed %v, want %v (%v)", tt.origin, allowed, tt.allowed, err)
		}
		if err == nil {
			waitFor(t, rec.connected)
		}
	}
}

type eagerWriter struct {
	recorder
}

func (e *eagerWriter) OnConnect(_ context.Context, _ *http.Request, w websocket.Writer, device *models.Device, _ *gorm.DB, _ *nats.Conn, _ *metrics.Metrics, _ *gorillaWebsocket.Conn) {
	for range 50 {
		w.WriteMessage(websocket.Message{Type: gorillaWebsocket.TextMessage, Data: []byte("hello")})
	}
	e.connected <- device.DongleID
}

func TestPingDoesNotRaceWrites(t *testing.T) {
	t.Parallel()
	eager := &eagerWriter{recorder{connected: make(chan string, 1)}}
	srv := serve(t, eager, nil)

	conn, err := dial(t, srv, "a", nil)
	if err != nil {
		t.Fatal(err)
	}
	waitFor(t, eager.connected)
	for range 50 {
		if _, msg, err := conn.ReadMessage(); err != nil || string(msg) != "hello" {
			t.Fatalf("got %q, %v", msg, err)
		}
	}
}
