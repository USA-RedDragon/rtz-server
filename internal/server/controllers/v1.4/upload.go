package v1dot4

import (
	"bufio"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/USA-RedDragon/connect-server/internal/config"
	"github.com/USA-RedDragon/connect-server/internal/db/models"
	v1dot4 "github.com/USA-RedDragon/connect-server/internal/server/apimodels/v1.4"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func GETUploadURL(c *gin.Context) {
	dongleID, ok := c.Params.Get("dongle_id")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "dongle_id is required"})
		return
	}

	config, ok := c.MustGet("config").(*config.Config)
	if !ok {
		slog.Error("Failed to get config from context")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}

	path := c.Query("path")
	if path == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "path is required"})
		return
	}

	c.JSON(http.StatusOK, v1dot4.UploadURLResponse{
		URL: config.HTTP.BackendURL + "/v1.4/" + dongleID + "/upload?path=" + url.QueryEscape(path),
		Headers: map[string]string{
			"Authorization": c.GetHeader("Authorization"),
		},
	})
}

func PUTUpload(c *gin.Context) {
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

	config, ok := c.MustGet("config").(*config.Config)
	if !ok {
		slog.Error("Failed to get config from context")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}

	path := c.Query("path")
	if path == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "path is required"})
		return
	}

	basePath, err := filepath.Abs(filepath.Join(config.Persistence.Uploads, dongleID))
	if err != nil {
		slog.Error("Failed to get base path", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}

	// SAFETY! We must place these files under `config.Persistence.Uploads`+dongleID+path
	// There can be NO upwards traversal in the path
	cleanedAbsolutePath, err := filepath.Abs(filepath.Join(config.Persistence.Uploads, dongleID, path))
	if err != nil {
		slog.Error("Failed to get absolute path", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}

	slog.Info("Upload file", "device", device.DongleID, "path", path, "basePath", basePath, "absolute_path", cleanedAbsolutePath)

	if !strings.HasPrefix(cleanedAbsolutePath, basePath) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid path"})
		return
	}

	fileReader := bufio.NewReader(c.Request.Body)
	f, err := os.Create(cleanedAbsolutePath)
	if err != nil {
		slog.Error("Failed to create file", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	written, err := io.Copy(w, fileReader)
	if err != nil {
		slog.Error("Failed to write file", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}
	slog.Debug("Uploaded file", "path", cleanedAbsolutePath, "size", written, "device", device.DongleID, "path", path)
	c.JSON(http.StatusOK, gin.H{})
}
