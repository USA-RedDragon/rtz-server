package server

import (
	"log/slog"
	"net/http"

	"github.com/USA-RedDragon/connect-server/internal/config"
	"github.com/USA-RedDragon/connect-server/internal/server/controllers"
	controllersV1 "github.com/USA-RedDragon/connect-server/internal/server/controllers/v1"
	controllersV1dot1 "github.com/USA-RedDragon/connect-server/internal/server/controllers/v1.1"
	controllersV1dot4 "github.com/USA-RedDragon/connect-server/internal/server/controllers/v1.4"
	controllersV2 "github.com/USA-RedDragon/connect-server/internal/server/controllers/v2"
	websocketControllers "github.com/USA-RedDragon/connect-server/internal/server/websocket"
	"github.com/USA-RedDragon/connect-server/internal/websocket"
	"github.com/gin-gonic/gin"
)

func applyRoutes(r *gin.Engine, config *config.Config, rpcWebsocket *websocketControllers.RPCWebsocket) {
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
	wsV2.GET("/:dongle_id", requireCookieAuth(config), websocket.CreateHandler(rpcWebsocket, config))

	r.POST("/:dongle_id", requireAuth(config, AuthTypeUser), requireDeviceOwnerOrShared(), controllers.HandleRPC)
}

func v1(group *gin.RouterGroup, config *config.Config) {
	group.GET("/me", requireAuth(config, AuthTypeUser), controllersV1.GETMe)
	group.PATCH("/devices/:dongle_id", requireAuth(config, AuthTypeUser), requireDeviceOwner(), controllersV1.PATCHDevice)
	group.POST("/devices/:dongle_id/unpair", requireAuth(config, AuthTypeUser), requireDeviceOwner(), controllersV1.POSTDeviceUnpair)
	group.POST("/devices/:dongle_id/add_user", requireAuth(config, AuthTypeUser), requireDeviceOwner(), controllersV1.POSTDeviceAddUser)
	group.GET("/devices/:dongle_id/location", requireAuth(config, AuthTypeUser), requireDeviceOwnerOrShared(), controllersV1.GETDeviceLocation)
	group.GET("/devices/:dongle_id/routes_segments", requireAuth(config, AuthTypeUser), requireDeviceOwnerOrShared(), controllersV1.GETDeviceRoutesSegments)
	group.GET("/me/devices", requireAuth(config, AuthTypeUser), controllersV1.GETMyDevices)
	group.POST("/navigation/:dongle_id/set_destination", requireAuth(config, AuthTypeUser|AuthTypeDevice), requireDeviceOwnerOrShared(), controllersV1.POSTSetDestination)
	group.GET("/navigation/:dongle_id/next", requireAuth(config, AuthTypeUser|AuthTypeDevice), requireDeviceOwnerOrShared(), controllersV1.GETNavigationNext)
	group.PUT("/navigation/:dongle_id/locations", requireAuth(config, AuthTypeUser|AuthTypeDevice), requireDeviceOwnerOrShared(), controllersV1.PUTNavigationLocations)
	group.DELETE("/navigation/:dongle_id/next", requireAuth(config, AuthTypeUser|AuthTypeDevice), requireDeviceOwnerOrShared(), controllersV1.DELETENavigationNext)
	group.GET("/navigation/:dongle_id/locations", requireAuth(config, AuthTypeUser|AuthTypeDevice), requireDeviceOwnerOrShared(), controllersV1.GETNavigationLocations)
	group.GET("/prime/subscription", requireAuth(config, AuthTypeUser), controllersV1.GETPrimeSubscription)
}

func v1dot1(group *gin.RouterGroup, config *config.Config) {
	group.GET("/devices/:dongle_id", requireAuth(config, AuthTypeUser|AuthTypeDevice), requireDeviceOwnerOrShared(), controllersV1dot1.GETDevice)
	group.GET("/devices/:dongle_id/stats", requireAuth(config, AuthTypeUser|AuthTypeDevice), requireDeviceOwnerOrShared(), controllersV1dot1.GETDeviceStats)
}

func v1dot4(group *gin.RouterGroup, config *config.Config) {
	group.GET("/:dongle_id/upload_url", requireAuth(config, AuthTypeDevice), controllersV1dot4.GETUploadURL)
	group.PUT("/:dongle_id/upload", requireAuth(config, AuthTypeDevice), controllersV1dot4.PUTUpload)
}

func v2(group *gin.RouterGroup, config *config.Config) {
	group.POST("/auth", controllersV2.POSTAuth)
	group.GET("/auth/g/redirect", controllersV2.GETGoogleRedirect)
	group.GET("/auth/h/redirect", controllersV2.GETGitHubRedirect)
	group.POST("/pilotpair", requireAuth(config, AuthTypeUser), controllersV2.POSTPilotPair)
	group.POST("/pilotauth", controllersV2.POSTPilotAuth)
}
