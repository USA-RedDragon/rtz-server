package v1

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/USA-RedDragon/rtz-server/internal/db/models"
	v1 "github.com/USA-RedDragon/rtz-server/internal/server/apimodels/v1"
	"github.com/USA-RedDragon/rtz-server/internal/utils"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const commaAPIHost = "api.comma.ai"
const schemeHTTPS = "https"

func PATCHDevice(c *gin.Context) {
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

	var req v1.DevicePatchable
	if err := c.BindJSON(&req); err != nil {
		slog.Error("Failed to bind request", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	err = db.Model(&device).Updates(models.Device{
		Alias: req.Alias,
	}).Error
	if err != nil {
		slog.Error("Failed to update device", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}

	device.Alias = req.Alias

	c.JSON(http.StatusOK, device)
}

func POSTDeviceAddUser(c *gin.Context) {
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

	var req v1.AddUserRequest
	if err := c.BindJSON(&req); err != nil {
		slog.Error("Failed to bind request", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	owner, ok := c.MustGet("user").(*models.User)
	if !ok {
		slog.Error("Failed to get user from context")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}

	var user models.User
	switch {
	case strings.HasPrefix(req.Email, "google_"):
		req.Email = strings.TrimPrefix(req.Email, "google_")
		user, err = models.FindUserByGoogleID(db, req.Email)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
				return
			}
			slog.Error("Failed to find user by Google ID", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
			return
		}
	case strings.HasPrefix(req.Email, "github_"):
		req.Email = strings.TrimPrefix(req.Email, "github_")
		id, err := strconv.Atoi(req.Email)
		if err != nil {
			slog.Error("Failed to convert GitHub ID to int", "error", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
			return
		}
		user, err = models.FindUserByGitHubID(db, id)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
				return
			}
			slog.Error("Failed to find user by GitHub ID", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
			return
		}
	case strings.HasPrefix(req.Email, "custom_"):
		req.Email = strings.TrimPrefix(req.Email, "custom_")
		id, err := strconv.Atoi(req.Email)
		if err != nil {
			slog.Error("Failed to convert Custom ID to int", "error", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
			return
		}
		user, err = models.FindUserByCustomID(db, id)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
				return
			}
			slog.Error("Failed to find user by Custom ID", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
			return
		}
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	err = db.Model(&models.DeviceShare{}).Where("device_id = ? AND shared_to_user_id = ?", device.ID, user.ID).FirstOrCreate(&models.DeviceShare{
		DeviceID:       device.ID,
		SharedToUserID: user.ID,
		OwnerID:        owner.ID,
	}).Error
	if err != nil {
		slog.Error("Failed to share device", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": 1})
}

func POSTDeviceUnpair(c *gin.Context) {
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

	err = db.Model(&device).Update("is_paired", false).Error
	if err != nil {
		slog.Error("Failed to unpair device", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}
	err = db.Model(&device).Update("owner_id", nil).Error
	if err != nil {
		slog.Error("Failed to unpair device", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": 1})
}

func GETDeviceLocation(c *gin.Context) {
	_, ok := c.Get("demo")
	if ok {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": 1, "status_code": 403, "description": "You don't have the permission to access the requested resource. It is either read-protected or not readable by the server."})
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

	resp := v1.LocationResponse{
		DongleID: device.DongleID,
		Lat:      device.LastGPSLat.Float64Value(),
		Lon:      device.LastGPSLng.Float64Value(),
	}
	if device.LastGPSTime.Valid() {
		resp.Time = device.LastGPSTime.TimeValue().UnixMilli()
	} else {
		resp.Time = 0
	}

	c.JSON(http.StatusOK, resp)
}

func GETDeviceRoutesSegments(c *gin.Context) {
	_, ok := c.Get("demo")
	if ok {
		url := c.Request.URL
		url.Host = commaAPIHost
		url.Scheme = schemeHTTPS
		resp, err := utils.HTTPRequest(c, http.MethodGet, url.String(), nil, map[string]string{
			"Authorization": c.GetHeader("Authorization"),
		})
		if err != nil {
			slog.Error("GETDeviceRoutesSegments", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			slog.Error("GETDeviceRoutesSegments", "status_code", resp.StatusCode)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
			return
		}

		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			slog.Error("GETDeviceRoutesSegments", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
			return
		}

		c.Data(http.StatusOK, "application/json", bodyBytes)
		return
	}
	end := c.Query("end")
	start := c.Query("start")
	if end == "" || start == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "start and end are required"})
		return
	}
	limit := c.DefaultQuery("limit", "5")

	startInt, err := strconv.Atoi(start)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "start must be an integer"})
		return
	}
	startTime := time.Unix(int64(startInt/1000), 0)
	endInt, err := strconv.Atoi(end)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "end must be an integer"})
		return
	}
	endTime := time.Unix(int64(endInt/1000), 0)
	limitInt, err := strconv.Atoi(limit)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "limit must be an integer"})
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
		slog.Error("Failed to find device by dongle ID", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}

	routes, err := models.FindRoutesByDeviceIDAndTimeRange(db, device.ID, startTime, endTime, limitInt)
	if err != nil {
		slog.Error("Failed to find routes by device ID and time range", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Try again later"})
		return
	}

	routeSegmentsResponse := []v1.RouteSegmentsResponse{}
	for _, route := range routes {
		routeSegmentsResponse = append(routeSegmentsResponse, v1.RouteSegmentsResponse{
			CAN:                true, // TODO: Implement
			CreationTime:       route.CreatedAt.Unix(),
			DeviceType:         device.DeviceType,
			DongleID:           device.DongleID,
			EndLat:             route.EndLat,
			EndLng:             route.EndLng,
			EndTime:            route.EndTime.Format("2006-01-02T15:04:05"),
			EndTimeUTCMillis:   route.EndTime.UnixMilli(),
			GitBranch:          route.GitBranch,
			GitCommit:          route.GitCommit,
			GitDirty:           route.GitDirty,
			GitRemote:          route.GitRemote,
			FullName:           device.DongleID + "|" + route.RouteID + ":" + strconv.Itoa(int(route.ID)),
			HPGPS:              false, // TODO: Implement
			InitLogMonoTime:    route.InitLogMonoTime,
			IsPreserved:        route.IsPreserved,
			IsPublic:           route.IsPublic,
			Length:             route.Length,
			MaxCamera:          -1,    // TODO: Implement
			MaxDCamera:         -1,    // TODO: Implement
			MaxECamera:         -1,    // TODO: Implement
			MaxLog:             -1,    // TODO: Implement
			MaxQCamera:         -1,    // TODO: Implement
			MaxQLog:            -1,    // TODO: Implement
			Passive:            false, // TODO: Implement
			Platform:           route.Platform,
			ProcCamera:         -1, // TODO: Implement
			ProcLog:            -1, // TODO: Implement
			ProcQCamera:        -1, // TODO: Implement
			ProcQLog:           -1, // TODO: Implement
			Radar:              route.Radar,
			SegmentEndTimes:    []int64{}, // TODO: Implement
			SegmentStartTimes:  []int64{}, // TODO: Implement
			SegmentNumbers:     []int{},   // TODO: Implement
			ShareExp:           "",        // TODO: Implement
			ShareSig:           "",        // TODO: Implement
			StartLat:           route.StartLat,
			StartLng:           route.StartLng,
			StartTime:          route.StartTime.Format("2006-01-02T15:04:05"),
			StartTimeUTCMillis: route.StartTime.UnixMilli(),
			URL:                route.URL,
			UserID:             device.OwnerID,
			Version:            route.Version,
			VIN:                "", // TODO: Implement
		})
	}

	c.JSON(http.StatusOK, routeSegmentsResponse)
}
