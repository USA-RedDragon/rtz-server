package controllers

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func GETMapboxDirections(c *gin.Context) {
	coordsParam := c.Param("coords")
	coords := strings.Split(coordsParam, ",")
	if len(coords) != 4 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "coords must be 4 comma separated values"})
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

	slog.Info("GETMapboxDirections", "lat1", coords[0], "lng1", coords[1], "lat2", coords[2], "lng2", coords[3], "annotations", annotations, "geometries", geometries, "overview", overview, "steps", steps, "banner_instructions", bannerInstructions, "alternatives", alternatives, "language", language, "waypoints", waypoints, "bearings", bearings)
	c.JSON(http.StatusNotFound, gin.H{})
}
