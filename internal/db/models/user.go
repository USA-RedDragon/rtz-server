package models

import (
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type User struct {
	ID           uint           `json:"id" gorm:"primaryKey" binding:"required"`
	GitHubUserID int            `json:"github_user_id" gorm:"uniqueIndex"`
	GoogleUserID string         `json:"google_user_id" gorm:"uniqueIndex"`
	Superuser    bool           `json:"superuser" gorm:"default:false"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"-"`
	DeletedAt    gorm.DeletedAt `json:"-" gorm:"index"`
}

func (u User) TableName() string {
	return "users"
}

func UserIDExists(db *gorm.DB, id uint) (bool, error) {
	var count int64
	err := db.Model(&User{}).Where(&User{ID: id}).Limit(1).Count(&count).Error
	return count > 0, err
}

func FindUserByID(db *gorm.DB, id uint) (User, error) {
	var user User
	err := db.First(&user, id).Error
	return user, err
}

func FindUserByGitHubID(db *gorm.DB, id int) (User, error) {
	var user User
	err := db.Where(&User{GitHubUserID: id}).First(&user).Error
	return user, err
}

func FindUserByGoogleID(db *gorm.DB, id string) (User, error) {
	var user User
	err := db.Where(&User{GoogleUserID: id}).First(&user).Error
	return user, err
}

func ListUsers(db *gorm.DB) ([]User, error) {
	var users []User
	err := db.Order("id asc").Find(&users).Error
	return users, err
}

func CountUsers(db *gorm.DB) (int, error) {
	var count int64
	err := db.Model(&User{}).Count(&count).Error
	return int(count), err
}

func DeleteUser(db *gorm.DB, id uint) error {
	err := db.Transaction(func(tx *gorm.DB) error {
		tx.Unscoped().Select(clause.Associations, "Devices").Delete(&User{ID: id})
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}
