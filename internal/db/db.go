package db

import (
	"fmt"
	"runtime"
	"strings"
	"time"

	configPkg "github.com/USA-RedDragon/rtz-server/internal/config"
	"github.com/USA-RedDragon/rtz-server/internal/db/models"
	"github.com/glebarez/sqlite"
	"github.com/uptrace/opentelemetry-go-extra/otelgorm"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
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
		switch {
		case hasUserAndPassword:
			prefix = fmt.Sprintf("%s:%s@", config.Persistence.Database.Username, config.Persistence.Database.Password)
		case hasUser:
			prefix = fmt.Sprintf("%s@", config.Persistence.Database.Username)
		case hasPassword:
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
		dialector = postgres.New(postgres.Config{
			DSN:                  postgresDSN(config),
			PreferSimpleProtocol: true,
		})
	}
	return dialector
}

// postgresDSN builds a key/value connection string, quoting each value so
// spaces and quotes in credentials survive.
func postgresDSN(config *configPkg.Config) string {
	quote := func(v string) string {
		return "'" + strings.ReplaceAll(strings.ReplaceAll(v, `\`, `\\`), "'", `\'`) + "'"
	}
	dsn := "host=" + quote(config.Persistence.Database.Host) + " dbname=" + quote(config.Persistence.Database.Database)
	if config.Persistence.Database.Port != 0 {
		dsn += fmt.Sprintf(" port=%d", config.Persistence.Database.Port)
	}
	if config.Persistence.Database.Username != "" {
		dsn += " user=" + quote(config.Persistence.Database.Username)
	}
	if config.Persistence.Database.Password != "" {
		dsn += " password=" + quote(config.Persistence.Database.Password)
	}
	if config.Persistence.Database.ExtraParameters != "" {
		dsn += " " + config.Persistence.Database.ExtraParameters
	}
	return dsn
}

func MakeDB(config *configPkg.Config) (db *gorm.DB, err error) {
	db, err = gorm.Open(getDialect(config))
	if err != nil {
		return db, fmt.Errorf("failed to open database: %w", err)
	}
	if config.HTTP.Tracing.OTLPEndpoint != "" {
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
		&models.CrashLog{},
		&models.Route{})
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
