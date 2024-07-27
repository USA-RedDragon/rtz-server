package storage

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

type Filesystem struct {
	Storage
	Manager

	root string
	dfd  int
}

func newFiles(root string) (Filesystem, error) {
	dfd, err := unix.Open(root, unix.O_DIRECTORY|unix.O_PATH|unix.O_CLOEXEC, 0)
	if err != nil {
		return Filesystem{}, err
	}
	return Filesystem{
		root: root,
		dfd:  dfd,
	}, nil
}

func (f Filesystem) Close() error {
	return unix.Close(f.dfd)
}

func (f Filesystem) Open(name string) (File, error) {
	return f.openFile(name, os.O_RDONLY, 0)
}

func (f Filesystem) Create(name string) (File, error) {
	return f.openFile(name, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
}

func (f Filesystem) openParentOf(name string) (*os.File, error) {
	parentPath := filepath.Dir(name)
	hackpadFile, err := f.openFile(parentPath, unix.O_DIRECTORY|unix.O_PATH, 0)
	if err != nil {
		return nil, err
	}
	file, ok := hackpadFile.(*os.File)
	if !ok {
		return nil, fmt.Errorf("unexpected file type: %T", hackpadFile)
	}
	return file, nil
}

func (f Filesystem) Mkdir(name string, perm fs.FileMode) error {
	// same as above: open the new parent
	parentDfile, err := f.openParentOf(name)
	if err != nil {
		return err
	}
	defer parentDfile.Close()

	err = unix.Mkdirat(int(parentDfile.Fd()), filepath.Base(name), uint32(perm))
	if err != nil {
		return err
	}

	return nil
}

func (f Filesystem) MkdirAll(path string, perm fs.FileMode) error {
	// end of recursion
	if path == "" || path == "." || path == "/" {
		return nil
	}

	// try first
	err := f.Mkdir(path, perm)
	if err == nil || errors.Is(err, unix.EEXIST) {
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
		if errors.Is(err, unix.EEXIST) {
			return nil
		}
		return err
	}

	return nil
}

func (f Filesystem) openFile(name string, flag int, perm fs.FileMode) (File, error) {
	// openat2 RESOLVE_IN_ROOT - so symlinks still work
	for {
		how := unix.OpenHow{
			Flags:   uint64(flag) | unix.O_CLOEXEC,
			Mode:    uint64(perm),
			Resolve: unix.RESOLVE_IN_ROOT,
		}
		fd, err := unix.Openat2(f.dfd, name, &how)
		if err != nil {
			// need to check for EINTR - Go issues 11180, 39237
			// also EAGAIN in case of unsafe race
			if errors.Is(err, unix.EINTR) || errors.Is(err, unix.EAGAIN) {
				continue
			}
			return nil, err
		}

		return os.NewFile(uintptr(fd), name), nil
	}
}

func (f Filesystem) Remove(name string) error {
	// tricky: we have to open the *parent*, then unlinkat
	//
	// unlinkat has no RESOLVE_IN_ROOT, AT_EMPTY_PATH, or AT_SYMLINK_NOFOLLOW
	parentDfile, err := f.openParentOf(name)
	if err != nil {
		return err
	}
	defer parentDfile.Close()

	err = unix.Unlinkat(int(parentDfile.Fd()), filepath.Base(name), 0)
	if err != nil {
		// try rmdir like Go
		return unix.Unlinkat(int(parentDfile.Fd()), filepath.Base(name), unix.AT_REMOVEDIR)
	}

	return nil
}

func (f Filesystem) Sub(dir string) (Storage, error) {
	return newFiles(filepath.Join(f.root, dir))
}
