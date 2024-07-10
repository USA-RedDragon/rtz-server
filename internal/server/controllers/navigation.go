package controllers

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

func GETMapboxDirections(c *gin.Context) {
	lat1 := c.Param("lat1")
	lng1 := c.Param("lng1")
	lat2 := c.Param("lat2")
	lng2 := c.Param("lng2")
	if lat1 == "" || lng1 == "" || lat2 == "" || lng2 == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "lat1, lng1, lat2, lng2 are required"})
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

	slog.Info("GETMapboxDirections", "lat1", lat1, "lng1", lng1, "lat2", lat2, "lng2", lng2, "annotations", annotations, "geometries", geometries, "overview", overview, "steps", steps, "banner_instructions", bannerInstructions, "alternatives", alternatives, "language", language, "waypoints", waypoints, "bearings", bearings)
	c.JSON(http.StatusNotFound, gin.H{})
}
