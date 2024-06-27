package models

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/USA-RedDragon/connect-server/internal/config"
	"github.com/USA-RedDragon/connect-server/internal/utils"
	gorm_seeder "github.com/kachit/gorm-seeder"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type User struct {
	ID        uint           `json:"id" gorm:"primaryKey" binding:"required"`
	Username  string         `json:"username" gorm:"uniqueIndex" binding:"required"`
	Password  string         `json:"-"`
	Points    uint           `json:"points"`
	Superuser bool           `json:"superuser" gorm:"default:false"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"-"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

func (u User) TableName() string {
	return "users"
}

func UserIDExists(db *gorm.DB, id uint) (bool, error) {
	var count int64
	err := db.Model(&User{}).Where("ID = ?", id).Limit(1).Count(&count).Error
	return count > 0, err
}

func FindUserByID(db *gorm.DB, id uint) (User, error) {
	var user User
	err := db.Preload("Devices").First(&user, id).Error
	return user, err
}

func ListUsers(db *gorm.DB) ([]User, error) {
	var users []User
	err := db.Preload("Devices").Order("id asc").Find(&users).Error
	return users, err
}

func CountUsers(db *gorm.DB) (int, error) {
	var count int64
	err := db.Model(&User{}).Count(&count).Error
	return int(count), err
}

type UsersSeeder struct {
	gorm_seeder.SeederAbstract
	config *config.Config
}

const UserSeederRows = 1

func NewUsersSeeder(cfg gorm_seeder.SeederConfiguration, config *config.Config) UsersSeeder {
	return UsersSeeder{gorm_seeder.NewSeederAbstract(cfg), config}
}

func (s *UsersSeeder) Seed(db *gorm.DB) error {
	if s.config.Registration.InitialAdmin.Username == "" {
		return fmt.Errorf("initial admin username is not set")
	}
	if s.config.Registration.InitialAdmin.Password == "" {
		return fmt.Errorf("initial admin password is not set")
	}
	hashedPassword, err := utils.HashPassword(s.config.Registration.InitialAdmin.Password, s.config.Registration.PasswordSalt)
	if err != nil {
		return err
	}
	var users = []User{
		{
			ID:        0,
			Username:  s.config.Registration.InitialAdmin.Username,
			Password:  hashedPassword,
			Superuser: true,
		},
	}
	slog.Info("Initial admin user created.")
	return db.CreateInBatches(users, s.Configuration.Rows).Error
}

func (s *UsersSeeder) Clear(db *gorm.DB) error {
	return db.Delete(&User{ID: 0}).Error
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
