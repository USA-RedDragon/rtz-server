package v1

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/USA-RedDragon/connect-server/internal/db/models"
	v1 "github.com/USA-RedDragon/connect-server/internal/server/apimodels/v1"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func PATCHDevice(c *gin.Context) {
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

	var req v1.DevicePatchable
	if err := c.BindJSON(&req); err != nil {
		slog.Error("Failed to bind request", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	err = db.Model(&device).Update("alias", req.Alias).Error
	if err != nil {
		slog.Error("Failed to update device", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}

	device.Alias = req.Alias

	c.JSON(http.StatusOK, device)
}

func POSTDeviceAddUser(c *gin.Context) {
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

	var req v1.AddUserRequest
	if err := c.BindJSON(&req); err != nil {
		slog.Error("Failed to bind request", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	owner, ok := c.MustGet("user").(*models.User)
	if !ok {
		slog.Error("Failed to get user from context")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}

	// Search for the user by Google ID first
	var user models.User
	user, err = models.FindUserByGoogleID(db, req.Email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Convert the req.Email to an integer, then look for GitHub ID
			ghID, err := strconv.Atoi(req.Email)
			if err != nil {
				slog.Error("Failed to convert email to int", "error", err)
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
				return
			}
			user, err = models.FindUserByGitHubID(db, ghID)
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
					return
				}
				slog.Error("Failed to find user by GitHub ID", "error", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
				return
			}
		} else {
			slog.Error("Failed to find user by Google ID", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
			return
		}
	}

	err = db.Model(&models.DeviceShare{}).Where("device_id = ? AND shared_to_user_id = ?", device.ID, user.ID).FirstOrCreate(&models.DeviceShare{
		DeviceID:       device.ID,
		SharedToUserID: user.ID,
		OwnerID:        owner.ID,
	}).Error
	if err != nil {
		slog.Error("Failed to share device", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": 1})
}

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
