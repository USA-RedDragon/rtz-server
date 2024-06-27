package db

import (
	"fmt"
	"log/slog"
	"runtime"
	"time"

	"github.com/USA-RedDragon/connect-server/internal/config"
	"github.com/USA-RedDragon/connect-server/internal/db/models"
	"github.com/glebarez/sqlite"
	gorm_seeder "github.com/kachit/gorm-seeder"
	"github.com/uptrace/opentelemetry-go-extra/otelgorm"
	"gorm.io/gorm"
)

func MakeDB(config *config.Config) (db *gorm.DB, err error) {
	db, err = gorm.Open(sqlite.Open(config.Persistence.Database), &gorm.Config{})
	if err != nil {
		return db, fmt.Errorf("failed to open database: %w", err)
	}
	if config.HTTP.OTLPEndpoint != "" {
		if err = db.Use(otelgorm.NewPlugin()); err != nil {
			return db, fmt.Errorf("failed to trace database: %w", err)
		}
	}

	err = db.AutoMigrate(&models.AppSettings{}, &models.User{})
	if err != nil {
		return db, fmt.Errorf("failed to migrate database: %w", err)
	}

	// Grab the first (and only) AppSettings record. If that record doesn't exist, create it.
	var appSettings models.AppSettings
	result := db.First(&appSettings)
	if result.RowsAffected == 0 {
		slog.Debug("App settings entry doesn't exist, creating it")
		// The record doesn't exist, so create it
		appSettings = models.AppSettings{
			HasSeeded: false,
		}
		err := db.Create(&appSettings).Error
		if err != nil {
			return db, fmt.Errorf("failed to create app settings: %w", err)
		}
		slog.Debug("App settings saved")
	}

	// If the record exists and HasSeeded is true, then we don't need to seed the database.
	if !appSettings.HasSeeded {
		usersSeeder := models.NewUsersSeeder(gorm_seeder.SeederConfiguration{Rows: models.UserSeederRows}, config)
		seedersStack := gorm_seeder.NewSeedersStack(db)
		seedersStack.AddSeeder(&usersSeeder)

		// Apply seed
		err = seedersStack.Seed()
		if err != nil {
			return db, fmt.Errorf("failed to seed database: %w", err)
		}
		appSettings.HasSeeded = true
		err := db.Save(&appSettings).Error
		if err != nil {
			return db, fmt.Errorf("failed to save app settings: %w", err)
		}
	}

	sqlDB, err := db.DB()
	if err != nil {
		return db, fmt.Errorf("failed to open database: %w", err)
	}
	sqlDB.SetMaxIdleConns(runtime.GOMAXPROCS(0))
	const connsPerCPU = 10
	sqlDB.SetMaxOpenConns(runtime.GOMAXPROCS(0) * connsPerCPU)
	const maxIdleTime = 10 * time.Minute
	sqlDB.SetConnMaxIdleTime(maxIdleTime)

	return
}
