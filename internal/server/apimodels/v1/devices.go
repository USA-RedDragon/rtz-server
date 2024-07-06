package v1

import "github.com/mattn/go-nulltype"

type LocationResponse struct {
	DongleID string               `json:"dongle_id"`
	Lat      nulltype.NullFloat64 `json:"lat,omitempty"`
	Lon      nulltype.NullFloat64 `json:"lng,omitempty"`
	Time     nulltype.NullInt64   `json:"time,omitempty"`
}
