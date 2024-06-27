package cmd_test

import (
	"testing"

	"github.com/USA-RedDragon/connect-server/cmd"
)

func TestDefault(t *testing.T) {
	t.Parallel()
	baseCmd := cmd.NewCommand("testing", "default")
	// Avoid port conflict
	baseCmd.SetArgs([]string{"--http.port", "8082", "--http.metrics.port", "8083"})
	err := baseCmd.Execute()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
