package models

import (
	"time"

	v1dot4 "github.com/USA-RedDragon/rtz-server/internal/server/apimodels/v1.4"
	"gorm.io/gorm"
)

type Route struct {
	ID                      uint      `json:"-" gorm:"primaryKey" binding:"required"`
	DeviceID                uint      `json:"device_id" binding:"required" gorm:"uniqueIndex,OnUpdate:CASCADE,OnDelete:SET NULL"`
	FirstClockWallTimeNanos uint64    `json:"-" binding:"required" gorm:"type:numeric"`
	FirstClockLogMonoTime   uint64    `json:"-" binding:"required" gorm:"type:numeric"`
	AllSegmentsProcessed    bool      `json:"-"`
	RouteID                 string    `json:"id" binding:"required" gorm:"uniqueIndex"`
	EndLat                  float64   `json:"end_lat"`
	EndLng                  float64   `json:"end_lng"`
	EndTime                 time.Time `json:"end_time"`
	GitBranch               string    `json:"git_branch" binding:"required"`
	GitCommit               string    `json:"git_commit" binding:"required"`
	GitDirty                bool      `json:"git_dirty" binding:"required"`
	GitRemote               string    `json:"git_remote" binding:"required"`
	InitLogMonoTime         uint64    `json:"init_log_mono_time" binding:"required" gorm:"type:numeric"`
	IsPreserved             bool      `json:"is_preserved"`
	IsPublic                bool      `json:"is_public"`
	Length                  float64   `json:"length"`
	Platform                string    `json:"platform" binding:"required"`
	Radar                   bool      `json:"radar"`
	StartLat                float64   `json:"start_lat"`
	StartLng                float64   `json:"start_lng"`
	StartTime               time.Time `json:"start_time"`
	URL                     string    `json:"url"`
	Version                 string    `json:"version" binding:"required"`
	// SegmentStartTimes       []uint64  `json:"segment_start_times" gorm:"type:integer[]"`
	// SegmentEndTimes         []uint64  `json:"segment_end_times" gorm:"type:integer[]"`

	CreatedAt time.Time      `json:"create_time"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

func (u Route) TableName() string {
	return "routes"
}

func (u Route) GetWallTimeFromBootTime(bootTime uint64) uint64 {
	return u.FirstClockWallTimeNanos + bootTime - u.FirstClockLogMonoTime
}

func FindRoutesByDeviceID(db *gorm.DB, deviceID uint) ([]Route, error) {
	var routes []Route
	err := db.Where(&Route{DeviceID: deviceID}).Find(&routes).Error
	return routes, err
}

func FindRouteForSegment(db *gorm.DB, deviceID uint, routeInfo v1dot4.RouteInfo) (Route, error) {
	var route Route
	err := db.Order("init_log_mono_time desc").Where("device_id = ? AND route_id = ?", deviceID, routeInfo.Route).First(&route).Error
	return route, err
}

func CountRoutesSince(db *gorm.DB, deviceID uint, since time.Time) (int64, error) {
	var count int64
	err := db.Model(&Route{}).Where("device_id = ? AND start_time > ?", deviceID, since).Count(&count).Error
	return count, err
}
