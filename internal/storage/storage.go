package storage

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"

	"github.com/USA-RedDragon/rtz-server/internal/config"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type Manager interface {
	Open(name string) (File, error)
	Create(name string) (File, error)
	Mkdir(name string, perm fs.FileMode) error
	MkdirAll(name string, perm fs.FileMode) error
	Remove(name string) error
	Sub(dir string) (Storage, error)
}

type File interface {
	io.ReadCloser
	io.Writer
}

type Storage interface {
	Manager
	Close() error
}

func NewStorage(cfg *config.Config) (Storage, error) {
	switch cfg.Persistence.Uploads.Driver {
	case config.UploadsDriverFilesystem:
		root := cfg.Persistence.Uploads.FilesystemOptions.Directory
		err := os.MkdirAll(root, 0755)
		if err != nil {
			return nil, fmt.Errorf("failed to create uploads directory: %w", err)
		}
		return newFiles(root)
	case config.UploadsDriverS3:
		awsCfg, err := awsConfig.LoadDefaultConfig(context.TODO())
		if err != nil {
			return nil, fmt.Errorf("failed to load AWS config: %w", err)
		}

		return newS3(
			cfg.Persistence.Uploads.S3Options.Region,
			cfg.Persistence.Uploads.S3Options.Bucket,
			"",
			s3.NewFromConfig(awsCfg, func(o *s3.Options) {
				o.Region = cfg.Persistence.Uploads.S3Options.Region
				if cfg.Persistence.Uploads.S3Options.Endpoint != "" {
					slog.Warn("using custom S3 endpoint", "endpoint", cfg.Persistence.Uploads.S3Options.Endpoint)
					o.BaseEndpoint = aws.String(cfg.Persistence.Uploads.S3Options.Endpoint)
					o.UsePathStyle = true
				}
			}))
	default:
		return nil, fmt.Errorf("unknown storage driver: %s", cfg.Persistence.Uploads.Driver)
	}
}
