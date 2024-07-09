package v1

import "github.com/USA-RedDragon/connect-server/internal/db/models"

type Destination struct {
	Set          bool    `json:"-"`
	Latitude     float64 `json:"latitude"`
	Longitude    float64 `json:"longitude"`
	PlaceDetails string  `json:"place_details"`
	PlaceName    string  `json:"place_name"`
}

type SaveLocation struct {
	Label        string          `json:"label,omitempty"`
	Latitude     float64         `json:"latitude" binding:"required"`
	Longitude    float64         `json:"longitude" binding:"required"`
	PlaceDetails string          `json:"place_details,omitempty"`
	PlaceName    string          `json:"place_name,omitempty"`
	SaveType     models.SaveType `json:"save_type" binding:"required"`
}
