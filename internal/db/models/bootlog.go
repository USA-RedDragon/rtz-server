package models

type BootLog struct {
	ID       uint   `json:"-" gorm:"primaryKey"`
	DeviceID uint   `json:"-" binding:"required" gorm:"uniqueIndex,OnUpdate:CASCADE,OnDelete:SET NULL"`
	FileName string `json:"file_name" binding:"required"`
}
