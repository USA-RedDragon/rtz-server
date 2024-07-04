package server

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/USA-RedDragon/connect-server/internal/config"
	"github.com/USA-RedDragon/connect-server/internal/db/models"
	"github.com/USA-RedDragon/connect-server/internal/events"
	websocketControllers "github.com/USA-RedDragon/connect-server/internal/server/websocket"
	"github.com/USA-RedDragon/connect-server/internal/utils"
	"github.com/USA-RedDragon/connect-server/internal/websocket"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

func applyRoutes(r *gin.Engine, config *config.Config, eventsChannel chan events.Event) {
	r.GET("/health", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	authMiddleware := requireAuth(config)

	apiV1 := r.Group("/v1")
	apiV1.GET("/navigation/:dongle_id/next", setDevice(), authMiddleware, func(c *gin.Context) {
		slog.Info("Get Next Navigation", "url", c.Request.URL.String())
	})

	apiV1.DELETE("/navigation/:dongle_id/next", setDevice(), authMiddleware, func(c *gin.Context) {
		slog.Info("Delete Next Navigation", "url", c.Request.URL.String())
	})

	apiV1.GET("/navigation/:dongle_id/locations", setDevice(), authMiddleware, func(c *gin.Context) {
		slog.Info("Get Locations", "url", c.Request.URL.String())
	})

	apiV11 := r.Group("/v1.1")
	apiV11.GET("/devices/:dongle_id/", setDevice(), authMiddleware, func(c *gin.Context) {
		db, ok := c.MustGet("db").(*gorm.DB)
		if !ok {
			slog.Error("Failed to get db from context")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
			return
		}

		dongleID := c.Param("dongle_id")
		if dongleID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "dongle_id is required"})
			return
		}

		device, err := models.FindDeviceByDongleID(db, dongleID)
		if err != nil {
			slog.Error("Failed to find device", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
			return
		}

		c.JSON(http.StatusOK, device)
	})

	apiV11.GET("/devices/:dongle_id/stats", setDevice(), authMiddleware, func(c *gin.Context) {
		slog.Info("Get Stats", "url", c.Request.URL.String())
	})

	apiV14 := r.Group("/v1.4")
	apiV14.GET("/:dongle_id/upload_url", setDevice(), authMiddleware, func(c *gin.Context) {
		slog.Info("Get Upload URL", "url", c.Request.URL.String())
	})

	apiV2 := r.Group("/v2")

	// Google Auth Redirect
	apiV2.POST("/auth/g/redirect", func(c *gin.Context) {
	})

	// GitHub Auth Redirect
	apiV2.POST("/auth/h/redirect", func(c *gin.Context) {
	})

	apiV2.POST("/pilotpair", func(c *gin.Context) {
	})

	apiV2.POST("/pilotauth", func(c *gin.Context) {
		if !config.Registration.Enabled {
			c.JSON(http.StatusNotFound, gin.H{"error": "Registration is disabled"})
			return
		}
		param_imei, ok := c.GetQuery("imei")
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "imei is required"})
			return
		}
		if len(param_imei) != 15 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "imei must be 15 characters"})
			return
		}
		imei, err := strconv.ParseInt(param_imei, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "imei is not an integer"})
		}
		if !utils.LuhnValid(int(imei)) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "imei is invalid"})
			return
		}

		param_imei2, ok := c.GetQuery("imei2")
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "imei2 is required"})
			return
		}
		var imei2 int64
		if len(param_imei2) != 0 {
			imei2, err = strconv.ParseInt(param_imei2, 10, 64)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "imei2 is not an integer"})
			}
		}
		if !utils.LuhnValid(int(imei2)) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "imei2 is invalid"})
			return
		}

		param_serial, ok := c.GetQuery("serial")
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "serial is required"})
			return
		}
		if len(param_serial) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "serial is required"})
			return
		}

		param_public_key, ok := c.GetQuery("public_key")
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "public_key is required"})
			return
		}
		if len(param_public_key) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "public_key is required"})
			return
		}

		param_register_token, ok := c.GetQuery("register_token")
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "register_token is required"})
			return
		}

		blk, _ := pem.Decode([]byte(param_public_key))
		key, err := x509.ParsePKIXPublicKey(blk.Bytes)
		if err != nil {
			slog.Error("Failed to parse public key", "error", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "public_key is invalid"})
			return
		}

		type Claims struct {
			Register bool `json:"register,omitempty"`
			jwt.RegisteredClaims
		}

		var claims *Claims

		token, err := jwt.NewParser(jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Name})).
			ParseWithClaims(param_register_token, new(Claims), func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
					slog.Error("Invalid signing method", "method", token.Header["alg"])
				}
				claims = token.Claims.(*Claims)

				// ParseWithClaims will skip expiration check
				// if expiration has default value;
				// forcing a check and exiting if not set
				if claims.ExpiresAt == nil {
					return nil, errors.New("token has no expiration")
				}

				if !claims.Register {
					return nil, errors.New("register_token is not a register token")
				}

				return key, nil
			})
		if err != nil {
			slog.Error("Failed to parse token", "error", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "register_token is invalid"})
			return
		}

		if !token.Valid {
			slog.Error("Invalid token")
			c.JSON(http.StatusBadRequest, gin.H{"error": "register_token is invalid"})
			return
		}

		db, ok := c.MustGet("db").(*gorm.DB)
		if !ok {
			slog.Error("Failed to get db from context")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
			return
		}

		_, err = models.FindDeviceBySerial(db, param_serial)
		// We can ignore the error here, as we're just checking if the device exists
		if err == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "serial is already registered"})
			return
		}

		dongleID, err := models.GenerateDongleID(db)
		if err != nil {
			slog.Error("Failed to generate dongle ID", "error", err)
		}

		err = db.Create(&models.Device{
			DongleID:  dongleID,
			Serial:    param_serial,
			PublicKey: param_public_key,
		}).Error
		if err != nil {
			slog.Error("Failed to create device", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"dongle_id": dongleID})
	})

	r.NoRoute(func(c *gin.Context) {
		slog.Warn("Not Found", "path", c.Request.URL.Path)
		c.JSON(http.StatusNotFound, gin.H{"error": "Not Found"})
	})

	wsV2 := r.Group("/ws/v2")
	wsV2.GET("/:dongle_id", setDevice(), requireCookieAuth(config), websocket.CreateHandler(websocketControllers.CreateEventsWebsocket(eventsChannel), config))
}
