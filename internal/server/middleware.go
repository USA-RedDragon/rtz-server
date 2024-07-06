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
	"github.com/USA-RedDragon/connect-server/internal/utils"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/gorm"
)

type AuthType uint8

const (
	AuthTypeUser AuthType = 1 << iota
	AuthTypeDevice
)

func applyMiddleware(r *gin.Engine, config *config.Config, otelComponent string, db *gorm.DB) {
	r.Use(gin.Recovery())
	r.Use(gin.Logger())
	r.TrustedPlatform = "X-Real-IP"

	// CORS
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowHeaders = append(corsConfig.AllowHeaders, "authorization")
	corsConfig.AllowCredentials = true
	if len(config.HTTP.CORSHosts) == 0 {
		corsConfig.AllowAllOrigins = true
	}
	corsConfig.AllowOrigins = config.HTTP.CORSHosts
	r.Use(cors.New(corsConfig))

	err := r.SetTrustedProxies(config.HTTP.TrustedProxies)
	if err != nil {
		slog.Error("Failed to set trusted proxies", "error", err.Error())
	}

	r.Use(dbMiddleware(db))
	r.Use(configMiddleware(config))

	if config.HTTP.Tracing.Enabled {
		r.Use(otelgin.Middleware(otelComponent))
		r.Use(tracingProvider(config))
	}
}

func configMiddleware(config *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("config", config)
		c.Next()
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

// Requires a jwt cookie
func requireCookieAuth(_ *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		cookie, err := c.Cookie("jwt")
		if err != nil || cookie == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

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
			slog.Error("Failed to parse device JWT token cookie", "error", err)
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

func requireAuth(config *config.Config, authType AuthType) gin.HandlerFunc {
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

		db, ok := c.MustGet("db").(*gorm.DB)
		if !ok {
			slog.Error("Failed to get db from context")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
			return
		}

		userAuthPass := false
		var userAuthErr error
		deviceAuthPass := false
		var deviceAuthErr error

		if authType&AuthTypeUser == AuthTypeUser {
			// Try verifying as user JWT
			uid, err := utils.VerifyJWT(config.JWT.Secret, jwtString)
			if err != nil {
				userAuthErr = err
			} else {
				user, err := models.FindUserByID(db, uid)
				if err != nil {
					userAuthErr = err
				} else {
					c.Set("user", &user)
					userAuthPass = true
				}
			}
		}

		if authType&AuthTypeDevice == AuthTypeDevice {
			// Try verifying as device JWT
			dongleID, ok := c.Params.Get("dongle_id")
			if !ok || dongleID == "" {
				deviceAuthErr = errors.New("missing dongle_id")
			} else {
				device, err := models.FindDeviceByDongleID(db, dongleID)
				if err != nil {
					deviceAuthErr = err
				} else {
					err = utils.VerifyDeviceJWT(device.DongleID, device.PublicKey, jwtString)
					if err != nil {
						deviceAuthErr = err
					} else {
						c.Set("device", &device)
						deviceAuthPass = true
					}
				}
			}
		}

		if authType&AuthTypeUser == AuthTypeUser && userAuthPass {
			c.Next()
			return
		}

		if authType&AuthTypeDevice == AuthTypeDevice && deviceAuthPass {
			c.Next()
			return
		}

		// Neither work, say why
		if deviceAuthErr != nil {
			slog.Error("Failed to verify device JWT", "error", deviceAuthErr)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}
		if userAuthErr != nil {
			slog.Error("Failed to verify user JWT", "error", userAuthErr)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}
	}
}

func requireDeviceOwner() gin.HandlerFunc {
	// User should be present from requireAuth
	// All these routes have a dongle_id param
	return func(c *gin.Context) {
		dongleID, ok := c.Params.Get("dongle_id")
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "dongle_id is required"})
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
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
			return
		}

		var subject any
		subject, ok = c.Get("user")
		if !ok {
			// Some of these routes also work with device auth
			subject, ok = c.Get("device")
			if !ok {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
				return
			}
		}
		switch subject := subject.(type) {
		case *models.User:
			if subject.ID != device.OwnerID {
				c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden"})
				return
			}
		case *models.Device:
			if subject.OwnerID != device.OwnerID {
				c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden"})
				return
			}

		default:
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
			return
		}

		c.Next()
	}
}
