package db

import (
	"fmt"
	"runtime"
	"time"

	configPkg "github.com/USA-RedDragon/rtz-server/internal/config"
	"github.com/USA-RedDragon/rtz-server/internal/db/models"
	"github.com/glebarez/sqlite"
	"github.com/uptrace/opentelemetry-go-extra/otelgorm"
	"gorm.io/gorm"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
)

func getDialect(config *configPkg.Config) gorm.Dialector {
	var dialector gorm.Dialector
	switch config.Persistence.Database.Driver {
	case configPkg.DatabaseDriverSQLite:
		dialector = sqlite.Open(
			config.Persistence.Database.Database + "?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)" + "&" + config.Persistence.Database.ExtraParameters,
		)
	case configPkg.DatabaseDriverMySQL:
		hasUser := config.Persistence.Database.Username != ""
		hasPassword := config.Persistence.Database.Password != ""
		hasUserAndPassword := hasUser && hasPassword
		prefix := ""
		if hasUserAndPassword {
			prefix = fmt.Sprintf("%s:%s@", config.Persistence.Database.Username, config.Persistence.Database.Password)
		} else if hasUser {
			prefix = fmt.Sprintf("%s@", config.Persistence.Database.Username)
		} else if hasPassword {
			prefix = fmt.Sprintf(":%s@", config.Persistence.Database.Password)
		}
		portStr := ""
		if config.Persistence.Database.Port != 0 {
			portStr = fmt.Sprintf(":%d", config.Persistence.Database.Port)
		}
		extraParamsStr := ""
		if config.Persistence.Database.ExtraParameters != "" {
			extraParamsStr = "&" + config.Persistence.Database.ExtraParameters
		}
		dsn := fmt.Sprintf("%stcp(%s%s)/%s?charset=utf8mb4&parseTime=True&loc=Local&%s",
			prefix,
			config.Persistence.Database.Host,
			portStr,
			config.Persistence.Database.Database,
			extraParamsStr)
		dialector = mysql.Open(dsn)
	case configPkg.DatabaseDriverPostgres:
		dsn := "host=" + config.Persistence.Database.Host + " dbname=" + config.Persistence.Database.Database
		if config.Persistence.Database.Port != 0 {
			dsn += fmt.Sprintf(" port=%d", config.Persistence.Database.Port)
		}
		if config.Persistence.Database.Username != "" {
			dsn += " user=" + config.Persistence.Database.Username
		}
		if config.Persistence.Database.Password != "" {
			dsn += " password=" + config.Persistence.Database.Password
		}
		if config.Persistence.Database.ExtraParameters != "" {
			dsn += " " + config.Persistence.Database.ExtraParameters
		}
		dialector = postgres.New(postgres.Config{
			DSN:                  dsn,
			PreferSimpleProtocol: true,
		})
	}
	return dialector
}

func MakeDB(config *configPkg.Config) (db *gorm.DB, err error) {
	db, err = gorm.Open(getDialect(config))
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
