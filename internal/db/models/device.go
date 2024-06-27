package models

import (
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
	Alias     string `json:"alias"`
	IsPaired  bool   `json:"is_paired" gorm:"default:false"`
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

	Owner   User `json:"owner" gorm:"foreignKey:OwnerID"`
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
	err := db.Preload("Owner").First(&device, id).Error
	return device, err
}

func ListDevices(db *gorm.DB) ([]Device, error) {
	var devices []Device
	err := db.Preload("Owner").Order("id asc").Find(&devices).Error
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

func GetUserDevices(db *gorm.DB, id uint) ([]Device, error) {
	var devices []Device
	err := db.Preload("Owner").Where("owner_id = ?", id).Order("id asc").Find(&devices).Error
	return devices, err
}

func CountUserDevices(db *gorm.DB, id uint) (int, error) {
	var count int64
	err := db.Model(&Device{}).Where("owner_id = ?", id).Count(&count).Error
	return int(count), err
}
