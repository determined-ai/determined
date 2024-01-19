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
	files  []archive.FileEntry
}

// DefaultDownloadPartSize is the default part size for downloading files from the local filesystem.
// This is the same as the default part size for S3.
const DefaultDownloadPartSize = units.MiB * 5

func (d *LocalDownloader) archivePath(ctx context.Context, path string, size int64) error {
	f, err := os.Open(filepath.Clean(d.prefix + path))
	if err != nil {
		return err
	}
	defer func() {
		_ = f.Close()
	}()

	if err := d.aw.WriteHeader(path, size); err != nil {
		return err
	}
	for {
		if size <= 0 {
			break
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		sizeRead, err := f.Read(d.buffer)
		if err != nil {
			return err
		}
		if _, err := d.aw.Write(d.buffer[:sizeRead]); err != nil {
			return err
		}
		size -= int64(sizeRead)
	}
	return nil
}

// Download downloads the checkpoint.
func (d *LocalDownloader) Download(ctx context.Context) error {
	err := d.download(ctx)
	return errors.Wrap(err, "checkpoint archive failed, "+
		"please verify that file system checkpoint storage is available to the server")
}

func (d *LocalDownloader) download(ctx context.Context) error {
	files, err := d.ListFiles(ctx)
	if err != nil {
		return err
	}
	for _, file := range files {
		if err := d.archivePath(ctx, file.Path, file.Size); err != nil {
			return err
		}
	}
	return nil
}

// Close closes the underlying ArchiveWriter.
func (d *LocalDownloader) Close() error {
	return d.aw.Close()
}

// ListFiles lists the files in the checkpoint.
func (d *LocalDownloader) ListFiles(ctx context.Context) ([]archive.FileEntry, error) {
	if d.files != nil {
		return d.files, nil
	}
	files := make([]archive.FileEntry, 0)
	collectFiles := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		files = append(files, archive.FileEntry{
			Path: strings.TrimPrefix(path, d.prefix),
			Size: info.Size(),
		})
		return nil
	}
	if err := filepath.Walk(d.prefix, collectFiles); err != nil {
		return nil, err
	}
	d.files = files
	return d.files, nil
}

// NewLocalDownloader returns a new LocalDownloader.
func NewLocalDownloader(aw archive.ArchiveWriter, prefix string) (*LocalDownloader, error) {
	if !strings.HasPrefix(prefix, "/") {
		prefix = "/" + prefix
	}
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	return &LocalDownloader{
		aw:     aw,
		prefix: filepath.Clean(prefix),
		buffer: make([]byte, DefaultDownloadPartSize),
	}, nil
}
