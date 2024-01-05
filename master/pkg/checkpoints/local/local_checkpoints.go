package local

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/go-units"
	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/checkpoints/archive"
)

// LocalDownloader implements downloading a checkpoint from the local filesystem
// and sends it to the client in an archive file.
type LocalDownloader struct {
	aw     archive.ArchiveWriter
	prefix string
	buffer []byte
}

// DefaultDownloadPartSize is the default part size for downloading files from the local filesystem.
// This is the same as the default part size for S3.
const DefaultDownloadPartSize = units.MiB * 5

func (d *LocalDownloader) archivePath(path string, info os.FileInfo, err error) error {
	if err != nil {
		return err
	}
	if info.IsDir() {
		return nil
	}
	f, err := os.Open(filepath.Clean(path))
	if err != nil {
		return err
	}
	defer func() {
		_ = f.Close()
	}()

	relPath := strings.TrimPrefix(path, d.prefix)
	remaining := info.Size()
	if err := d.aw.WriteHeader(relPath, remaining); err != nil {
		return err
	}
	for {
		if remaining <= 0 {
			break
		}
		sizeRead, err := f.Read(d.buffer)
		if err != nil {
			return err
		}
		if _, err := d.aw.Write(d.buffer[:sizeRead]); err != nil {
			return err
		}
		remaining -= int64(sizeRead)
	}
	return nil
}

// Download downloads the checkpoint.
func (d *LocalDownloader) Download(ctx context.Context) error {
	err := filepath.Walk(d.prefix, d.archivePath)
	return errors.Wrap(err, "checkpoint archive failed, "+
		"please verify that file system checkpoint storage is available to the server")
}

// Close closes the underlying ArchiveWriter.
func (d *LocalDownloader) Close() error {
	return d.aw.Close()
}

// NewLocalDownloader returns a new LocalDownloader.
func NewLocalDownloader(aw archive.ArchiveWriter, prefix string) *LocalDownloader {
	if !strings.HasPrefix(prefix, "/") {
		prefix = "/" + prefix
	}
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	return &LocalDownloader{
		aw:     aw,
		prefix: prefix,
		buffer: make([]byte, DefaultDownloadPartSize),
	}
}
