package server

import (
	"log/slog"
	"net/http"

	"github.com/USA-RedDragon/connect-server/internal/config"
	"github.com/USA-RedDragon/connect-server/internal/events"
	controllersV1 "github.com/USA-RedDragon/connect-server/internal/server/controllers/v1"
	controllersV1dot1 "github.com/USA-RedDragon/connect-server/internal/server/controllers/v1.1"
	controllersV1dot4 "github.com/USA-RedDragon/connect-server/internal/server/controllers/v1.4"
	controllersV2 "github.com/USA-RedDragon/connect-server/internal/server/controllers/v2"
	websocketControllers "github.com/USA-RedDragon/connect-server/internal/server/websocket"
	"github.com/USA-RedDragon/connect-server/internal/websocket"
	"github.com/gin-gonic/gin"
)

func applyRoutes(r *gin.Engine, config *config.Config, eventsChannel chan events.Event) {
	r.GET("/health", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	apiV1 := r.Group("/v1")
	v1(apiV1, config)

	apiV1dot1 := r.Group("/v1.1")
	v1dot1(apiV1dot1, config)

	apiV1dot4 := r.Group("/v1.4")
	v1dot4(apiV1dot4, config)

	apiV2 := r.Group("/v2")
	v2(apiV2, config)

	r.NoRoute(func(c *gin.Context) {
		slog.Warn("Not Found", "path", c.Request.URL.Path)
		c.JSON(http.StatusNotFound, gin.H{"error": "Not Found"})
	})

	wsV2 := r.Group("/ws/v2")
	wsV2.GET("/:dongle_id", setDevice(), requireCookieAuth(config), websocket.CreateHandler(websocketControllers.CreateEventsWebsocket(eventsChannel), config))
}

func v1(group *gin.RouterGroup, config *config.Config) {
	group.GET("/me", requireJWTAuth(config), controllersV1.GETMe)
	group.GET("/me/devices", requireJWTAuth(config), controllersV1.GETMyDevices)
	group.GET("/navigation/:dongle_id/next", setDevice(), requireAuth(config), controllersV1.GETNavigationNext)
	group.DELETE("/navigation/:dongle_id/next", setDevice(), requireAuth(config), controllersV1.DELETENavigationNext)
	group.GET("/navigation/:dongle_id/locations", setDevice(), requireAuth(config), controllersV1.GETNavigationLocations)
}

func v1dot1(group *gin.RouterGroup, config *config.Config) {
	group.GET("/devices/:dongle_id", setDevice(), requireAuth(config), controllersV1dot1.GETDevice)
	group.GET("/devices/:dongle_id/stats", setDevice(), requireAuth(config), controllersV1dot1.GETDeviceStats)
}

func v1dot4(group *gin.RouterGroup, config *config.Config) {
	group.GET("/:dongle_id/upload_url", setDevice(), requireAuth(config), controllersV1dot4.GETUploadURL)
}

func v2(group *gin.RouterGroup, config *config.Config) {
	group.POST("/auth", controllersV2.POSTAuth)
	group.GET("/auth/g/redirect", controllersV2.GETGoogleRedirect)
	group.GET("/auth/h/redirect", controllersV2.GETGitHubRedirect)
	group.POST("/pilotpair", requireJWTAuth(config), controllersV2.POSTPilotPair)
	group.POST("/pilotauth", controllersV2.POSTPilotAuth)
}
