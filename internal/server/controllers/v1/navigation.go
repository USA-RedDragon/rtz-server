package v1

import (
	"log/slog"
	"net/http"
	"reflect"
	"strconv"
	"time"

	"github.com/USA-RedDragon/rtz-server/internal/config"
	"github.com/USA-RedDragon/rtz-server/internal/db/models"
	"github.com/USA-RedDragon/rtz-server/internal/metrics"
	"github.com/USA-RedDragon/rtz-server/internal/server/apimodels"
	v1 "github.com/USA-RedDragon/rtz-server/internal/server/apimodels/v1"
	"github.com/USA-RedDragon/rtz-server/internal/server/websocket"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/mattn/go-nulltype"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

func POSTSetDestination(c *gin.Context) {
	var destination v1.Destination
	if err := c.BindJSON(&destination); err != nil {
		slog.Error("Failed to bind request", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

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

	maybeRedis, ok := c.Get("redis")
	if !ok && config.Redis.Enabled {
		slog.Error("Failed to get redis from context")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}
	redis, ok := maybeRedis.(*redis.Client)
	if !ok {
		redis = nil
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

	if time.Unix(device.LastAthenaPing, 0).Add(60 * time.Second).After(time.Now()) {
		// Last ping + 60 secs was after now, so the device is online
		rpcCaller, ok := c.MustGet("rpcWebsocket").(*websocket.RPCWebsocket)
		if !ok {
			slog.Error("Failed to get rpc from context")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
			return
		}
		metrics, ok := c.MustGet("metrics").(*metrics.Metrics)
		if !ok {
			slog.Error("Failed to get metrics from context")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
			return
		}
		uuid, err := uuid.NewRandom()
		if err != nil {
			slog.Error("Failed to generate UUID", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
			return
		}
		resp, err := rpcCaller.Call(c, redis, metrics, device.DongleID, apimodels.RPCCall{
			ID:     uuid.String(),
			Method: "setNavDestination",
			Params: map[string]any{
				"latitude":      destination.Latitude,
				"longitude":     destination.Longitude,
				"place_name":    destination.PlaceName,
				"place_details": destination.PlaceDetails,
			},
		})
		if err != nil {
			slog.Error("Failed to call RPC", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
			return
		}
		if resp.Error != "" {
			slog.Error("RPC error", "error", resp.Error)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
			return
		}
		result, ok := resp.Result.(map[string]any)
		if !ok {
			slog.Error("Failed to convert result to string", "result", resp.Result)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
			return
		}
		success, ok := result["success"]
		if !ok {
			slog.Error("Failed to find success", "success", result["success"])
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
			return
		}
		successFloat, ok := success.(float64)
		if !ok {
			slog.Error("Failed to convert success to int", "success", success, "type", reflect.TypeOf(result["success"]))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
			return
		}
		if successFloat == 1 {
			err = db.Model(&device).Updates(models.Device{
				DestinationSet: false,
			}).Error
			if err != nil {
				slog.Error("Failed to update device", "error", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
				return
			}
			c.JSON(http.StatusOK, gin.H{
				"success":    true,
				"saved_next": false,
			})
			return
		}
		// On failure, fall through to save the destination in the db
	}
	err = db.Model(&device).Updates(models.Device{
		DestinationSet:          true,
		DestinationLatitude:     destination.Latitude,
		DestinationLongitude:    destination.Longitude,
		DestinationPlaceName:    destination.PlaceName,
		DestinationPlaceDetails: destination.PlaceDetails,
	}).Error
	if err != nil {
		slog.Error("Failed to update device", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"saved_next": true,
	})
}

func GETNavigationNext(c *gin.Context) {
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
	if device.DestinationSet {
		err = db.Model(&device).Updates(models.Device{
			DestinationSet: false,
		}).Error
		if err != nil {
			slog.Error("Failed to update device", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
			return
		}
		dest := v1.Destination{
			Latitude:     device.DestinationLatitude,
			Longitude:    device.DestinationLongitude,
			PlaceName:    device.DestinationPlaceName,
			PlaceDetails: device.DestinationPlaceDetails,
		}
		c.JSON(http.StatusOK, dest)
		return
	}
	c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
}

func DELETENavigationNext(c *gin.Context) {
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
	err = db.Model(&device).Updates(models.Device{
		DestinationSet: false,
	}).Error
	if err != nil {
		slog.Error("Failed to update device", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}
	c.JSON(http.StatusOK, gin.H{})
}

func GETNavigationLocations(c *gin.Context) {
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
	locations, err := models.FindLocationsByDeviceID(db, device.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}
	c.JSON(http.StatusOK, locations)
}

func PUTNavigationLocations(c *gin.Context) {
	var location v1.SaveLocation
	if err := c.BindJSON(&location); err != nil {
		slog.Error("Failed to bind request", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	if location.SaveType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "save_type is required"})
		return
	}

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

	if location.Label == "home" {
		// Delete existing home location
		err = db.Where(&models.Location{DeviceID: device.ID, Label: nulltype.NullStringOf("home")}).Delete(&models.Location{}).Error
		if err != nil {
			slog.Error("Failed to delete home location", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
			return
		}
	} else if location.Label == "work" {
		// Delete existing home location
		err = db.Where(&models.Location{DeviceID: device.ID, Label: nulltype.NullStringOf("work")}).Delete(&models.Location{}).Error
		if err != nil {
			slog.Error("Failed to delete work location", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
			return
		}
	}

	dbLocation := models.Location{
		DeviceID:     device.ID,
		Latitude:     location.Latitude,
		Longitude:    location.Longitude,
		PlaceDetails: location.PlaceDetails,
		PlaceName:    location.PlaceName,
		SaveType:     location.SaveType,
	}
	if location.Label != "" {
		dbLocation.Label = nulltype.NullStringOf(location.Label)
	}

	err = db.Create(&dbLocation).Error
	if err != nil {
		slog.Error("Failed to save location", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}

	c.JSON(http.StatusOK, gin.H{})
}

func DELETENavigationLocation(c *gin.Context) {
	type req struct {
		ID string `json:"id" binding:"required"`
	}
	var location req
	if err := c.BindJSON(&location); err != nil {
		slog.Error("Failed to bind request", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	uintID, err := strconv.ParseUint(location.ID, 10, 32)
	if err != nil {
		slog.Error("Failed to parse id", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	db, ok := c.MustGet("db").(*gorm.DB)
	if !ok {
		slog.Error("Failed to get db from context")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}

	err = db.Where(&models.Location{ID: uint(uintID)}).Delete(&models.Location{}).Error
	if err != nil {
		slog.Error("Failed to delete location", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}

	c.JSON(http.StatusOK, gin.H{})
}
