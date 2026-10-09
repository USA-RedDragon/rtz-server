package v1

import (
	"errors"
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
	"github.com/nats-io/nats.go"
	"gorm.io/gorm"
)

func POSTSetDestination(c *gin.Context) {
	var destination v1.Destination
	if err := c.BindJSON(&destination); err != nil {
		slog.Error("Failed to bind request", errorKey, err)
		c.JSON(http.StatusBadRequest, gin.H{errorKey: msgInvalidRequest})
		return
	}

	dongleID, ok := c.Params.Get("dongle_id")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{errorKey: msgDongleIDRequired})
		return
	}

	config, ok := c.MustGet("config").(*config.Config)
	if !ok {
		slog.Error("Failed to get config from context")
		c.JSON(http.StatusInternalServerError, gin.H{errorKey: msgTryAgainLater})
		return
	}

	maybeNats, ok := c.Get("nats")
	if !ok && config.NATS.Enabled {
		slog.Error("Failed to get NATS from context")
		c.JSON(http.StatusInternalServerError, gin.H{errorKey: msgTryAgainLater})
		return
	}
	nc, ok := maybeNats.(*nats.Conn)
	if !ok {
		nc = nil
	}

	db, ok := c.MustGet("db").(*gorm.DB)
	if !ok {
		slog.Error("Failed to get db from context")
		c.JSON(http.StatusInternalServerError, gin.H{errorKey: msgTryAgainLater})
		return
	}
	device, err := models.FindDeviceByDongleID(db, dongleID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{errorKey: msgTryAgainLater})
		return
	}

	if time.Unix(device.LastAthenaPing, 0).Add(60 * time.Second).After(time.Now()) {
		// Last ping + 60 secs was after now, so the device is online
		rpcCaller, ok := c.MustGet("rpcWebsocket").(*websocket.RPCWebsocket)
		if !ok {
			slog.Error("Failed to get rpc from context")
			c.JSON(http.StatusInternalServerError, gin.H{errorKey: msgTryAgainLater})
			return
		}
		metrics, ok := c.MustGet("metrics").(*metrics.Metrics)
		if !ok {
			slog.Error("Failed to get metrics from context")
			c.JSON(http.StatusInternalServerError, gin.H{errorKey: msgTryAgainLater})
			return
		}
		uuid, err := uuid.NewRandom()
		if err != nil {
			slog.Error("Failed to generate UUID", errorKey, err)
			c.JSON(http.StatusInternalServerError, gin.H{errorKey: msgTryAgainLater})
			return
		}
		resp, err := rpcCaller.Call(c, nc, metrics, device.DongleID, apimodels.RPCCall{
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
			if errors.Is(err, nats.ErrNoResponders) {
				c.JSON(http.StatusNotFound, gin.H{errorKey: "Dongle not connected"})
				return
			}
			if errors.Is(err, websocket.ErrNotConnected) {
				c.JSON(http.StatusNotFound, gin.H{errorKey: "Dongle not connected"})
				return
			}
			slog.Error("Failed to call RPC", errorKey, err)
			c.JSON(http.StatusInternalServerError, gin.H{errorKey: msgTryAgainLater})
			return
		}
		if resp.Error != "" {
			slog.Error("RPC error", errorKey, resp.Error)
			c.JSON(http.StatusInternalServerError, gin.H{errorKey: msgTryAgainLater})
			return
		}
		result, ok := resp.Result.(map[string]any)
		if !ok {
			slog.Error("Failed to convert result to string", "result", resp.Result)
			c.JSON(http.StatusInternalServerError, gin.H{errorKey: msgTryAgainLater})
			return
		}
		success, ok := result[successKey]
		if !ok {
			slog.Error("Failed to find success", successKey, result[successKey])
			c.JSON(http.StatusInternalServerError, gin.H{errorKey: msgTryAgainLater})
			return
		}
		successFloat, ok := success.(float64)
		if !ok {
			slog.Error("Failed to convert success to int", successKey, success, "type", reflect.TypeOf(result[successKey]))
			c.JSON(http.StatusInternalServerError, gin.H{errorKey: msgTryAgainLater})
			return
		}
		if successFloat == 1 {
			err = db.Model(&device).Update("destination_set", false).Error
			if err != nil {
				slog.Error("Failed to update device", errorKey, err)
				c.JSON(http.StatusInternalServerError, gin.H{errorKey: msgTryAgainLater})
				return
			}
			c.JSON(http.StatusOK, gin.H{
				successKey:   true,
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
		slog.Error("Failed to update device", errorKey, err)
		c.JSON(http.StatusInternalServerError, gin.H{errorKey: msgTryAgainLater})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		successKey:   true,
		"saved_next": true,
	})
}

func GETNavigationNext(c *gin.Context) {
	dongleID, ok := c.Params.Get("dongle_id")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{errorKey: msgDongleIDRequired})
		return
	}
	db, ok := c.MustGet("db").(*gorm.DB)
	if !ok {
		slog.Error("Failed to get db from context")
		c.JSON(http.StatusInternalServerError, gin.H{errorKey: msgTryAgainLater})
		return
	}
	device, err := models.FindDeviceByDongleID(db, dongleID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{errorKey: msgTryAgainLater})
		return
	}
	if device.DestinationSet {
		err = db.Model(&device).Update("destination_set", false).Error
		if err != nil {
			slog.Error("Failed to update device", errorKey, err)
			c.JSON(http.StatusInternalServerError, gin.H{errorKey: msgTryAgainLater})
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
	c.JSON(http.StatusNotFound, gin.H{errorKey: "Not found"})
}

func DELETENavigationNext(c *gin.Context) {
	dongleID, ok := c.Params.Get("dongle_id")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{errorKey: msgDongleIDRequired})
		return
	}
	db, ok := c.MustGet("db").(*gorm.DB)
	if !ok {
		slog.Error("Failed to get db from context")
		c.JSON(http.StatusInternalServerError, gin.H{errorKey: msgTryAgainLater})
		return
	}
	device, err := models.FindDeviceByDongleID(db, dongleID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{errorKey: msgTryAgainLater})
		return
	}
	err = db.Model(&device).Update("destination_set", false).Error
	if err != nil {
		slog.Error("Failed to update device", errorKey, err)
		c.JSON(http.StatusInternalServerError, gin.H{errorKey: msgTryAgainLater})
		return
	}
	c.JSON(http.StatusOK, gin.H{})
}

