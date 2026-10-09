package config_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	configulator "github.com/USA-RedDragon/configulator/v2"
	"github.com/USA-RedDragon/rtz-server/internal/config"
	"github.com/spf13/pflag"
)

//nolint:gochecknoglobals
var requiredFlags = []string{
	"--jwt.secret", "changeme",
	"--http.backend_url", "http://localhost:8081",
	"--mapbox.secret_token", "dummy",
	"--mapbox.public_token", "dummy",
}

func load(t *testing.T, args ...string) (*config.Config, error) {
	t.Helper()
	fs := pflag.NewFlagSet("testing", pflag.ContinueOnError)
	c := config.NewConfigulator(fs)
	if err := fs.Parse(args); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return c.Load()
}

func writeConfig(t *testing.T, data string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestExampleConfig(t *testing.T) {
	t.Parallel()
	if _, err := load(t, append([]string{"--config", "../../config.example.yaml"}, requiredFlags...)...); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestMissingConfigFile(t *testing.T) {
	t.Parallel()
	var missing *configulator.MissingFileError
	if _, err := load(t, append([]string{"--config", "does-not-exist.yaml"}, requiredFlags...)...); !errors.As(err, &missing) {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestMissingOTLPEndpoint(t *testing.T) {
	t.Parallel()
	_, err := load(t, append([]string{"--http.tracing.enabled", "true"}, requiredFlags...)...)
	if !errors.Is(err, config.ErrOTLPEndpointRequired) {
		t.Errorf("unexpected error: %v", err)
	}

	_, err = load(t, append([]string{"--http.tracing.enabled", "true", "--http.tracing.otlp_endpoint", "dummy"}, requiredFlags...)...)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestMissingRequired(t *testing.T) {
	t.Parallel()
	tests := []struct {
		path  string
		flags []string
	}{
		{"jwt.secret", []string{"--http.backend_url", "x", "--mapbox.secret_token", "x", "--mapbox.public_token", "x"}},
		{"http.backend_url", []string{"--jwt.secret", "x", "--mapbox.secret_token", "x", "--mapbox.public_token", "x"}},
		{"mapbox.public_token", []string{"--jwt.secret", "x", "--http.backend_url", "x", "--mapbox.secret_token", "x"}},
		{"mapbox.secret_token", []string{"--jwt.secret", "x", "--http.backend_url", "x", "--mapbox.public_token", "x"}},
	}
	for _, tt := range tests {
		_, err := load(t, tt.flags...)
		var required *configulator.RequiredError
		if !errors.As(err, &required) || required.Path != tt.path {
			t.Errorf("%s: unexpected error: %v", tt.path, err)
		}
	}
}

func TestEmptyRequired(t *testing.T) {
	t.Parallel()
	tests := []struct {
		flag string
		want error
	}{
		{"--jwt.secret", config.ErrJWTSecretRequired},
		{"--http.backend_url", config.ErrBackendURLRequired},
		{"--mapbox.public_token", config.ErrMapboxPublicTokenRequired},
		{"--mapbox.secret_token", config.ErrMapboxSecretTokenRequired},
	}
	for _, tt := range tests {
		_, err := load(t, append(append([]string{}, requiredFlags...), tt.flag, "")...)
		if !errors.Is(err, tt.want) {
			t.Errorf("%s: got %v, want %v", tt.flag, err, tt.want)
		}
	}
}

func TestDefaults(t *testing.T) {
	t.Parallel()
	cfg, err := load(t, requiredFlags...)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.HTTP.IPV4Host != "0.0.0.0" || cfg.HTTP.IPV6Host != "::" || cfg.HTTP.Port != 8080 {
		t.Errorf("unexpected HTTP listener: %s %s %d", cfg.HTTP.IPV4Host, cfg.HTTP.IPV6Host, cfg.HTTP.Port)
	}
	if cfg.HTTP.Metrics.IPV4Host != "127.0.0.1" || cfg.HTTP.Metrics.IPV6Host != "::1" || cfg.HTTP.Metrics.Port != 8081 {
		t.Errorf("unexpected metrics listener: %s %s %d", cfg.HTTP.Metrics.IPV4Host, cfg.HTTP.Metrics.IPV6Host, cfg.HTTP.Metrics.Port)
	}
	if cfg.Persistence.Database.Driver != config.DatabaseDriverSQLite || cfg.Persistence.Database.Database != "rtz.db" {
		t.Errorf("unexpected database: %s %s", cfg.Persistence.Database.Driver, cfg.Persistence.Database.Database)
	}
	if cfg.Persistence.Uploads.Driver != config.UploadsDriverFilesystem || cfg.Persistence.Uploads.FilesystemOptions.Directory != "uploads/" {
		t.Errorf("unexpected uploads: %s %s", cfg.Persistence.Uploads.Driver, cfg.Persistence.Uploads.FilesystemOptions.Directory)
	}
	if cfg.ParallelLogParsers != 4 || cfg.LogLevel != config.LogLevelInfo {
		t.Errorf("unexpected parsers and log level: %d %s", cfg.ParallelLogParsers, cfg.LogLevel)
	}
}

func TestFileListeners(t *testing.T) {
	t.Parallel()
	path := writeConfig(t, `http:
  ipv4_host: 127.0.0.2
  ipv6_host: ::2
  port: 9000
  metrics:
    ipv4_host: 127.0.0.3
    ipv6_host: ::3
    port: 9001
`)
	cfg, err := load(t, append([]string{"--config", path}, requiredFlags...)...)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.HTTP.IPV4Host != "127.0.0.2" || cfg.HTTP.IPV6Host != "::2" || cfg.HTTP.Port != 9000 {
		t.Errorf("unexpected HTTP listener: %s %s %d", cfg.HTTP.IPV4Host, cfg.HTTP.IPV6Host, cfg.HTTP.Port)
	}
	if cfg.HTTP.Metrics.IPV4Host != "127.0.0.3" || cfg.HTTP.Metrics.IPV6Host != "::3" || cfg.HTTP.Metrics.Port != 9001 {
		t.Errorf("unexpected metrics listener: %s %s %d", cfg.HTTP.Metrics.IPV4Host, cfg.HTTP.Metrics.IPV6Host, cfg.HTTP.Metrics.Port)
	}
}

func TestFileExtraParameters(t *testing.T) {
	t.Parallel()
	path := writeConfig(t, "persistence:\n  database:\n    extra_parameters: sslmode=require\n")
	cfg, err := load(t, append([]string{"--config", path}, requiredFlags...)...)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Persistence.Database.ExtraParameters != "sslmode=require" {
		t.Errorf("unexpected extra parameters: %q", cfg.Persistence.Database.ExtraParameters)
	}
}

func TestFileDriversLowercased(t *testing.T) {
	t.Parallel()
	path := writeConfig(t, "persistence:\n  database:\n    driver: SQLite\n  uploads:\n    driver: FileSystem\n")
	cfg, err := load(t, append([]string{"--config", path}, requiredFlags...)...)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Persistence.Database.Driver != config.DatabaseDriverSQLite {
		t.Errorf("unexpected database driver: %s", cfg.Persistence.Database.Driver)
	}
	if cfg.Persistence.Uploads.Driver != config.UploadsDriverFilesystem {
		t.Errorf("unexpected uploads driver: %s", cfg.Persistence.Uploads.Driver)
	}
}

func TestFileZeroPortSticks(t *testing.T) {
	t.Parallel()
	path := writeConfig(t, "http:\n  port: 0\n")
	cfg, err := load(t, append([]string{"--config", path}, requiredFlags...)...)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.HTTP.Port != 0 {
		t.Errorf("unexpected HTTP port: %d", cfg.HTTP.Port)
	}
}

func TestInvalidDrivers(t *testing.T) {
	t.Parallel()
	tests := []struct {
		flags []string
		want  error
	}{
		{[]string{"--persistence.database.driver", "oracle"}, config.ErrInvalidDatabaseDriver},
		{[]string{"--persistence.uploads.driver", "memory"}, config.ErrInvalidUploadsDriver},
	}
	for _, tt := range tests {
		if _, err := load(t, append(tt.flags, requiredFlags...)...); !errors.Is(err, tt.want) {
			t.Errorf("%v: got %v, want %v", tt.flags, err, tt.want)
		}
	}
}

// Parallel tests are not allowed with t.Setenv
//
//nolint:paralleltest
func TestEnvConfig(t *testing.T) {
	t.Setenv("HTTP__PORT", "8087")
	t.Setenv("HTTP__METRICS__PORT", "8088")
	t.Setenv("HTTP__METRICS__IPV4_HOST", "0.0.0.0")
	t.Setenv("HTTP__METRICS__IPV6_HOST", "::0")
	t.Setenv("HTTP__IPV4_HOST", "127.0.0.1")
	t.Setenv("HTTP__IPV6_HOST", "::1")
	t.Setenv("HTTP__PPROF__ENABLED", "true")
	t.Setenv("HTTP__TRUSTED_PROXIES", "127.0.0.1,127.0.0.2")
	t.Setenv("HTTP__METRICS__ENABLED", "true")
	t.Setenv("HTTP__TRACING__ENABLED", "true")
	t.Setenv("HTTP__TRACING__OTLP_ENDPOINT", "http://localhost:4317")
	t.Setenv("HTTP__CORS_HOSTS", "http://localhost:8080,http://localhost:8081")
	t.Setenv("HTTP__BACKEND_URL", "http://localhost:8081")
	t.Setenv("PERSISTENCE__DATABASE__DRIVER", "POSTGRES")
	t.Setenv("PERSISTENCE__DATABASE__DATABASE", "test.sqlite3")
	t.Setenv("PERSISTENCE__DATABASE__HOST", "host")
	t.Setenv("PERSISTENCE__DATABASE__PORT", "5432")
	t.Setenv("PERSISTENCE__DATABASE__USERNAME", "user")
	t.Setenv("PERSISTENCE__DATABASE__PASSWORD", "password")
	t.Setenv("PERSISTENCE__DATABASE__EXTRA_PARAMETERS", "sslmode=require")
	t.Setenv("PERSISTENCE__UPLOADS__DRIVER", "filesystem")
	t.Setenv("PERSISTENCE__UPLOADS__FILESYSTEM_OPTIONS__DIRECTORY", "uploads")
	t.Setenv("PERSISTENCE__UPLOADS__S3_OPTIONS__BUCKET", "test-uploads")
	t.Setenv("PERSISTENCE__UPLOADS__S3_OPTIONS__REGION", "us-east-1")
	t.Setenv("PERSISTENCE__UPLOADS__S3_OPTIONS__ENDPOINT", "http://localhost:9000")
	t.Setenv("REGISTRATION__ENABLED", "true")
	t.Setenv("AUTH__GOOGLE__ENABLED", "true")
	t.Setenv("AUTH__GOOGLE__CLIENT_ID", "googleid")
	t.Setenv("AUTH__GOOGLE__CLIENT_SECRET", "googlesecret")
	t.Setenv("AUTH__GITHUB__ENABLED", "true")
	t.Setenv("AUTH__GITHUB__CLIENT_ID", "githubid")
	t.Setenv("AUTH__GITHUB__CLIENT_SECRET", "githubsecret")
	t.Setenv("AUTH__CUSTOM__ENABLED", "true")
	t.Setenv("AUTH__CUSTOM__CLIENT_ID", "customid")
	t.Setenv("AUTH__CUSTOM__CLIENT_SECRET", "customsecret")
	t.Setenv("AUTH__CUSTOM__TOKEN_URL", "http://localhost:8081")
	t.Setenv("AUTH__CUSTOM__USER_URL", "http://localhost:8082")
	t.Setenv("JWT__SECRET", "jwtsecret")
	t.Setenv("MAPBOX__PUBLIC_TOKEN", "mapboxpublic")
	t.Setenv("MAPBOX__SECRET_TOKEN", "mapboxsecret")
	t.Setenv("NATS__ENABLED", "true")
	t.Setenv("NATS__URL", "nats://localhost:4444")
	t.Setenv("NATS__TOKEN", "nats")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("PARALLEL_LOG_PARSERS", "8")

	cfg, err := load(t)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	checks := []struct {
		name      string
		got, want any
	}{
		{"http.port", cfg.HTTP.Port, uint16(8087)},
		{"http.metrics.port", cfg.HTTP.Metrics.Port, uint16(8088)},
		{"http.metrics.ipv4_host", cfg.HTTP.Metrics.IPV4Host, "0.0.0.0"},
		{"http.metrics.ipv6_host", cfg.HTTP.Metrics.IPV6Host, "::0"},
		{"http.ipv4_host", cfg.HTTP.IPV4Host, "127.0.0.1"},
		{"http.ipv6_host", cfg.HTTP.IPV6Host, "::1"},
		{"http.pprof.enabled", cfg.HTTP.PProf.Enabled, true},
		{"http.trusted_proxies", len(cfg.HTTP.TrustedProxies), 2},
		{"http.trusted_proxies[0]", cfg.HTTP.TrustedProxies[0], "127.0.0.1"},
		{"http.trusted_proxies[1]", cfg.HTTP.TrustedProxies[1], "127.0.0.2"},
		{"http.metrics.enabled", cfg.HTTP.Metrics.Enabled, true},
		{"http.tracing.enabled", cfg.HTTP.Tracing.Enabled, true},
		{"http.tracing.otlp_endpoint", cfg.HTTP.Tracing.OTLPEndpoint, "http://localhost:4317"},
		{"http.cors_hosts", len(cfg.HTTP.CORSHosts), 2},
		{"http.cors_hosts[0]", cfg.HTTP.CORSHosts[0], "http://localhost:8080"},
		{"http.cors_hosts[1]", cfg.HTTP.CORSHosts[1], "http://localhost:8081"},
		{"http.backend_url", cfg.HTTP.BackendURL, "http://localhost:8081"},
		{"persistence.database.driver", cfg.Persistence.Database.Driver, config.DatabaseDriverPostgres},
		{"persistence.database.database", cfg.Persistence.Database.Database, "test.sqlite3"},
		{"persistence.database.host", cfg.Persistence.Database.Host, "host"},
		{"persistence.database.port", cfg.Persistence.Database.Port, uint16(5432)},
		{"persistence.database.username", cfg.Persistence.Database.Username, "user"},
		{"persistence.database.password", cfg.Persistence.Database.Password, "password"},
		{"persistence.database.extra_parameters", cfg.Persistence.Database.ExtraParameters, "sslmode=require"},
		{"persistence.uploads.driver", cfg.Persistence.Uploads.Driver, config.UploadsDriverFilesystem},
		{"persistence.uploads.filesystem_options.directory", cfg.Persistence.Uploads.FilesystemOptions.Directory, "uploads"},
		{"persistence.uploads.s3_options.bucket", cfg.Persistence.Uploads.S3Options.Bucket, "test-uploads"},
		{"persistence.uploads.s3_options.region", cfg.Persistence.Uploads.S3Options.Region, "us-east-1"},
		{"persistence.uploads.s3_options.endpoint", cfg.Persistence.Uploads.S3Options.Endpoint, "http://localhost:9000"},
		{"registration.enabled", cfg.Registration.Enabled, true},
		{"auth.google.enabled", cfg.Auth.Google.Enabled, true},
		{"auth.google.client_id", cfg.Auth.Google.ClientID, "googleid"},
		{"auth.google.client_secret", cfg.Auth.Google.ClientSecret, "googlesecret"},
		{"auth.github.enabled", cfg.Auth.GitHub.Enabled, true},
		{"auth.github.client_id", cfg.Auth.GitHub.ClientID, "githubid"},
		{"auth.github.client_secret", cfg.Auth.GitHub.ClientSecret, "githubsecret"},
		{"auth.custom.enabled", cfg.Auth.Custom.Enabled, true},
		{"auth.custom.client_id", cfg.Auth.Custom.ClientID, "customid"},
		{"auth.custom.client_secret", cfg.Auth.Custom.ClientSecret, "customsecret"},
		{"auth.custom.token_url", cfg.Auth.Custom.TokenURL, "http://localhost:8081"},
		{"auth.custom.user_url", cfg.Auth.Custom.UserURL, "http://localhost:8082"},
		{"jwt.secret", cfg.JWT.Secret, "jwtsecret"},
		{"mapbox.public_token", cfg.Mapbox.PublicToken, "mapboxpublic"},
		{"mapbox.secret_token", cfg.Mapbox.SecretToken, "mapboxsecret"},
		{"nats.enabled", cfg.NATS.Enabled, true},
		{"nats.url", cfg.NATS.URL, "nats://localhost:4444"},
		{"nats.token", cfg.NATS.Token, "nats"},
		{"log_level", cfg.LogLevel, config.LogLevelDebug},
		{"parallel_log_parsers", cfg.ParallelLogParsers, uint(8)},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, c.got, c.want)
		}
	}
}
