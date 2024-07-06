package v1

import (
	"log/slog"
	"net/http"

	"github.com/USA-RedDragon/connect-server/internal/db/models"
	v1 "github.com/USA-RedDragon/connect-server/internal/server/apimodels/v1"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func POSTDeviceUnpair(c *gin.Context) {
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

	err = db.Model(&device).Update("owner_id", nil).Update("is_paired", false).Error
	if err != nil {
		slog.Error("Failed to unpair device", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": 1})
}

func GETDeviceLocation(c *gin.Context) {
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

	resp := v1.LocationResponse{
		DongleID: device.DongleID,
		Lat:      device.LastGPSLat.Float64Value(),
		Lon:      device.LastGPSLng.Float64Value(),
	}
	if device.LastGPSTime.Valid() {
		resp.Time = device.LastGPSTime.TimeValue().UnixMilli()
	} else {
		resp.Time = 0
	}

	c.JSON(http.StatusOK, resp)
}

func GETDeviceRoutesSegments(c *gin.Context) {
	c.JSON(http.StatusOK, []int{})
}
