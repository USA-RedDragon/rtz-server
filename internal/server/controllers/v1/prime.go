package v1

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/USA-RedDragon/rtz-server/internal/db/models"
	v1 "github.com/USA-RedDragon/rtz-server/internal/server/apimodels/v1"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func GETPrimeSubscription(c *gin.Context) {
	dongleID, ok := c.GetQuery("dongle_id")
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

	owner, err := models.FindUserByID(db, device.OwnerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}

	var id string
	if owner.GoogleUserID.Valid() {
		id = owner.GoogleUserID.String()
	} else {
		id = fmt.Sprintf("%d", owner.GitHubUserID.Int64Value())
	}
	c.JSON(http.StatusOK, v1.PrimeSubscriptionResponse{
		IsPrimeSim:        false,
		Plan:              "free",
		RequiresMigration: false,
		SubscribedAt:      uint(device.CreatedAt.Unix()),
		UserID:            id,
	})
}
