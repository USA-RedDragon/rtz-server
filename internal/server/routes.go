package server

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/USA-RedDragon/connect-server/internal/config"
	"github.com/USA-RedDragon/connect-server/internal/events"
	websocketControllers "github.com/USA-RedDragon/connect-server/internal/server/websocket"
	"github.com/USA-RedDragon/connect-server/internal/utils"
	"github.com/USA-RedDragon/connect-server/internal/websocket"
	"github.com/gin-gonic/gin"
)

func applyRoutes(r *gin.Engine, config *config.Config, eventsChannel chan events.Event) {
	r.GET("/health", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	authMiddleware := requireAuth(config)

	apiV1 := r.Group("/v1")
	apiV1.GET("/navigation/:dongle_id/next", authMiddleware, func(c *gin.Context) {
		slog.Info("Get Next Navigation", "url", c.Request.URL.String())
	})

	apiV1.DELETE("/navigation/:dongle_id/next", authMiddleware, func(c *gin.Context) {
		slog.Info("Delete Next Navigation", "url", c.Request.URL.String())
	})

	apiV1.GET("/navigation/:dongle_id/locations", authMiddleware, func(c *gin.Context) {
		slog.Info("Get Locations", "url", c.Request.URL.String())
	})

	apiV11 := r.Group("/v1.1")
	apiV11.GET("/devices/:dongle_id/", authMiddleware, func(c *gin.Context) {
		slog.Info("Get Device", "url", c.Request.URL.String())
	})

	apiV11.GET("/devices/:dongle_id/stats", authMiddleware, func(c *gin.Context) {
		slog.Info("Get Stats", "url", c.Request.URL.String())
	})

	apiV14 := r.Group("/v1.4")
	apiV14.GET("/:dongle_id/upload_url", authMiddleware, func(c *gin.Context) {
		slog.Info("Get Upload URL", "url", c.Request.URL.String())
	})

	apiV2 := r.Group("/v2")
	apiV2.POST("/pilotauth", func(c *gin.Context) {
		slog.Info("Pilot Auth", "url", c.Request.URL.String())
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
		if len(param_imei2) != 15 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "imei2 must be 15 characters"})
			return
		}
		imei2, err := strconv.ParseInt(param_imei2, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "imei2 is not an integer"})
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
		// Validate register_token as JWT signed by the public key

		slog.Info("Pilot Auth", "imei", imei, "imei2", imei2, "serial", param_serial, "public_key", param_public_key, "register_token", param_register_token)

		c.JSON(http.StatusInternalServerError, gin.H{"status": "ok"})
	})

	r.NoRoute(func(c *gin.Context) {
		slog.Warn("Not Found", "path", c.Request.URL.Path)
		c.JSON(http.StatusNotFound, gin.H{"error": "Not Found"})
	})

	wsV2 := r.Group("/ws/v2")
	wsV2.GET("/:dongle_id", requireCookieAuth(config), websocket.CreateHandler(websocketControllers.CreateEventsWebsocket(eventsChannel), config))
}
