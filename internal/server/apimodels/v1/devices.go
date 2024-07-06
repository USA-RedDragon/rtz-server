package v1

import "github.com/datumbrain/nulltypes"

type LocationResponse struct {
	DongleID string                `json:"dongle_id"`
	Lat      nulltypes.NullFloat64 `json:"lat"`
	Lon      nulltypes.NullFloat64 `json:"lng"`
	Time     nulltypes.NullInt64   `json:"time"`
}
