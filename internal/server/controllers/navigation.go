package controllers

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/USA-RedDragon/connect-server/internal/config"
	"github.com/USA-RedDragon/connect-server/internal/utils"
	"github.com/gin-gonic/gin"
)

func GETMapboxDirections(c *gin.Context) {
	config, ok := c.MustGet("config").(*config.Config)
	if !ok {
		slog.Error("Failed to get config from context")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}
	coordsParam := c.Param("coords")
	coordPairs := strings.Split(coordsParam, ";")
	if len(coordPairs) != 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "coords must be 2 semicolon-separated coordinate pairs"})
		return
	}
	sourceCoords := strings.Split(coordPairs[0], ",")
	destCoords := strings.Split(coordPairs[1], ",")
	if len(sourceCoords) != 2 || len(destCoords) != 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "coords must be 2 comma-separated values"})
		return
	}
	query := c.Request.URL.Query()
	query.Set("access_token", config.Mapbox.PublicToken)
	url := c.Request.URL
	url.Host = "api.mapbox.com"
	url.RawQuery = query.Encode()

	resp, err := utils.HTTPRequest(http.MethodGet, url.String(), nil, nil)
	if err != nil {
		slog.Error("GETMapboxDirections", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		slog.Error("GETMapboxDirections", "status_code", resp.StatusCode)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Error("GETMapboxDirections", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}

	var jsonyResp map[string]any
	if err := json.Unmarshal(bodyBytes, &jsonyResp); err != nil {
		slog.Error("GETMapboxDirections", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}

	c.JSON(http.StatusOK, jsonyResp)
}
