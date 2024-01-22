package gcs

import (
	"context"
	"strings"

	"cloud.google.com/go/storage"
	"github.com/docker/go-units"
	"google.golang.org/api/iterator"

	"github.com/determined-ai/determined/master/pkg/checkpoints/archive"
)

// GCSDownloader implements downloading a checkpoint from GCS
// and sends it to the client in an archive file.
type GCSDownloader struct {
	aw     archive.ArchiveWriter
	client *storage.Client
	bucket *storage.BucketHandle
	prefix string
	buffer []byte
	files  []archive.FileEntry
}

// DefaultDownloadPartSize is the default part size for downloading files from GCS.
// This is the same as the default part size for S3.
const DefaultDownloadPartSize = units.MiB * 5

func (d *GCSDownloader) archiveDownload(ctx context.Context, path string, size int64) error {
	r, err := d.bucket.Object(d.prefix + path).NewReader(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = r.Close()
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
		sizeRead, err := r.Read(d.buffer)
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
func (d *GCSDownloader) Download(ctx context.Context) error {
	files, err := d.ListFiles(ctx)
	if err != nil {
		return err
	}
	for _, file := range files {
		if err := d.archiveDownload(ctx, file.Path, file.Size); err != nil {
			return err
		}
	}
	return nil
}

// Close closes the underlying ArchiveWriter.
func (d *GCSDownloader) Close() error {
	_ = d.client.Close()
	return d.aw.Close()
}

// ListFiles lists the files in the checkpoint.
func (d *GCSDownloader) ListFiles(ctx context.Context) ([]archive.FileEntry, error) {
	if d.files != nil {
		return d.files, nil
	}
	files := make([]archive.FileEntry, 0)

	items := d.bucket.Objects(ctx, &storage.Query{Prefix: d.prefix})
	for {
		item, err := items.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if strings.HasSuffix(item.Name, "/") {
			continue
		}
		files = append(files, archive.FileEntry{
			Path: strings.TrimPrefix(item.Name, d.prefix),
			Size: item.Size,
		})
	}
	d.files = files
	return d.files, nil
}

// NewGCSDownloader returns a new GCSDownloader.
func NewGCSDownloader(
	ctx context.Context,
	aw archive.ArchiveWriter,
	bucket string,
	prefix string,
) (*GCSDownloader, error) {
	prefix = strings.TrimLeft(prefix, "/")
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, err
	}
	return &GCSDownloader{
		aw:     aw,
		client: client,
		bucket: client.Bucket(bucket),
		prefix: prefix,
		buffer: make([]byte, DefaultDownloadPartSize),
	}, nil
}
