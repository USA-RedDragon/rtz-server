package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/USA-RedDragon/rtz-server/internal/config"
	"github.com/gin-gonic/gin"
)

// Not parallel because it swaps gin's global log writer.
//
//nolint:paralleltest
func TestRequestLogOmitsQuery(t *testing.T) {
	var logs bytes.Buffer
	defaultWriter := gin.DefaultWriter
	gin.DefaultWriter = &logs
	t.Cleanup(func() { gin.DefaultWriter = defaultWriter })

	s := NewServer(&config.Config{}, nil, nil, nil, nil, nil)
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/missing?access_token=secret-token", nil)
	s.ipv4Server.Handler.ServeHTTP(httptest.NewRecorder(), req)

	if strings.Contains(logs.String(), "secret-token") {
		t.Errorf("request log contains the access token:\n%s", logs.String())
	}
	if n := strings.Count(logs.String(), "/missing"); n != 1 {
		t.Errorf("request logged %d times, want 1:\n%s", n, logs.String())
	}
}
