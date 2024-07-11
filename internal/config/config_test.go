package config_test

import (
	"context"
	"testing"

	"github.com/USA-RedDragon/rtz-server/cmd"
	"github.com/USA-RedDragon/rtz-server/internal/config"
)

var requiredFlags = []string{
	"--jwt.secret", "changeme",
	"--http.backend_url", "http://localhost:8081",
	"--mapbox.secret_token", "dummy",
	"--mapbox.public_token", "dummy",
}

func TestExampleConfig(t *testing.T) {
	t.Parallel()
	baseCmd := cmd.NewCommand("testing", "deadbeef")
	// Avoid port conflict
	baseCmd.SetArgs([]string{"--config", "../../config.example.yaml", "--http.port", "8083", "--http.metrics.port", "8084"})
	err := baseCmd.Execute()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestTracing(t *testing.T) {
	t.Parallel()
	// Avoid port conflict
	baseArgs := []string{"--http.port", "8085", "--http.metrics.port", "8086", "--http.tracing.enabled", "true"}
	baseCmd := cmd.NewCommand("testing", "deadbeef")
	baseCmd.SetArgs(append(baseArgs, requiredFlags...))
	err := baseCmd.Execute()
	if err == nil {
		t.Error("Tracing enabled but OTLP endpoint not set")
	}
	baseArgs = append(baseArgs, "--http.tracing.otlp_endpoint", "http://localhost:4317")
	baseCmd.SetArgs(append(baseArgs, requiredFlags...))
	err = baseCmd.Execute()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestEnvConfig(t *testing.T) {
	cmd := cmd.NewCommand("testing", "deadbeef")
	cmd.SetContext(context.Background())
	t.Setenv("HTTP__PORT", "8087")
	t.Setenv("HTTP__METRICS__PORT", "8088")
	t.Setenv("HTTP__METRICS__IPV4_HOST", "0.0.0.0")
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
}
