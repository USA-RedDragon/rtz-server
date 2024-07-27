package controllers

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/USA-RedDragon/rtz-server/internal/config"
	"github.com/USA-RedDragon/rtz-server/internal/utils"
	"github.com/gin-gonic/gin"
)

//nolint:golint,gosec
const commaStyleToken = "cGsuZXlKMUlqb2lZMjl0YldGaGFTSXNJbUVpT2lKamFuZ3lZWFYwYzIwd01HVTJORGx1TVdSNGFtVXlkR2w1SW4wLjZWYjExUzZ0ZFg2QXJwajZ0clJFX2c"
const mapboxAPIHost = "api.mapbox.com"
const schemeHTTPS = "https"

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
	url.Host = mapboxAPIHost
	url.Scheme = schemeHTTPS
	url.RawQuery = query.Encode()

	resp, err := utils.HTTPRequest(c, http.MethodGet, url.String(), nil, nil)
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

func GETMapboxStyle(c *gin.Context) {
	owner := c.Param("owner")
	if owner == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "owner is required"})
		return
	}

	styleID := c.Param("styleID")
	if styleID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "styleID is required"})
		return
	}

	query := c.Request.URL.Query()
	if owner == "commaai" {
		token, err := base64.RawStdEncoding.DecodeString(commaStyleToken)
		if err != nil {
			slog.Error("GETMapboxStyle", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
			return
		}
		query.Set("access_token", string(token))
	} else {
		config, ok := c.MustGet("config").(*config.Config)
		if !ok {
			slog.Error("Failed to get config from context")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
			return
		}

		query.Set("access_token", config.Mapbox.PublicToken)
	}
	url := c.Request.URL
	url.Host = mapboxAPIHost
	url.Scheme = schemeHTTPS
	url.RawQuery = query.Encode()

	resp, err := utils.HTTPRequest(c, http.MethodGet, url.String(), nil, nil)
	if err != nil {
		slog.Error("GETMapboxStyle", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		slog.Error("GETMapboxStyle", "status_code", resp.StatusCode)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Error("GETMapboxStyle", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}

	var jsonyResp map[string]any
	if err := json.Unmarshal(bodyBytes, &jsonyResp); err != nil {
		slog.Error("GETMapboxStyle", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}

	c.JSON(http.StatusOK, jsonyResp)
}

func GETMapboxStyleSpriteAsset(c *gin.Context) {
	owner := c.Param("owner")
	if owner == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "owner is required"})
		return
	}

	styleID := c.Param("styleID")
	if styleID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "styleID is required"})
		return
	}

	version := c.Param("version")
	if version == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "version is required"})
		return
	}

	query := c.Request.URL.Query()
	if owner == "commaai" {
		token, err := base64.RawStdEncoding.DecodeString(commaStyleToken)
		if err != nil {
			slog.Error("GETMapboxStyleSpriteAsset", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
			return
		}
		query.Set("access_token", string(token))
	} else {
		config, ok := c.MustGet("config").(*config.Config)
		if !ok {
			slog.Error("Failed to get config from context")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
			return
		}

		query.Set("access_token", config.Mapbox.PublicToken)
	}
	url := c.Request.URL
	url.Host = mapboxAPIHost
	url.Scheme = schemeHTTPS
	url.RawQuery = query.Encode()

	resp, err := utils.HTTPRequest(c, http.MethodGet, url.String(), nil, nil)
	if err != nil {
		slog.Error("GETMapboxStyleSpriteAsset", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		slog.Error("GETMapboxStyleSpriteAsset", "status_code", resp.StatusCode)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Error("GETMapboxStyleSpriteAsset", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	c.Data(http.StatusOK, contentType, bodyBytes)
}

func GETMapboxTileset(c *gin.Context) {
	tileset := c.Param("tileset")
	if tileset == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tileset is required"})
		return
	}

	if !strings.HasSuffix(tileset, ".json") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tileset must be a .json file"})
		return
	}

	config, ok := c.MustGet("config").(*config.Config)
	if !ok {
		slog.Error("Failed to get config from context")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}

	query := c.Request.URL.Query()
	query.Set("access_token", config.Mapbox.PublicToken)
	url := c.Request.URL
	url.Host = mapboxAPIHost
	url.Scheme = schemeHTTPS
	url.RawQuery = query.Encode()

	resp, err := utils.HTTPRequest(c, http.MethodGet, url.String(), nil, nil)
	if err != nil {
		slog.Error("GETMapboxTileset", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		slog.Error("GETMapboxTileset", "status_code", resp.StatusCode)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Error("GETMapboxTileset", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}

	var jsonyResp map[string]any
	if err := json.Unmarshal(bodyBytes, &jsonyResp); err != nil {
		slog.Error("GETMapboxTileset", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}

	c.JSON(http.StatusOK, jsonyResp)
}

func GETMapboxTile(c *gin.Context) {
	tileset := c.Param("tileset")
	if tileset == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tileset is required"})
		return
	}

	zoom := c.Param("zoom")
	if zoom == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "zoom is required"})
		return
	}

	x := c.Param("x")
	if x == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "x is required"})
		return
	}

	y := c.Param("y")
	if y == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "y is required"})
		return
	}

	if !strings.HasSuffix(y, ".mvt") || !strings.HasSuffix(y, ".vector.pbf") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "format must be a .vector.pbf or .mvt file"})
		return
	}

	config, ok := c.MustGet("config").(*config.Config)
	if !ok {
		slog.Error("Failed to get config from context")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}

	query := c.Request.URL.Query()
	query.Set("access_token", config.Mapbox.PublicToken)
	url := c.Request.URL
	url.Host = mapboxAPIHost
	url.Scheme = schemeHTTPS
	url.RawQuery = query.Encode()

	resp, err := utils.HTTPRequest(c, http.MethodGet, url.String(), nil, nil)
	if err != nil {
		slog.Error("GETMapboxTile", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		slog.Error("GETMapboxTile", "status_code", resp.StatusCode)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Error("GETMapboxTile", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	c.Data(http.StatusOK, contentType, bodyBytes)
}
