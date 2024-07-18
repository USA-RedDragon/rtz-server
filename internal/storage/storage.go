package storage

import (
	"fmt"
	"io"
	"io/fs"
	"os"

	"github.com/USA-RedDragon/rtz-server/internal/config"
)

type StorageManager interface {
	Open(name string) (File, error)
	OpenFile(name string, flag int, perm fs.FileMode) (File, error)
	Create(name string) (File, error)
	Mkdir(name string, perm fs.FileMode) error
	MkdirAll(name string, perm fs.FileMode) error
	ReadDir(name string) ([]fs.DirEntry, error)
	ReadFile(name string) ([]byte, error)
	Remove(name string) error
	WriteFile(name string, data []byte, perm fs.FileMode) error
	Sub(dir string) (Storage, error)
}

type File interface {
	fs.File
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
		return nil, fmt.Errorf("S3 storage not implemented")
	case config.UploadsDriverMemory:
		return nil, fmt.Errorf("memory storage not implemented")
	default:
		return nil, fmt.Errorf("unknown storage driver: %s", cfg.Persistence.Uploads.Driver)
	}
}
