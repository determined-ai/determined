package gcs

import (
	"context"
	"fmt"
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
	bucket string
	prefix string
	buffer []byte
}

// DefaultDownloadPartSize is the default part size for downloading files from GCS.
// This is the same as the default part size for S3.
const DefaultDownloadPartSize = units.MiB * 5

func (d *GCSDownloader) fileDownload(
	ctx context.Context,
	b *storage.BucketHandle,
	o *storage.ObjectAttrs,
) error {
	r, err := b.Object(o.Name).NewReader(ctx)
	if err != nil {
		return fmt.Errorf("error creating gcs object reader: %w", err)
	}
	defer func() {
		_ = r.Close()
	}()
	if err := d.aw.WriteHeader(strings.TrimPrefix(o.Name, d.prefix), o.Size); err != nil {
		return fmt.Errorf("error writing gcs object header: %w", err)
	}
	remaining := o.Size
	for {
		if remaining <= 0 {
			break
		}
		sizeRead, err := r.Read(d.buffer)
		if err != nil {
			return fmt.Errorf("error reading from gcs object buffer: %w", err)
		}
		if _, err := d.aw.Write(d.buffer[:sizeRead]); err != nil {
			return fmt.Errorf("error writing gcs object buffer result: %w", err)
		}
		remaining -= int64(sizeRead)
	}
	return nil
}

func (d *GCSDownloader) download(ctx context.Context) error {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("error creating new gcs storage client: %w", err)
	}
	defer func() {
		_ = client.Close()
	}()
	bucket := client.Bucket(d.bucket)
	items := bucket.Objects(ctx, &storage.Query{Prefix: d.prefix})
	for {
		item, err := items.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return fmt.Errorf("error getting next bucket object from gcs: %w", err)
		}
		if err = d.fileDownload(ctx, bucket, item); err != nil {
			return fmt.Errorf("error downloading file from gcs bucket: %w", err)
		}
	}
	return nil
}

// Download downloads the checkpoint.
func (d *GCSDownloader) Download(ctx context.Context) error {
	if err := d.download(ctx); err != nil {
		return fmt.Errorf("checkpoint download failed: %w", err)
	}
	return nil
}

// Close closes the underlying ArchiveWriter.
func (d *GCSDownloader) Close() error {
	if err := d.aw.Close(); err != nil {
		return fmt.Errorf("error closing gcs downloader: %w", err)
	}
	return nil
}

// NewGCSDownloader returns a new GCSDownloader.
func NewGCSDownloader(aw archive.ArchiveWriter, bucket string, prefix string) *GCSDownloader {
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	return &GCSDownloader{
		aw:     aw,
		bucket: bucket,
		prefix: prefix,
		buffer: make([]byte, DefaultDownloadPartSize),
	}
}
