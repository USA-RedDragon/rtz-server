package v1dot4

import (
	"log/slog"

	"github.com/gin-gonic/gin"
)

func GETUploadURL(c *gin.Context) {
	slog.Info("Get Upload URL", "url", c.Request.URL.String())
}
