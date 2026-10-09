package server

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/USA-RedDragon/rtz-server/internal/config"
	"github.com/USA-RedDragon/rtz-server/internal/db/models"
	"github.com/USA-RedDragon/rtz-server/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

func newRSAKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	return key
}

func publicKeyPEM(t *testing.T, key *rsa.PrivateKey) string {
	t.Helper()
	der, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}))
}

func deviceToken(t *testing.T, key *rsa.PrivateKey, identity string) string {
	t.Helper()
	now := time.Now()
	token, err := jwt.NewWithClaims(jwt.SigningMethodRS256, utils.DeviceJWT{
		Identity: identity,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now.Add(-30 * time.Second)),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
		},
	}).SignedString(key)
	if err != nil {
		t.Fatal(err)
	}
	return token
}

func TestRequireDeviceAuthWithoutDongleID(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.Device{}); err != nil {
		t.Fatal(err)
	}
	deviceKey := newRSAKey(t)
	if err := db.Create(&models.Device{DongleID: "abc", Serial: "serial", PublicKey: publicKeyPEM(t, deviceKey)}).Error; err != nil {
		t.Fatal(err)
	}

	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("db", db) })
	r.GET("/v4/:tileset", requireAuth(&config.Config{}, AuthTypeDevice), func(c *gin.Context) {
		device, ok := c.Get("device")
		if d, isDevice := device.(*models.Device); !ok || !isDevice || d.DongleID != "abc" {
			c.Status(http.StatusTeapot)
			return
		}
		c.Status(http.StatusOK)
	})

	tests := []struct {
		name  string
		token string
		want  int
	}{
		{"device key", deviceToken(t, deviceKey, "abc"), http.StatusOK},
		{"forged key", deviceToken(t, newRSAKey(t), "abc"), http.StatusUnauthorized},
		{"unknown device", deviceToken(t, deviceKey, "nope"), http.StatusUnauthorized},
		{"not a jwt", "garbage", http.StatusUnauthorized},
	}
	for _, tt := range tests {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/v4/tiles", nil)
		req.Header.Set("Authorization", "JWT "+tt.token)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != tt.want {
			t.Errorf("%s: got status %d, want %d", tt.name, w.Code, tt.want)
		}
	}
}

func TestRequireCookieAuthBadStoredKey(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.Device{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.Device{DongleID: "abc", Serial: "serial", PublicKey: "not a PEM key"}).Error; err != nil {
		t.Fatal(err)
	}

	r := gin.New()
	r.Use(gin.Recovery(), func(c *gin.Context) { c.Set("db", db) })
	r.GET("/ws/v2/:dongle_id", requireCookieAuth(&config.Config{}), func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/ws/v2/abc", nil)
	req.AddCookie(&http.Cookie{Name: "jwt", Value: deviceToken(t, newRSAKey(t), "abc"), Secure: true, HttpOnly: true, SameSite: http.SameSiteStrictMode})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("got status %d, want %d", w.Code, http.StatusUnauthorized)
	}
}
