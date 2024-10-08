package v1dot4

import (
	"bufio"
	"encoding/binary"
	"errors"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/USA-RedDragon/rtz-server/internal/config"
	"github.com/USA-RedDragon/rtz-server/internal/db/models"
	"github.com/USA-RedDragon/rtz-server/internal/logparser"
	v1dot4 "github.com/USA-RedDragon/rtz-server/internal/server/apimodels/v1.4"
	"github.com/USA-RedDragon/rtz-server/internal/storage"
	"github.com/gin-gonic/gin"
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
	storage, ok := c.MustGet("storage").(storage.Storage)
	if !ok {
		slog.Error("Failed to get storage from context")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}
	logQueue, ok := c.MustGet("logQueue").(*logparser.LogQueue)
	if !ok {
		slog.Error("Failed to get log queue from context")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}
	device, err := models.FindDeviceByDongleID(db, dongleID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}

	path := c.Query("path")
	if path == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "path is required"})
		return
	}

	if !fs.ValidPath(path) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid path"})
		return
	}

	err = storage.Mkdir(dongleID, 0755)
	if err != nil && !errors.Is(err, fs.ErrExist) {
		slog.Error("Failed to create dongle directory", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}

	base, err := storage.Sub(dongleID)
	if err != nil {
		slog.Error("Failed to get base storage", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}
	defer base.Close()

	err = base.MkdirAll(filepath.Dir(path), 0755)
	if err != nil {
		slog.Error("Failed to create directories", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}

	fileReader := bufio.NewReader(c.Request.Body)
	f, err := base.Create(path)
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

	switch {
	case strings.Contains(path, "boot/"):
		// Boot log
		err = db.Create(&models.BootLog{
			DeviceID: device.ID,
			FileName: filepath.Base(path),
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
			FileName: filepath.Base(path),
		}).Error
		if err != nil {
			slog.Error("Failed to create crash log", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
			return
		}
	case newRouteRegex.Match([]byte(path)):
		slog.Warn("New route upload", "path", path)
		match := newRouteRegex.FindStringSubmatch(path)
		var result v1dot4.RouteInfo
		for i, name := range newRouteRegex.SubexpNames() {
			switch name {
			case "motonic":
				result.Motonic = match[i]
			case "route":
				result.Route = match[i]
			case "segment":
				result.Segment = match[i]
			}
		}

		// Verify file type
		switch {
		case strings.Contains(path, "qlog.bz2"):
			file, err := base.Open(path)
			if err != nil {
				slog.Error("Failed to open file", "error", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
				return
			}
			header := make([]byte, 3)
			read, err := file.Read(header)
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
			file.Close()
			go logQueue.AddLog(path, dongleID, result)
		case strings.Contains(path, "qlog.zst"):
			file, err := base.Open(path)
			if err != nil {
				slog.Error("Failed to open file", "error", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
				return
			}
			var magic uint32
			err = binary.Read(file, binary.LittleEndian, &magic)
			if err != nil {
				slog.Error("Failed to read magic", "error", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
				return
			}
			// 0xFD2FB528 is the magic number for zstd
			// https://github.com/facebook/zstd/blob/dev/doc/zstd_compression_format.md#zstandard-frames
			if magic != 0xFD2FB528 {
				slog.Error("Invalid magic", "magic", magic)
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid file"})
				return
			}
			file.Close()
			go logQueue.AddLog(path, dongleID, result)
		}
	case oldRouteRegex.Match([]byte(path)):
		slog.Warn("Old route upload", "path", path)
	default:
		slog.Warn("Got unknown upload path", "path", path)
	}

	c.JSON(http.StatusOK, gin.H{})
}
