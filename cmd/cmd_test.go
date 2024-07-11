package cmd_test

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

func TestDefault(t *testing.T) {
	t.Parallel()
	baseCmd := cmd.NewCommand("testing", "default")
	// Avoid port conflict
	baseCmd.SetArgs(append([]string{"--http.port", "8082", "--http.metrics.port", "8083"}, requiredFlags...))
	err := baseCmd.Execute()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
