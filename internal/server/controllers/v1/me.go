package v1

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/USA-RedDragon/rtz-server/internal/db/models"
	v1 "github.com/USA-RedDragon/rtz-server/internal/server/apimodels/v1"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func GETMe(c *gin.Context) {
	user, ok := c.MustGet("user").(*models.User)
	if !ok {
		slog.Error("Failed to get user from context")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}

	userResp := v1.GETMeResponse{
		Email:          "no emails here",
		ID:             fmt.Sprintf("%d", user.ID),
		Prime:          true,
		RegisteredDate: uint(user.CreatedAt.Unix()),
		Superuser:      user.Superuser,
	}

	switch {
	case user.GitHubUserID.Valid():
		userResp.UserID = fmt.Sprintf("github_%d", user.GitHubUserID.Int64Value())
	case user.GoogleUserID.Valid():
		userResp.UserID = "google_" + user.GoogleUserID.StringValue()
	case user.CustomUserID.Valid():
		userResp.UserID = fmt.Sprintf("custom_%d", user.CustomUserID.Int64Value())
	}

	c.JSON(http.StatusOK, userResp)
}

func GETMyDevices(c *gin.Context) {
	user, ok := c.MustGet("user").(*models.User)
	if !ok {
		slog.Error("Failed to get user from context")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}

	db, ok := c.MustGet("db").(*gorm.DB)
	if !ok {
		slog.Error("Failed to get db from context")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}

	devices, err := models.GetDevicesOwnedByUser(db, user.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusOK, []v1.GETMyDevicesResponse{})
			return
		}
		slog.Error("Failed to get devices", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}

	devicesResp := []v1.GETMyDevicesResponse{}
	for _, device := range devices {
		devicesResp = append(devicesResp, v1.GETMyDevicesResponse{
			Device: device,
			Alias:  device.Alias.StringValue(),
			EligibleFeatures: v1.EligibleFeatures{
				Navigation: true,
				Prime:      true,
				PrimeData:  true,
			},
			IsOwner: true,
		})
	}

	sharedDevices, err := models.ListSharedToByUserID(db, user.ID)
	if err != nil {
		slog.Error("Failed to get shared devices", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}
	for _, sharedDevice := range sharedDevices {
		device, err := models.FindDeviceByID(db, sharedDevice.DeviceID)
		if err != nil {
			slog.Error("Failed to get shared device", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
			return
		}
		devicesResp = append(devicesResp, v1.GETMyDevicesResponse{
			Device: device,
			Alias:  device.Alias.StringValue(),
			EligibleFeatures: v1.EligibleFeatures{
				Navigation: true,
				Prime:      true,
				PrimeData:  true,
			},
			IsOwner: false,
		})
	}

	c.JSON(http.StatusOK, devicesResp)
}
