package models

import (
	"time"

	"github.com/mattn/go-nulltype"
	"gorm.io/gorm"
)

type SaveType string

const (
	Favorite SaveType = "favorite"
	Recent   SaveType = "recent"
	Home     SaveType = "home"
	Work     SaveType = "work"
)

type Location struct {
	ID           uint                `json:"id" gorm:"primaryKey" binding:"required"`
	DeviceID     uint                `json:"device_id" binding:"required" gorm:"uniqueIndex,OnUpdate:CASCADE,OnDelete:SET NULL"`
	Latitude     float64             `json:"latitude" binding:"required"`
	Longitude    float64             `json:"longitude" binding:"required"`
	Label        nulltype.NullString `json:"label,omitempty"`
	Modified     time.Time           `json:"modified" gorm:"autoUpdateTime:milli,default:current_timestamp"`
	PlaceDetails string              `json:"place_details,omitempty"`
	PlaceName    string              `json:"place_name,omitempty"`
	SaveType     SaveType            `json:"save_type" binding:"required"`

	CreatedAt time.Time      `json:"created_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

func (u Location) TableName() string {
	return "locations"
}

func FindLocationsByDeviceID(db *gorm.DB, deviceID uint) ([]Location, error) {
	var locations []Location
	err := db.Where(&Location{DeviceID: deviceID}).Find(&locations).Error
	return locations, err
}
