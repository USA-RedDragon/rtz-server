package v1

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/USA-RedDragon/connect-server/internal/db/models"
	v1 "github.com/USA-RedDragon/connect-server/internal/server/apimodels/v1"
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

	if user.GitHubUserID != 0 {
		userResp.UserID = fmt.Sprintf("%d", user.GitHubUserID)
	} else {
		userResp.UserID = user.GoogleUserID
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
			EligibleFeatures: v1.EligibleFeatures{
				Navigation: true,
				Prime:      true,
				PrimeData:  true,
			},
			IsOwner: true,
		})
	}

	c.JSON(http.StatusOK, devicesResp)
}
