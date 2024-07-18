package storage

import (
	"fmt"
	"io"
	"io/fs"
	"os"

	"github.com/USA-RedDragon/rtz-server/internal/config"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type StorageManager interface {
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
	StorageManager
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
		s3Options := s3.Options{
			Region:       cfg.Persistence.Uploads.S3Options.Region,
			UsePathStyle: true,
		}
		if cfg.Persistence.Uploads.S3Options.Endpoint != "" {
			s3Options.BaseEndpoint = aws.String(cfg.Persistence.Uploads.S3Options.Endpoint)
		}
		return newS3(
			cfg.Persistence.Uploads.S3Options.Region,
			cfg.Persistence.Uploads.S3Options.Bucket,
			"",
			s3.New(s3Options))
	default:
		return nil, fmt.Errorf("unknown storage driver: %s", cfg.Persistence.Uploads.Driver)
	}
}
