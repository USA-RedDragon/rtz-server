package v1

import (
	"log/slog"

	"github.com/gin-gonic/gin"
)

func GETNavigationNext(c *gin.Context) {
	slog.Info("Get Next Navigation", "url", c.Request.URL.String())
}

func DELETENavigationNext(c *gin.Context) {
	slog.Info("Delete Next Navigation", "url", c.Request.URL.String())
}

func GETNavigationLocations(c *gin.Context) {
	slog.Info("Get Locations", "url", c.Request.URL.String())
}
