package v1

type LocationResponse struct {
	DongleID string  `json:"dongle_id"`
	Lat      float64 `json:"lat"`
	Lon      float64 `json:"lng"`
	Time     int64   `json:"time"`
}
