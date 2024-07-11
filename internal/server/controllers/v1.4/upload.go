package v1dot4

import (
	"bufio"
	"compress/bzip2"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/USA-RedDragon/rtz-server/internal/config"
	"github.com/USA-RedDragon/rtz-server/internal/db/models"
	"github.com/USA-RedDragon/rtz-server/internal/logparser"
	v1dot4 "github.com/USA-RedDragon/rtz-server/internal/server/apimodels/v1.4"
	"github.com/gin-gonic/gin"
	"github.com/mattn/go-nulltype"
	"gorm.io/gorm"
)

var (
	newRouteRegex = regexp.MustCompile(`^(?P<motonic>[0-9a-fA-F]+)--(?P<route>[0-9a-fA-F]+)--(?P<segment>\d+)`)
	oldRouteRegex = regexp.MustCompile(`^(?P<date>\d{4}-\d{2}-\d{2})--(?P<time>\d{2}-\d{2}-\d{2})`)
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
	case newRouteRegex.Match([]byte(path)):
		slog.Warn("New route upload", "path", path)
		if strings.Contains(path, "qlog.bz2") {
			file, err := os.Open(cleanedAbsolutePath)
			if err != nil {
				slog.Error("Failed to open file", "error", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
				return
			}
			defer file.Close()
			header := make([]byte, 3)
			read, err := file.ReadAt(header, 0)
			if err != nil {
				slog.Error("Failed to read header", "error", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
				return
			}
			if read != 3 || string(header) != "BZh" {
				slog.Error("Invalid header", "header", string(header))
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid file"})
				return
			}
			segmentData, err := logparser.DecodeSegmentData(bzip2.NewReader(bufio.NewReader(file)))
			if err != nil {
				slog.Error("Failed to decode segment data", "error", err)
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid file"})
				return
			}
			slog.Info("Got segment data", "data", len(segmentData.GPSLocations))
			if (!device.LastGPSTime.Valid() || segmentData.LatestTimestamp > uint64(device.LastGPSTime.TimeValue().UnixNano())) && len(segmentData.GPSLocations) > 0 {
				latestTimeStamp := time.Unix(0, int64(segmentData.LatestTimestamp))
				err := db.Model(&device).
					Updates(models.Device{
						LastGPSTime: nulltype.NullTimeOf(latestTimeStamp),
						LastGPSLat:  nulltype.NullFloat64Of(segmentData.GPSLocations[len(segmentData.GPSLocations)-1].Latitude),
						LastGPSLng:  nulltype.NullFloat64Of(segmentData.GPSLocations[len(segmentData.GPSLocations)-1].Longitude),
					}).Error
				if err != nil {
					slog.Error("Failed to update device", "error", err)
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
					return
				}
			}
		}
	case oldRouteRegex.Match([]byte(path)):
		slog.Warn("Old route upload", "path", path)
	default:
		slog.Warn("Got unknown upload path", "path", path)
	}

	c.JSON(http.StatusOK, gin.H{})
}
