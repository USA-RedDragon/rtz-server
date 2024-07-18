package storage

import (
	"bytes"
	"context"
	"io"
	"io/fs"
	"log/slog"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"golang.org/x/sync/errgroup"
)

type S3 struct {
	Storage
	StorageManager

	root     string
	region   string
	bucket   string
	s3Client *s3.Client
}

type bufferWriter struct {
	io.Writer
	buffer []byte
}

func (w *bufferWriter) Write(p []byte) (n int, err error) {
	w.buffer = append(w.buffer, p...)
	return len(p), nil
}

type S3File struct {
	File
	key        string
	body       io.ReadCloser
	writer     bufferWriter
	hasWritten bool
	filesystem *S3
}

func (f S3File) Write(p []byte) (n int, err error) {
	f.hasWritten = true
	return f.writer.Write(p)
}

func (f S3File) Close() error {
	errGrp := errgroup.Group{}
	errGrp.Go(func() error {
		return f.body.Close()
	})
	// Write the buffer to S3
	if f.hasWritten {
		errGrp.Go(func() error {
			_, err := f.filesystem.s3Client.PutObject(context.TODO(), &s3.PutObjectInput{
				Bucket: aws.String(f.filesystem.bucket),
				Key:    aws.String(f.key),
				Body:   bytes.NewReader(f.writer.buffer),
			})
			return err
		})
	}

	return errGrp.Wait()
}

func newS3(region, bucket, root string, s3Client *s3.Client) (S3, error) {
	return S3{
		region:   region,
		bucket:   bucket,
		root:     root,
		s3Client: s3Client,
	}, nil
}

func (s S3) Close() error {
	return nil
}

func (s S3) Open(name string) (File, error) {
	res, err := s.s3Client.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(filepath.Join(s.root, name)),
	})
	if err != nil {
		slog.Error("failed to open file", "error", err)
		return nil, err
	}

	return S3File{
		body:       res.Body,
		filesystem: &s,
		key:        filepath.Join(s.root, name),
		writer:     bufferWriter{buffer: make([]byte, 0)},
	}, nil
}

func (s S3) Mkdir(name string, perm fs.FileMode) error {
	// No-op: S3 doesn't have directories
	return nil
}

func (s S3) MkdirAll(path string, perm fs.FileMode) error {
	// No-op: S3 doesn't have directories
	return nil
}

func (s S3) Sub(dir string) (Storage, error) {
	return newS3(s.region, s.bucket, filepath.Join(s.root, dir), s.s3Client)
}

func (s S3) Create(name string) (File, error) {
	return S3File{
		filesystem: &s,
		key:        filepath.Join(s.root, name),
		writer:     bufferWriter{buffer: make([]byte, 0)},
	}, nil
}

func (s S3) Remove(name string) error {
	_, err := s.s3Client.DeleteObject(context.TODO(), &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(filepath.Join(s.root, name)),
	})
	return err
}
