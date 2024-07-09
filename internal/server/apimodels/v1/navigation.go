package v1

type Destination struct {
	Set          bool    `json:"-"`
	Latitude     float64 `json:"latitude"`
	Longitude    float64 `json:"longitude"`
	PlaceDetails string  `json:"place_details"`
	PlaceName    string  `json:"place_name"`
}
