package v1

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/USA-RedDragon/rtz-server/internal/db/models"
	v1 "github.com/USA-RedDragon/rtz-server/internal/server/apimodels/v1"
	"github.com/USA-RedDragon/rtz-server/internal/utils"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func GETRouteQCameraM3U8(c *gin.Context) {
	id, ok := c.Params.Get("id")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id is required"})
		return
	}
	idParts := strings.Split(id, "|")
	if len(idParts) != 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id must be in the format of <device_id>|<route>"})
		return
	}
	deviceID := idParts[0]
	if deviceID == "1d3dc3e03047b0c7" {
		url := c.Request.URL
		url.Host = "api.comma.ai"
		url.Scheme = "https"
		resp, err := utils.HTTPRequest(c, http.MethodGet, url.String(), nil, nil)
		if err != nil {
			slog.Error("GETRouteQCameraM3U8", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			slog.Error("GETRouteQCameraM3U8", "status_code", resp.StatusCode)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
			return
		}

		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			slog.Error("GETRouteQCameraM3U8", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
			return
		}

		c.Data(http.StatusOK, "application/x-mpegURL", bodyBytes)
		return
	}
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Not implemented"})
}

func GETMe(c *gin.Context) {
	_, ok := c.Get("demo")
	if ok {
		c.JSON(http.StatusOK, v1.GETMeResponse{
			Email:          "comma.connect.user@gmail.com",
			ID:             "0decddcfdf241a60",
			Prime:          false,
			RegisteredDate: 1716959966,
			Superuser:      false,
			UserID:         "google_115606701206535685614",
		})
		return
	}
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
	_, ok := c.Get("demo")
	if ok {
		c.JSON(http.StatusOK, []v1.GETMyDevicesResponse{{
			Device: models.Device{
				DeviceType:     "threex",
				DongleID:       "1d3dc3e03047b0c7",
				IsPaired:       true,
				LastAthenaPing: 0,
				Prime:          false,
				PrimeType:      0,
				PublicKey:      "-----BEGIN PUBLIC KEY-----MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAvd6w9111wPVAgzrZyIhrX/mQm5uoSD9rDQxlWJemaYqKoREBwO6Hvs12PtWa0eXMa/1ZJugblXMG4oWqoswyLQ5QOqVNNWTcdE8avLtcW5QP+DzbCzUW7nVLUF9UgDUvsCjd95E5o/qEpsTV7NIisjJr+xhO7HXBdqVwmee5fUmgWI3/yHMMptT5kD1ZpmgTjDJqLZP7g78dpSZ8uc7NSLoI5fkaTrJU6HiY1vbVcQLe1IEOMEqW0QdxaRhA2Jr5OV3Hd9zYdGMvh/wYFX14ZYG2dYSKHXj9hlTbiMxiBuLq2hjrEC+Bfv1lHploFxmr3fGz7Sup0fqCQSjwpQI9qQIDAQAB-----END PUBLIC KEY-----",
				Serial:         "c0ffee0",
				TrialClaimed:   false,
			},
			Alias:   "demo 3x",
			IsOwner: false,
			EligibleFeatures: v1.EligibleFeatures{
				Navigation: true,
				Prime:      true,
				PrimeData:  true,
			},
		}})
		return
	}

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
