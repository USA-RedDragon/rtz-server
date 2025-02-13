package storage

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

type Filesystem struct {
	Storage
	Manager

	osRoot *os.Root
}

func newFiles(root string) (Filesystem, error) {
	osRoot, err := os.OpenRoot(root)
	if err != nil {
		return Filesystem{}, err
	}
	return Filesystem{osRoot: osRoot}, nil
}

func (f Filesystem) Close() error {
	return f.osRoot.Close()
}

func (f Filesystem) Open(name string) (File, error) {
	return f.osRoot.OpenFile(name, os.O_RDONLY, 0)
}

func (f Filesystem) Create(name string) (File, error) {
	return f.osRoot.Create(name)
}

func (f Filesystem) Mkdir(name string, perm fs.FileMode) error {
	return f.osRoot.Mkdir(name, perm)
}

func (f Filesystem) MkdirAll(path string, perm fs.FileMode) error {
	// end of recursion
	if path == "" || path == "." || path == "/" {
		return nil
	}

	// try first
	err := f.Mkdir(path, perm)
	if err == nil || errors.Is(err, os.ErrExist) {
		return nil
	}

	// if it failed, try w/ parent
	err = f.MkdirAll(filepath.Dir(path), perm)
	if err != nil {
		return err
	}

	// try again
	err = f.Mkdir(path, perm)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return nil
		}
		return err
	}

	return nil
}

func (f Filesystem) Remove(name string) error {
	return f.osRoot.Remove(name)
}

func (f Filesystem) Sub(dir string) (Storage, error) {
	newRoot, err := f.osRoot.OpenRoot(dir)
	if err != nil {
		return Filesystem{}, err
	}
	return Filesystem{osRoot: newRoot}, nil
}