func GETNavigationLocations(c *gin.Context) {
	dongleID, ok := c.Params.Get("dongle_id")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{errorKey: msgDongleIDRequired})
		return
	}
	db, ok := c.MustGet("db").(*gorm.DB)
	if !ok {
		slog.Error("Failed to get db from context")
		c.JSON(http.StatusInternalServerError, gin.H{errorKey: msgTryAgainLater})
		return
	}
	device, err := models.FindDeviceByDongleID(db, dongleID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{errorKey: msgTryAgainLater})
		return
	}
	locations, err := models.FindLocationsByDeviceID(db, device.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{errorKey: msgTryAgainLater})
		return
	}
	c.JSON(http.StatusOK, locations)
}

func PUTNavigationLocations(c *gin.Context) {
	var location v1.SaveLocation
	if err := c.BindJSON(&location); err != nil {
		slog.Error("Failed to bind request", errorKey, err)
		c.JSON(http.StatusBadRequest, gin.H{errorKey: msgInvalidRequest})
		return
	}

	if location.SaveType == "" {
		c.JSON(http.StatusBadRequest, gin.H{errorKey: "save_type is required"})
		return
	}

	dongleID, ok := c.Params.Get("dongle_id")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{errorKey: msgDongleIDRequired})
		return
	}
	db, ok := c.MustGet("db").(*gorm.DB)
	if !ok {
		slog.Error("Failed to get db from context")
		c.JSON(http.StatusInternalServerError, gin.H{errorKey: msgTryAgainLater})
		return
	}
	device, err := models.FindDeviceByDongleID(db, dongleID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{errorKey: msgTryAgainLater})
		return
	}

	switch location.Label {
	case "home":
		// Delete existing home location
		err = db.Where(&models.Location{DeviceID: device.ID, Label: nulltype.NullStringOf("home")}).Delete(&models.Location{}).Error
		if err != nil {
			slog.Error("Failed to delete home location", errorKey, err)
			c.JSON(http.StatusInternalServerError, gin.H{errorKey: msgTryAgainLater})
			return
		}
	case "work":
		// Delete existing work location
		err = db.Where(&models.Location{DeviceID: device.ID, Label: nulltype.NullStringOf("work")}).Delete(&models.Location{}).Error
		if err != nil {
			slog.Error("Failed to delete work location", errorKey, err)
			c.JSON(http.StatusInternalServerError, gin.H{errorKey: msgTryAgainLater})
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
		slog.Error("Failed to save location", errorKey, err)
		c.JSON(http.StatusInternalServerError, gin.H{errorKey: msgTryAgainLater})
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
		slog.Error("Failed to bind request", errorKey, err)
		c.JSON(http.StatusBadRequest, gin.H{errorKey: msgInvalidRequest})
		return
	}
	uintID, err := strconv.ParseUint(location.ID, 10, 32)
	if err != nil {
		slog.Error("Failed to parse id", errorKey, err)
		c.JSON(http.StatusBadRequest, gin.H{errorKey: msgInvalidRequest})
		return
	}

	db, ok := c.MustGet("db").(*gorm.DB)
	if !ok {
		slog.Error("Failed to get db from context")
		c.JSON(http.StatusInternalServerError, gin.H{errorKey: msgTryAgainLater})
		return
	}

	err = db.Where(&models.Location{ID: uint(uintID)}).Delete(&models.Location{}).Error
	if err != nil {
		slog.Error("Failed to delete location", errorKey, err)
		c.JSON(http.StatusInternalServerError, gin.H{errorKey: msgTryAgainLater})
		return
	}

	c.JSON(http.StatusOK, gin.H{})
}
