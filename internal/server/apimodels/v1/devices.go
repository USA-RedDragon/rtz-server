package v1

import "github.com/mattn/go-nulltype"

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
