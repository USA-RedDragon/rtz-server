package db

import (
	"fmt"
	"runtime"
	"time"

	"github.com/USA-RedDragon/connect-server/internal/config"
	"github.com/USA-RedDragon/connect-server/internal/db/models"
	"github.com/glebarez/sqlite"
	"github.com/uptrace/opentelemetry-go-extra/otelgorm"
	"gorm.io/gorm"
)

func MakeDB(config *config.Config) (db *gorm.DB, err error) {
	db, err = gorm.Open(sqlite.Open(config.Persistence.Database+"?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)"), &gorm.Config{})
	if err != nil {
		return db, fmt.Errorf("failed to open database: %w", err)
	}
	if config.HTTP.OTLPEndpoint != "" {
		if err = db.Use(otelgorm.NewPlugin()); err != nil {
			return db, fmt.Errorf("failed to trace database: %w", err)
		}
	}

	err = db.AutoMigrate(
		&models.Device{},
		&models.User{},
		&models.Location{},
		&models.DeviceShare{},
		&models.BootLog{},
		&models.CrashLog{})
	if err != nil {
		return db, fmt.Errorf("failed to migrate database: %w", err)
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
