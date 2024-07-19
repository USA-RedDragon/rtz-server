package models

import (
	"time"

	"gorm.io/gorm"
)

type Route struct {
	ID                      uint      `json:"id" gorm:"primaryKey" binding:"required"`
	DeviceID                uint      `json:"device_id" binding:"required" gorm:"uniqueIndex,OnUpdate:CASCADE,OnDelete:SET NULL"`
	FirstClockWallTimeNanos uint64    `json:"-" binding:"required" gorm:"type:numeric"`
	FirstClockLogMonoTime   uint64    `json:"-" binding:"required" gorm:"type:numeric"`
	AllSegmentsProcessed    bool      `json:"-"`
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

func FindRouteForSegment(db *gorm.DB, deviceID uint, initLogMonoTime uint64) (Route, error) {
	var route Route
	twoMinsNanoseconds := uint64(1000) * 1000 * 1000 * 60 * 2
	err := db.Order("init_log_mono_time desc").Where("device_id = ? AND all_segments_processed = ? AND init_log_mono_time > ?", deviceID, false, initLogMonoTime-twoMinsNanoseconds).First(&route).Error
	return route, err
}
