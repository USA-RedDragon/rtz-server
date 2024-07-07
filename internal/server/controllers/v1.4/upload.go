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

	if !strings.HasPrefix(cleanedAbsolutePath, basePath) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid path"})
		return
	}

	err = os.MkdirAll(filepath.Dir(cleanedAbsolutePath), 0755)
	if err != nil {
		slog.Error("Failed to create directories", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}

	fileReader := bufio.NewReader(c.Request.Body)
	tmpFile := cleanedAbsolutePath + ".tmp"
	f, err := os.Create(tmpFile)
	if err != nil {
		slog.Error("Failed to create file", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}
	w := bufio.NewWriter(f)
	_, err = io.Copy(w, fileReader)
	if err != nil {
		slog.Error("Failed to write file", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}

	err = w.Flush()
	if err != nil {
		slog.Error("Failed to flush file", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}

	err = f.Close()
	if err != nil {
		slog.Error("Failed to close file", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}

	err = os.Rename(tmpFile, cleanedAbsolutePath)
	if err != nil {
		slog.Error("Failed to rename file", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}

	switch {
	case strings.Contains(path, "boot/"):
		// Boot log
		err = db.Create(&models.BootLog{
			DeviceID: device.ID,
			FileName: filepath.Base(cleanedAbsolutePath),
		}).Error
		if err != nil {
			slog.Error("Failed to create boot log", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
			return
		}
	case strings.Contains(path, "crash/"):
		// Crash log
		err = db.Create(&models.CrashLog{
			DeviceID: device.ID,
			FileName: filepath.Base(cleanedAbsolutePath),
		}).Error
		if err != nil {
			slog.Error("Failed to create crash log", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
			return
		}
	default:
		slog.Warn("Got unknown upload path", "path", path)
	}

	c.JSON(http.StatusOK, gin.H{})
}
