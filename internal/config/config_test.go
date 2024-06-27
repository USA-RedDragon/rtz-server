package config_test

import (
	"testing"

	"github.com/USA-RedDragon/connect-server/cmd"
)

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
	baseCmd := cmd.NewCommand("testing", "deadbeef")
	// Avoid port conflict
	baseCmd.SetArgs([]string{"--http.port", "8085", "--http.metrics.port", "8086", "--http.tracing.enabled", "true"})
	err := baseCmd.Execute()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestEnvConfig(t *testing.T) {
	t.Setenv("HTTP__PORT", "8087")
	t.Setenv("HTTP__METRICS__PORT", "8088")
	baseCmd := cmd.NewCommand("testing", "deadbeef")
	err := baseCmd.Execute()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
