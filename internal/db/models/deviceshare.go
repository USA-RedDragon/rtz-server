package models

import (
	"gorm.io/gorm"
)

type DeviceShare struct {
	ID             uint `json:"-" gorm:"primaryKey" binding:"required"`
	DeviceID       uint `json:"-" binding:"required" gorm:"uniqueIndex,OnUpdate:CASCADE,OnDelete:SET NULL"`
	OwnerID        uint `json:"-" binding:"required" gorm:"uniqueIndex,OnUpdate:CASCADE,OnDelete:SET NULL"`
	SharedToUserID uint `json:"-" binding:"required" gorm:"uniqueIndex,OnUpdate:CASCADE,OnDelete:SET NULL"`
}

func (u DeviceShare) TableName() string {
	return "device_shares"
}

func ListSharedByOwnerID(db *gorm.DB, ownerID uint) ([]DeviceShare, error) {
	var shares []DeviceShare
	err := db.Where(&DeviceShare{OwnerID: ownerID}).Find(&shares).Error
	return shares, err
}

func ListSharedToByUserID(db *gorm.DB, sharedToID uint) ([]DeviceShare, error) {
	var shares []DeviceShare
	err := db.Where(&DeviceShare{SharedToUserID: sharedToID}).Find(&shares).Error
	return shares, err
}
