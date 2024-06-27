package server

import (
	"log/slog"
	"net/http"

	"github.com/USA-RedDragon/connect-server/internal/config"
	"github.com/USA-RedDragon/connect-server/internal/events"
	websocketControllers "github.com/USA-RedDragon/connect-server/internal/server/websocket"
	"github.com/USA-RedDragon/connect-server/internal/websocket"
	"github.com/gin-gonic/gin"
)

func applyRoutes(r *gin.Engine, config *config.Config, eventsChannel chan events.Event) {
	r.GET("/health", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	apiV1 := r.Group("/v1")
	apiV1.GET("/navigation/:dongle_id/next", func(c *gin.Context) {
		slog.Info("Get Next Navigation", "url", c.Request.URL.String())
	})

	apiV1.DELETE("/navigation/:dongle_id/next", func(c *gin.Context) {
		slog.Info("Delete Next Navigation", "url", c.Request.URL.String())
	})

	apiV1.GET("/navigation/:dongle_id/locations", func(c *gin.Context) {
		slog.Info("Get Locations", "url", c.Request.URL.String())
	})

	apiV11 := r.Group("/v1.1")
	apiV11.GET("/devices/:dongle_id/", func(c *gin.Context) {
		slog.Info("Get Device", "url", c.Request.URL.String())
	})

	apiV11.GET("/devices/:dongle_id/stats", func(c *gin.Context) {
		slog.Info("Get Stats", "url", c.Request.URL.String())
	})

	apiV14 := r.Group("/v1.4")
	apiV14.GET("/:dongle_id/upload_url", func(c *gin.Context) {
		slog.Info("Get Upload URL", "url", c.Request.URL.String())
	})

	apiV2 := r.Group("/v2")
	apiV2.POST("/pilotauth", func(c *gin.Context) {
		slog.Info("Pilot Auth", "url", c.Request.URL.String())
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	r.NoRoute(func(c *gin.Context) {
		slog.Warn("Not Found", "path", c.Request.URL.Path)
		c.JSON(http.StatusNotFound, gin.H{"error": "Not Found"})
	})

	wsV2 := r.Group("/ws/v2")
	wsV2.GET("/:dongle_id", websocket.CreateHandler(websocketControllers.CreateEventsWebsocket(eventsChannel), config))
}
