package controllers

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func GETMapboxDirections(c *gin.Context) {
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
	annotations := c.Query("annotations")
	geometries := c.Query("geometries")
	overview := c.Query("overview")
	steps := c.Query("steps")
	bannerInstructions := c.Query("banner_instructions")
	alternatives := c.Query("alternatives")
	language := c.Query("language")
	waypoints := c.Query("waypoints")
	bearings := c.Query("bearings")

	slog.Info("GETMapboxDirections", "lat1", sourceCoords[0], "lng1", sourceCoords[1], "lat2", destCoords[2], "lng2", destCoords[3], "annotations", annotations, "geometries", geometries, "overview", overview, "steps", steps, "banner_instructions", bannerInstructions, "alternatives", alternatives, "language", language, "waypoints", waypoints, "bearings", bearings)
	c.JSON(http.StatusNotFound, gin.H{})
}
