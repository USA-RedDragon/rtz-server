package server

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/USA-RedDragon/connect-server/internal/config"
	"github.com/USA-RedDragon/connect-server/internal/db/models"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/gorm"
)

func applyMiddleware(r *gin.Engine, config *config.Config, otelComponent string, db *gorm.DB) {
	r.Use(gin.Recovery())
	r.Use(gin.Logger())
	r.TrustedPlatform = "X-Real-IP"

	err := r.SetTrustedProxies(config.HTTP.TrustedProxies)
	if err != nil {
		slog.Error("Failed to set trusted proxies", "error", err.Error())
	}

	r.Use(dbMiddleware(db))

	if config.HTTP.Tracing.Enabled {
		r.Use(otelgin.Middleware(otelComponent))
		r.Use(tracingProvider(config))
	}
}

func tracingProvider(config *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		if config.HTTP.Tracing.OTLPEndpoint != "" {
			ctx := c.Request.Context()
			span := trace.SpanFromContext(ctx)
			if span.IsRecording() {
				span.SetAttributes(
					attribute.String("http.method", c.Request.Method),
					attribute.String("http.path", c.Request.URL.Path),
				)
			}
		}
		c.Next()
	}
}

func dbMiddleware(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("db", db)
		c.Next()
	}
}

// Requires an Authorization: JWT <token> header
func requireAuth(_ *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		if !strings.HasPrefix(authHeader, "JWT ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		jwtString := strings.TrimPrefix(authHeader, "JWT ")

		device, ok := c.MustGet("device").(*models.Device)
		if !ok {
			slog.Error("Failed to get device from context")
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		type Claims struct {
			jwt.RegisteredClaims
			Identity string `json:"identity"`
		}
		claims := new(Claims)

		// Verify the token
		token, err := jwt.NewParser(jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Name})).ParseWithClaims(jwtString, claims, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, fmt.Errorf("invalid signing method: %s", token.Header["alg"])
			}
			claims = token.Claims.(*Claims)

			// ParseWithClaims will skip expiration check
			// if expiration has default value;
			// forcing a check and exiting if not set
			if claims.ExpiresAt == nil {
				return nil, errors.New("token has no expiration")
			}

			if claims.Identity != device.DongleID {
				return nil, errors.New("identity does not match device")
			}

			blk, _ := pem.Decode([]byte(device.PublicKey))
			key, err := x509.ParsePKIXPublicKey(blk.Bytes)
			if err != nil {
				return nil, fmt.Errorf("failed to parse public key: %w", err)
			}
			return key, nil
		})
		if err != nil {
			slog.Error("Failed to parse token", "error", err)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		if !token.Valid {
			slog.Error("Invalid token")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		c.Next()
	}
}

func setDevice() gin.HandlerFunc {
	return func(c *gin.Context) {
		dongleID, ok := c.Params.Get("dongle_id")
		if !ok {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "dongle_id is required"})
			return
		}
		db, ok := c.MustGet("db").(*gorm.DB)
		if !ok {
			slog.Error("Failed to get db from context")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
			return
		}
		device, err := models.FindDeviceByDongleID(db, dongleID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}
		c.Set("device", &device)

		c.Next()
	}
}

// Requires a jwt cookie
func requireCookieAuth(_ *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		cookie, err := c.Cookie("jwt")
		if err != nil || cookie == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		device, ok := c.MustGet("device").(*models.Device)
		if !ok {
			slog.Error("Failed to get device from context")
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		claims := new(jwt.RegisteredClaims)

		// Verify the token
		token, err := jwt.NewParser(jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Name})).ParseWithClaims(cookie, claims, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, fmt.Errorf("invalid signing method: %s", token.Header["alg"])
			}
			claims = token.Claims.(*jwt.RegisteredClaims)

			// ParseWithClaims will skip expiration check
			// if expiration has default value;
			// forcing a check and exiting if not set
			if claims.ExpiresAt == nil {
				return nil, errors.New("token has no expiration")
			}

			blk, _ := pem.Decode([]byte(device.PublicKey))
			key, err := x509.ParsePKIXPublicKey(blk.Bytes)
			if err != nil {
				return nil, fmt.Errorf("failed to parse public key: %w", err)
			}
			return key, nil
		})
		if err != nil {
			slog.Error("Failed to parse token", "error", err)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		if !token.Valid {
			slog.Error("Invalid token")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		c.Next()
	}
}
