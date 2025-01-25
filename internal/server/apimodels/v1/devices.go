package v1

import (
	"github.com/USA-RedDragon/rtz-server/internal/db/models"
	"github.com/mattn/go-nulltype"
)

type LocationResponse struct {
	DongleID string  `json:"dongle_id"`
	Lat      float64 `json:"lat"`
	Lon      float64 `json:"lng"`
	Time     int64   `json:"time"`
}

type DevicePatchable struct {
	Alias nulltype.NullString `json:"alias" binding:"required"`
}

type AddUserRequest struct {
	Email string `json:"email" binding:"required"`
}

type RouteSegmentsResponse struct {
	CAN                bool              `json:"can"`
	CreationTime       int64             `json:"create_time"`
	DeviceType         models.DeviceType `json:"device_type"`
	DongleID           string            `json:"dongle_id"`
	EndLat             float64           `json:"end_lat"`
	EndLng             float64           `json:"end_lng"`
	EndTime            string            `json:"end_time"`
	EndTimeUTCMillis   int64             `json:"end_time_utc_millis"`
	FullName           string            `json:"fullname"`
	GitBranch          string            `json:"git_branch"`
	GitCommit          string            `json:"git_commit"`
	GitDirty           bool              `json:"git_dirty"`
	GitRemote          string            `json:"git_remote"`
	HPGPS              bool              `json:"hpgps"`
	InitLogMonoTime    int64             `json:"init_logmonotime"`
	IsPreserved        bool              `json:"is_preserved"`
	IsPublic           bool              `json:"is_public"`
	Length             float64           `json:"length"`
	MaxCamera          int               `json:"maxcamera"`
	MaxDCamera         int               `json:"maxdcamera"`
	MaxECamera         int               `json:"maxecamera"`
	MaxLog             int               `json:"maxlog"`
	MaxQCamera         int               `json:"maxqcamera"`
	MaxQLog            int               `json:"maxqlog"`
	Passive            bool              `json:"passive"`
	Platform           string            `json:"platform"`
	ProcCamera         int               `json:"proccamera"`
	ProcLog            int               `json:"proclog"`
	ProcQCamera        int               `json:"procqcamera"`
	ProcQLog           int               `json:"procqlog"`
	Radar              bool              `json:"radar"`
	SegmentEndTimes    []int64           `json:"segment_end_times"`
	SegmentStartTimes  []int64           `json:"segment_start_times"`
	SegmentNumbers     []int             `json:"segment_numbers"`
	ShareExp           string            `json:"share_exp"`
	ShareSig           string            `json:"share_sig"`
	StartLat           float64           `json:"start_lat"`
	StartLng           float64           `json:"start_lng"`
	StartTime          string            `json:"start_time"`
	StartTimeUTCMillis int64             `json:"start_time_utc_millis"`
	URL                string            `json:"url"`
	UserID             uint              `json:"user_id"`
	Version            string            `json:"version"`
	VIN                string            `json:"vin"`
}
