package config_test

import (
	"context"
	"errors"
	"testing"

	"github.com/USA-RedDragon/rtz-server/cmd"
	"github.com/USA-RedDragon/rtz-server/internal/config"
)

//nolint:golint,gochecknoglobals
var requiredFlags = []string{
	"--jwt.secret", "changeme",
	"--http.frontend_url", "http://localhost:8082",
	"--http.backend_url", "http://localhost:8081",
	"--mapbox.secret_token", "dummy",
	"--mapbox.public_token", "dummy",
}

func TestExampleConfig(t *testing.T) {
	t.Parallel()
	cmd := cmd.NewCommand("testing", "deadbeef")
	cmd.SetContext(context.Background())
	err := cmd.ParseFlags([]string{"--config", "../../config.example.yaml"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	testConfig, err := config.LoadConfig(cmd)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if err := testConfig.Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TesMissingOLTPEndpoint(t *testing.T) {
	t.Parallel()

	cmd := cmd.NewCommand("testing", "deadbeef")
	cmd.SetContext(context.Background())
	err := cmd.ParseFlags(append([]string{"--http.tracing.enabled", "true"}, requiredFlags...))
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	testConfig, err := config.LoadConfig(cmd)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if err := testConfig.Validate(); !errors.Is(err, config.ErrOTLPEndpointRequired) {
		t.Errorf("unexpected error: %v", err)
	}

	err = cmd.ParseFlags(append([]string{"--http.tracing.enabled", "true", "--http.tracing.otlp_endpoint", "dummy"}, requiredFlags...))
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	testConfig, err = config.LoadConfig(cmd)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if err := testConfig.Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestMissingJWTSecret(t *testing.T) {
	t.Parallel()
	cmd := cmd.NewCommand("testing", "deadbeef")
	cmd.SetContext(context.Background())
	err := cmd.ParseFlags([]string{
		"--http.backend_url", "http://localhost:8081",
		"--mapbox.secret_token", "dummy",
		"--mapbox.public_token", "dummy",
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	testConfig, err := config.LoadConfig(cmd)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if err := testConfig.Validate(); !errors.Is(err, config.ErrJWTSecretRequired) {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestMissingMapboxTokens(t *testing.T) {
	t.Parallel()
	baseCmd := cmd.NewCommand("testing", "deadbeef")
	baseCmd.SetContext(context.Background())
	baseFlags := []string{"--jwt.secret", "changeme", "--http.backend_url", "http://localhost:8081", "--http.frontend_url", "http://localhost:8083"}
	err := baseCmd.ParseFlags(append(baseFlags, []string{"--mapbox.secret_token", "dummy"}...))
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	testConfig, err := config.LoadConfig(baseCmd)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if err := testConfig.Validate(); !errors.Is(err, config.ErrMapboxPublicTokenRequired) {
		t.Errorf("unexpected error: %v", err)
	}
	baseCmd = cmd.NewCommand("testing", "deadbeef")
	baseCmd.SetContext(context.Background())
	err = baseCmd.ParseFlags(append(baseFlags, []string{"--mapbox.public_token", "dummy"}...))
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	testConfig, err = config.LoadConfig(baseCmd)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if err := testConfig.Validate(); !errors.Is(err, config.ErrMapboxSecretTokenRequired) {
		t.Errorf("unexpected error: %v", err)
	}
}

// Parallel tests are not allowed with t.Setenv
//
//nolint:golint,paralleltest
func TestEnvConfig(t *testing.T) {
	cmd := cmd.NewCommand("testing", "deadbeef")
	cmd.SetContext(context.Background())
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
	t.Setenv("PERSISTENCE__DATABASE__DRIVER", "postgres")
	t.Setenv("PERSISTENCE__DATABASE__DATABASE", "test.sqlite3")
	t.Setenv("PERSISTENCE__DATABASE__HOST", "host")
	t.Setenv("PERSISTENCE__DATABASE__PORT", "5432")
	t.Setenv("PERSISTENCE__DATABASE__USERNAME", "user")
	t.Setenv("PERSISTENCE__DATABASE__PASSWORD", "password")
	t.Setenv("PERSISTENCE__DATABASE__EXTRA_PARAMETERS", "sslmode=require")
	t.Setenv("PERSISTENCE__UPLOADS", "notuploads")
	t.Setenv("REGISTRATION__ENABLED", "true")
	t.Setenv("AUTH__GOOGLE__CLIENT_ID", "googleid")
	t.Setenv("AUTH__GOOGLE__CLIENT_SECRET", "googlesecret")
	t.Setenv("AUTH__GITHUB__CLIENT_ID", "githubid")
	t.Setenv("AUTH__GITHUB__CLIENT_SECRET", "githubsecret")
	t.Setenv("REDIS__ENABLED", "true")
	t.Setenv("REDIS__ADDRESS", "localhost:6379")
	t.Setenv("REDIS__USERNAME", "user123")
	t.Setenv("REDIS__PASSWORD", "password")
	t.Setenv("REDIS__DATABASE", "0")
	t.Setenv("REDIS__SENTINEL__ENABLED", "true")
	t.Setenv("REDIS__SENTINEL__ADDRESSES", "localhost:26379,localhost:26380")
	t.Setenv("REDIS__SENTINEL__MASTER_NAME", "master")
	t.Setenv("REDIS__SENTINEL__USERNAME", "user")
	t.Setenv("REDIS__SENTINEL__PASSWORD", "password")

	config, err := config.LoadConfig(cmd)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if config.HTTP.Port != 8087 {
		t.Errorf("unexpected HTTP port: %d", config.HTTP.Port)
	}
	if config.HTTP.Metrics.Port != 8088 {
		t.Errorf("unexpected HTTP metrics port: %d", config.HTTP.Metrics.Port)
	}
	if config.HTTP.Metrics.IPV4Host != "0.0.0.0" {
		t.Errorf("unexpected HTTP metrics IPv4 host: %s", config.HTTP.Metrics.IPV4Host)
	}
	if config.HTTP.Metrics.IPV6Host != "::0" {
		t.Errorf("unexpected HTTP metrics IPv6 host: %s", config.HTTP.Metrics.IPV6Host)
	}
	if config.HTTP.IPV4Host != "127.0.0.1" {
		t.Errorf("unexpected HTTP IPv4 host: %s", config.HTTP.IPV4Host)
	}
	if config.HTTP.IPV6Host != "::1" {
		t.Errorf("unexpected HTTP IPv6 host: %s", config.HTTP.IPV6Host)
	}
	if !config.HTTP.PProf.Enabled {
		t.Error("unexpected HTTP pprof enabled")
	}
	if len(config.HTTP.TrustedProxies) != 2 {
		t.Errorf("unexpected HTTP trusted proxies: %v", config.HTTP.TrustedProxies)
	}
	if config.HTTP.TrustedProxies[0] != "127.0.0.1" {
		t.Errorf("unexpected HTTP trusted proxy: %s", config.HTTP.TrustedProxies[0])
	}
	if config.HTTP.TrustedProxies[1] != "127.0.0.2" {
		t.Errorf("unexpected HTTP trusted proxy: %s", config.HTTP.TrustedProxies[1])
	}
	if !config.HTTP.Metrics.Enabled {
		t.Error("unexpected HTTP metrics enabled")
	}
	if !config.HTTP.Tracing.Enabled {
		t.Error("unexpected HTTP tracing enabled")
	}
	if config.HTTP.Tracing.OTLPEndpoint != "http://localhost:4317" {
		t.Errorf("unexpected HTTP tracing OTLP endpoint: %s", config.HTTP.Tracing.OTLPEndpoint)
	}
	if len(config.HTTP.CORSHosts) != 2 {
		t.Errorf("unexpected HTTP CORS hosts: %v", config.HTTP.CORSHosts)
	}
	if config.HTTP.CORSHosts[0] != "http://localhost:8080" {
		t.Errorf("unexpected HTTP CORS host: %s", config.HTTP.CORSHosts[0])
	}
	if config.HTTP.CORSHosts[1] != "http://localhost:8081" {
		t.Errorf("unexpected HTTP CORS host: %s", config.HTTP.CORSHosts[1])
	}
	if config.HTTP.BackendURL != "http://localhost:8081" {
		t.Errorf("unexpected HTTP backend URL: %s", config.HTTP.BackendURL)
	}
	if config.Persistence.Database.Database != "test.sqlite3" {
		t.Errorf("unexpected persistence database: %s", config.Persistence.Database.Database)
	}
	if config.Persistence.Database.Driver != "postgres" {
		t.Errorf("unexpected persistence driver: %s", config.Persistence.Database.Driver)
	}
	if config.Persistence.Database.Host != "host" {
		t.Errorf("unexpected persistence host: %s", config.Persistence.Database.Host)
	}
	if config.Persistence.Database.Port != 5432 {
		t.Errorf("unexpected persistence port: %d", config.Persistence.Database.Port)
	}
	if config.Persistence.Database.Username != "user" {
		t.Errorf("unexpected persistence username: %s", config.Persistence.Database.Username)
	}
	if config.Persistence.Database.Password != "password" {
		t.Errorf("unexpected persistence password: %s", config.Persistence.Database.Password)
	}
	if config.Persistence.Database.ExtraParameters != "sslmode=require" {
		t.Errorf("unexpected persistence extra parameters: %s", config.Persistence.Database.ExtraParameters)
	}
	if config.Persistence.Uploads != "notuploads" {
		t.Errorf("unexpected persistence uploads: %s", config.Persistence.Uploads)
	}
	if !config.Registration.Enabled {
		t.Error("unexpected registration enabled")
	}
	if config.Auth.Google.ClientID != "googleid" {
		t.Errorf("unexpected Google client ID: %s", config.Auth.Google.ClientID)
	}
	if config.Auth.Google.ClientSecret != "googlesecret" {
		t.Errorf("unexpected Google client secret: %s", config.Auth.Google.ClientSecret)
	}
	if config.Auth.GitHub.ClientID != "githubid" {
		t.Errorf("unexpected GitHub client ID: %s", config.Auth.GitHub.ClientID)
	}
	if config.Auth.GitHub.ClientSecret != "githubsecret" {
		t.Errorf("unexpected GitHub client secret: %s", config.Auth.GitHub.ClientSecret)
	}
	if !config.Redis.Enabled {
		t.Error("unexpected Redis enabled")
	}
	if config.Redis.Address != "localhost:6379" {
		t.Errorf("unexpected Redis address: %s", config.Redis.Address)
	}
	if config.Redis.Username != "user123" {
		t.Errorf("unexpected Redis username: %s", config.Redis.Username)
	}
	if config.Redis.Password != "password" {
		t.Errorf("unexpected Redis password: %s", config.Redis.Password)
	}
	if config.Redis.Database != 0 {
		t.Errorf("unexpected Redis database: %d", config.Redis.Database)
	}
	if !config.Redis.Sentinel.Enabled {
		t.Error("unexpected Redis sentinel enabled")
	}
	if len(config.Redis.Sentinel.Addresses) != 2 {
		t.Errorf("unexpected Redis sentinel hosts: %v", config.Redis.Sentinel.Addresses)
	}
	if config.Redis.Sentinel.Addresses[0] != "localhost:26379" {
		t.Errorf("unexpected Redis sentinel host: %s", config.Redis.Sentinel.Addresses[0])
	}
	if config.Redis.Sentinel.Addresses[1] != "localhost:26380" {
		t.Errorf("unexpected Redis sentinel host: %s", config.Redis.Sentinel.Addresses[1])
	}
	if config.Redis.Sentinel.MasterName != "master" {
		t.Errorf("unexpected Redis sentinel master: %s", config.Redis.Sentinel.MasterName)
	}
	if config.Redis.Sentinel.Username != "user" {
		t.Errorf("unexpected Redis sentinel username: %s", config.Redis.Sentinel.Username)
	}
	if config.Redis.Sentinel.Password != "password" {
		t.Errorf("unexpected Redis sentinel password: %s", config.Redis.Sentinel.Password)
	}
}
