package v1dot1

import (
	"log/slog"
	"net/http"

	"github.com/USA-RedDragon/connect-server/internal/db/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func GETDevice(c *gin.Context) {
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
}

func GETDeviceStats(c *gin.Context) {
	slog.Info("Get Stats", "url", c.Request.URL.String())
}
