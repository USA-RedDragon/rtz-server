package models

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"gorm.io/gorm"
)

type DeviceType string

const (
	DeviceTypeNeo   DeviceType = "neo"
	DeviceTypePanda DeviceType = "panda"
	DeviceTypeApp   DeviceType = "app"
)

type Device struct {
	ID        uint   `json:"-" gorm:"primaryKey" binding:"required"`
	DongleID  string `json:"dongle_id" gorm:"uniqueIndex" binding:"required"`
	Serial    string `json:"serial" gorm:"uniqueIndex" binding:"required"`
	PublicKey string `json:"public_key" gorm:"uniqueIndex" binding:"required"`
	// Prime defaults to true
	Prime bool `json:"prime" gorm:"default:true"`
	// PrimeType defaults to 1 for "standard prime"
	PrimeType uint `json:"prime_type" gorm:"default:1"`
	// TrialClaimed defaults to true
	TrialClaimed     bool       `json:"trial_claimed" gorm:"default:true"`
	DeviceType       DeviceType `json:"device_type"`
	LastGPSTime      time.Time  `json:"last_gps_time"`
	LastGPSLat       float64    `json:"last_gps_lat"`
	LastGPSLng       float64    `json:"last_gps_lng"`
	LastGPSAccuracy  float64    `json:"last_gps_accuracy"`
	LastGPSSpeed     float64    `json:"last_gps_speed"`
	LastGPSBearing   float64    `json:"last_gps_bearing"`
	OpenPilotVersion string     `json:"openpilot_version"`
	LastAthenaPing   int64      `json:"last_athena_ping"`

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
	err := db.Model(&Device{}).Where("ID = ?", id).Limit(1).Count(&count).Error
	return count > 0, err
}

func FindDeviceByID(db *gorm.DB, id uint) (Device, error) {
	var device Device
	err := db.First(&device, id).Error
	return device, err
}

func FindDeviceByDongleID(db *gorm.DB, id string) (Device, error) {
	var device Device
	err := db.Where("dongle_id = ?", id).First(&device).Error
	return device, err
}

func FindDeviceBySerial(db *gorm.DB, serial string) (Device, error) {
	var device Device
	err := db.Where("serial = ?", serial).First(&device).Error
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
	candidate := base64.RawURLEncoding.EncodeToString(buff)[:16]

	_, err = FindDeviceByDongleID(db, candidate)
	if err == nil {
		// The device already exists, so try again
		return GenerateDongleID(db)
	}
	return candidate, nil
}

func UpdateAthenaPingTimestamp(db *gorm.DB, id uint) error {
	return db.Model(&Device{}).Where("id = ?", id).Update("last_athena_ping", time.Now().Unix()).Error
}
