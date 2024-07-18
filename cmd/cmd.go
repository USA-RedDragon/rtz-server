package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"syscall"
	"time"

	"github.com/USA-RedDragon/rtz-server/internal/config"
	"github.com/USA-RedDragon/rtz-server/internal/db"
	"github.com/USA-RedDragon/rtz-server/internal/logparser"
	"github.com/USA-RedDragon/rtz-server/internal/metrics"
	"github.com/USA-RedDragon/rtz-server/internal/server"
	"github.com/USA-RedDragon/rtz-server/internal/storage"
	"github.com/nats-io/nats.go"
	"github.com/spf13/cobra"
	"github.com/ztrue/shutdown"
	"golang.org/x/sync/errgroup"
)

func NewCommand(version, commit string) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "rtz-server",
		Version: fmt.Sprintf("%s - %s", version, commit),
		Annotations: map[string]string{
			"version": version,
			"commit":  commit,
		},
		RunE:          run,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	config.RegisterFlags(cmd)
	return cmd
}

func run(cmd *cobra.Command, _ []string) error {
	slog.Info("rtz-server", "version", cmd.Annotations["version"], "commit", cmd.Annotations["commit"])

	cfg, err := config.LoadConfig(cmd)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	switch cfg.LogLevel {
	case config.LogLevelDebug:
		slog.SetLogLoggerLevel(slog.LevelDebug)
	case config.LogLevelInfo:
		slog.SetLogLoggerLevel(slog.LevelInfo)
	case config.LogLevelWarn:
		slog.SetLogLoggerLevel(slog.LevelWarn)
	case config.LogLevelError:
		slog.SetLogLoggerLevel(slog.LevelError)
	}

	err = cfg.Validate()
	if err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	storage, err := storage.NewStorage(cfg)
	if err != nil {
		return fmt.Errorf("failed to create storage: %w", err)
	}

	var nc *nats.Conn
	if cfg.NATS.Enabled {
		nc, err = nats.Connect(cfg.NATS.URL, nats.Token(cfg.NATS.Token))
		if err != nil {
			return fmt.Errorf("failed to connect to NATS: %w", err)
		}
	}

	metrics := metrics.NewMetrics()

	db, err := db.MakeDB(cfg)
	if err != nil {
		return fmt.Errorf("failed to make database: %w", err)
	}
	slog.Info("Database connection established")

	logQueue := logparser.NewLogQueue(cfg, db, storage, metrics)
	go logQueue.Start()

	slog.Info("Starting HTTP server")
	server := server.NewServer(cfg, db, nc, logQueue, metrics, storage)
	err = server.Start()
	if err != nil {
		return fmt.Errorf("failed to start HTTP server: %w", err)
	}

	stop := func(_ os.Signal) {
		slog.Info("Shutting down")

		errGrp := errgroup.Group{}

		errGrp.Go(func() error {
			return server.Stop()
		})

		errGrp.Go(func() error {
			logQueue.Stop()
			return nil
		})

		errGrp.Go(func() error {
			return storage.Close()
		})

		if cfg.NATS.Enabled {
			errGrp.Go(func() error {
				return nc.Drain()
			})
		}

		err := errGrp.Wait()
		if err != nil {
			slog.Error("Shutdown error", "error", err.Error())
		}
		slog.Info("Shutdown complete")
	}

	if cmd.Annotations["version"] == "testing" {
		doneChannel := make(chan struct{})
		go func() {
			slog.Info("Sleeping for 5 seconds")
			time.Sleep(5 * time.Second)
			slog.Info("Sending SIGTERM")
			stop(syscall.SIGTERM)
			doneChannel <- struct{}{}
		}()
		<-doneChannel
	} else {
		shutdown.AddWithParam(stop)
		shutdown.Listen(syscall.SIGINT, syscall.SIGKILL, syscall.SIGTERM, syscall.SIGQUIT)
	}

	return nil
}
