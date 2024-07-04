package v1

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/USA-RedDragon/connect-server/internal/db/models"
	v1 "github.com/USA-RedDragon/connect-server/internal/server/apimodels/v1"
	"github.com/gin-gonic/gin"
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
