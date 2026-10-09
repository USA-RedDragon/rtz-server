package db

import (
	"testing"

	configPkg "github.com/USA-RedDragon/rtz-server/internal/config"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestPostgresDSNQuotesValues(t *testing.T) {
	t.Parallel()
	var cfg configPkg.Config
	cfg.Persistence.Database = configPkg.Database{ //nolint:gosec
		Driver:          configPkg.DatabaseDriverPostgres,
		Host:            "db.example.com",
		Port:            5433,
		Database:        "rtz db",
		Username:        "rtz user",
		Password:        `pa ss'wo\rd`,
		ExtraParameters: "sslmode=disable",
	}
	pgCfg, err := pgconn.ParseConfig(postgresDSN(&cfg))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pgCfg.Host != "db.example.com" || pgCfg.Port != 5433 {
		t.Errorf("unexpected host and port: %s:%d", pgCfg.Host, pgCfg.Port)
	}
	if pgCfg.Database != "rtz db" {
		t.Errorf("unexpected database: %q", pgCfg.Database)
	}
	if pgCfg.User != "rtz user" {
		t.Errorf("unexpected user: %q", pgCfg.User)
	}
	if pgCfg.Password != `pa ss'wo\rd` {
		t.Errorf("unexpected password: %q", pgCfg.Password)
	}
	if pgCfg.TLSConfig != nil {
		t.Error("extra parameters were not applied")
	}
}
