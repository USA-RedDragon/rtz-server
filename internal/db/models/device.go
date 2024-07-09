package models

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/mattn/go-nulltype"
	"gorm.io/gorm"
)

type DeviceType string

const (
	DeviceTypeNeo    DeviceType = "neo"
	DeviceTypePanda  DeviceType = "panda"
	DeviceTypeApp    DeviceType = "app"
	DeviceTypeThreeX DeviceType = "threex"
)

type Device struct {
	ID        uint                `json:"-" gorm:"primaryKey" binding:"required"`
	Alias     nulltype.NullString `json:"alias,omitempty"`
	DongleID  string              `json:"dongle_id" gorm:"uniqueIndex" binding:"required"`
	Serial    string              `json:"serial" gorm:"uniqueIndex" binding:"required"`
	PublicKey string              `json:"public_key" gorm:"uniqueIndex" binding:"required"`
	IsPaired  bool                `json:"is_paired" gorm:"default:false"`
	// Prime defaults to true
	Prime bool `json:"prime" gorm:"default:true"`
	// PrimeType defaults to 1 for "standard prime"
	PrimeType uint `json:"prime_type" gorm:"default:1"`
	// TrialClaimed defaults to true
	TrialClaimed            bool                 `json:"trial_claimed" gorm:"default:true"`
	DeviceType              DeviceType           `json:"device_type"`
	LastGPSTime             nulltype.NullTime    `json:"last_gps_time,omitempty"`
	LastGPSLat              nulltype.NullFloat64 `json:"last_gps_lat,omitempty"`
	LastGPSLng              nulltype.NullFloat64 `json:"last_gps_lng,omitempty"`
	LastGPSAccuracy         nulltype.NullFloat64 `json:"last_gps_accuracy,omitempty"`
	LastGPSSpeed            nulltype.NullFloat64 `json:"last_gps_speed,omitempty"`
	LastGPSBearing          nulltype.NullFloat64 `json:"last_gps_bearing,omitempty"`
	OpenPilotVersion        string               `json:"openpilot_version"`
	LastAthenaPing          int64                `json:"last_athena_ping"`
	DestinationSet          bool                 `json:"-"`
	DestinationLatitude     float64              `json:"-"`
	DestinationLongitude    float64              `json:"-"`
	DestinationPlaceDetails string               `json:"-"`
	DestinationPlaceName    string               `json:"-"`

	OwnerID uint `json:"-"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"-"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

func (u Device) TableName() string {
	return "devices"
}

func DeviceIDExists(db *gorm.DB, id uint) (bool, error) {
	var count int64
	err := db.Model(&Device{}).Where(&Device{ID: id}).Limit(1).Count(&count).Error
	return count > 0, err
}

func FindDeviceByID(db *gorm.DB, id uint) (Device, error) {
	var device Device
	err := db.First(&device, id).Error
	return device, err
}

func FindDeviceByDongleID(db *gorm.DB, id string) (Device, error) {
	var device Device
	err := db.Where(&Device{DongleID: id}).First(&device).Error
	return device, err
}

func FindDeviceBySerial(db *gorm.DB, serial string) (Device, error) {
	var device Device
	err := db.Where(&Device{Serial: serial}).First(&device).Error
	return device, err
}

func ListDevices(db *gorm.DB) ([]Device, error) {
	var devices []Device
	err := db.Order("id asc").Find(&devices).Error
	return devices, err
}

func CountDevices(db *gorm.DB) (int, error) {
	var count int64
	err := db.Model(&Device{}).Count(&count).Error
	return int(count), err
}

func DeleteDevice(db *gorm.DB, id uint) error {
	err := db.Transaction(func(tx *gorm.DB) error {
		tx.Unscoped().Delete(&Device{ID: id})
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

// GenerateDognleID generates a unique random dongle ID
func GenerateDongleID(db *gorm.DB) (string, error) {
	buff := make([]byte, 32)
	len, err := rand.Read(buff)
	if err != nil {
		return "", err
	}
	if len != 32 {
		return "", fmt.Errorf("not enough random bytes")
	}
	candidate := hex.EncodeToString(buff)[:16]

	_, err = FindDeviceByDongleID(db, candidate)
	if err == nil {
		// The device already exists, so try again
		return GenerateDongleID(db)
	}
	return candidate, nil
}

func UpdateAthenaPingTimestamp(db *gorm.DB, id uint) error {
	return db.Model(&Device{}).Where(&Device{ID: id}).Updates(Device{
		LastAthenaPing: time.Now().Unix(),
	}).Error
}

func GetDevicesOwnedByUser(db *gorm.DB, userID uint) ([]Device, error) {
	var devices []Device
	err := db.Where(&Device{OwnerID: userID}).Find(&devices).Error
	return devices, err
}
