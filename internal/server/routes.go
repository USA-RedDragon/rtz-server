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

func applyRoutes(r *gin.Engine, config *config.HTTP, eventsChannel chan events.Event) {
	r.GET("/health", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
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
