package config_test

import (
	"testing"

	"github.com/USA-RedDragon/rtz-server/cmd"
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
	t.Setenv("HTTP__PORT", "8087")
	t.Setenv("HTTP__METRICS__PORT", "8088")
	baseCmd := cmd.NewCommand("testing", "deadbeef")
	baseCmd.SetArgs(requiredFlags)
	err := baseCmd.Execute()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
