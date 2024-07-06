package v1

type LocationResponse struct {
	DongleID string  `json:"dongle_id"`
	Lat      float64 `json:"lat,omitempty"`
	Lon      float64 `json:"lng,omitempty"`
	Time     int64   `json:"time,omitempty"`
}
