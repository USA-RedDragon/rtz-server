package v1dot1

import (
	"log/slog"
	"net/http"

	"github.com/USA-RedDragon/rtz-server/internal/db/models"
	v1dot1 "github.com/USA-RedDragon/rtz-server/internal/server/apimodels/v1.1"
	"github.com/gin-gonic/gin"
	"github.com/mattn/go-nulltype"
	"gorm.io/gorm"
)

func GETDevice(c *gin.Context) {
	_, ok := c.Get("demo")
	if ok {
		c.JSON(http.StatusOK, models.Device{
			Alias:          nulltype.NullStringOf("demo 3x"),
			DeviceType:     "threex",
			DongleID:       "1d3dc3e03047b0c7",
			IsPaired:       true,
			LastAthenaPing: 0,
			Prime:          false,
			PrimeType:      0,
			PublicKey:      "-----BEGIN PUBLIC KEY-----\nMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAvd6w9111wPVAgzrZyIhr\nX/mQm5uoSD9rDQxlWJemaYqKoREBwO6Hvs12PtWa0eXMa/1ZJugblXMG4oWqoswy\nLQ5QOqVNNWTcdE8avLtcW5QP+DzbCzUW7nVLUF9UgDUvsCjd95E5o/qEpsTV7NIi\nsjJr+xhO7HXBdqVwmee5fUmgWI3/yHMMptT5kD1ZpmgTjDJqLZP7g78dpSZ8uc7N\nSLoI5fkaTrJU6HiY1vbVcQLe1IEOMEqW0QdxaRhA2Jr5OV3Hd9zYdGMvh/wYFX14\nZYG2dYSKHXj9hlTbiMxiBuLq2hjrEC+Bfv1lHploFxmr3fGz7Sup0fqCQSjwpQI9\nqQIDAQAB\n-----END PUBLIC KEY-----\n",
			Serial:         "c0ffee0",
			TrialClaimed:   false,
		})
		return
	}

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

	c.JSON(http.StatusOK, device)
}

func GETDeviceStats(c *gin.Context) {
	_, ok := c.Get("demo")
	if ok {
		c.JSON(http.StatusOK, v1dot1.StatsResponse{
			All: v1dot1.Stats{
				Distance: 77.92072987556458,
				Minutes:  174,
				Routes:   12,
			},
			Week: v1dot1.Stats{
				Distance: 0,
				Minutes:  0,
				Routes:   0,
			},
		})
	}
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

	slog.Info("Get Stats", "url", c.Request.URL.String(), "device", device.DongleID)

	// TODO: Implement stats

	c.JSON(http.StatusOK, v1dot1.StatsResponse{})
}
