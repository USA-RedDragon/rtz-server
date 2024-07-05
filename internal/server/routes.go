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
	wsV2.GET("/:dongle_id", requireCookieAuth(config), websocket.CreateHandler(websocketControllers.CreateEventsWebsocket(eventsChannel), config))
}

func v1(group *gin.RouterGroup, config *config.Config) {
	group.GET("/me", requireAuth(config, AuthTypeUser), controllersV1.GETMe)
	group.GET("/me/devices", requireAuth(config, AuthTypeUser), controllersV1.GETMyDevices)
	group.GET("/navigation/:dongle_id/next", requireAuth(config, AuthTypeUser|AuthTypeDevice), controllersV1.GETNavigationNext)
	group.DELETE("/navigation/:dongle_id/next", requireAuth(config, AuthTypeUser|AuthTypeDevice), controllersV1.DELETENavigationNext)
	group.GET("/navigation/:dongle_id/locations", requireAuth(config, AuthTypeUser|AuthTypeDevice), controllersV1.GETNavigationLocations)
	group.GET("/prime/subscription", requireAuth(config, AuthTypeUser), controllersV1.GETPrimeSubscription)
}

func v1dot1(group *gin.RouterGroup, config *config.Config) {
	group.GET("/devices/:dongle_id", requireAuth(config, AuthTypeUser|AuthTypeDevice), controllersV1dot1.GETDevice)
	group.GET("/devices/:dongle_id/stats", requireAuth(config, AuthTypeUser|AuthTypeDevice), controllersV1dot1.GETDeviceStats)
}

func v1dot4(group *gin.RouterGroup, config *config.Config) {
	group.GET("/:dongle_id/upload_url", requireAuth(config, AuthTypeUser|AuthTypeDevice), controllersV1dot4.GETUploadURL)
}

func v2(group *gin.RouterGroup, config *config.Config) {
	group.POST("/auth", controllersV2.POSTAuth)
	group.GET("/auth/g/redirect", controllersV2.GETGoogleRedirect)
	group.GET("/auth/h/redirect", controllersV2.GETGitHubRedirect)
	group.POST("/pilotpair", requireAuth(config, AuthTypeUser), controllersV2.POSTPilotPair)
	group.POST("/pilotauth", controllersV2.POSTPilotAuth)
}
