package main

import (
	"log/slog"
	"os"

	"github.com/USA-RedDragon/rtz-server/cmd"
)

// https://goreleaser.com/cookbooks/using-main.version/
//
//nolint:golint,gochecknoglobals
var (
	version = "dev"
	commit  = "none"
)

func main() {
	rootCmd := cmd.NewCommand(version, commit)
	if err := rootCmd.Execute(); err != nil {
		slog.Error("Encountered an error.", "error", err.Error())
		os.Exit(1)
	}
}
