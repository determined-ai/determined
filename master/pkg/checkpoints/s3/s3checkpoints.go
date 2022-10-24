package s3

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/hashicorp/go-multierror"

	"github.com/determined-ai/determined/master/pkg/checkpoints/archive"
)

// WriteAt writes the content in buffer p.
func (w *seqWriterAt) WriteAt(p []byte, off int64) (int, error) {
	if off != w.written {
		return 0, fmt.Errorf(
			"only supporting sequential writes,"+
				" writing at offset %d while %d bytes have been written",
			off, w.written)
	}
	n, err := w.next.Write(p)
	w.written += int64(n)
	if err != nil {
		return 0, err
	}

	return n, err
}

// GetS3BucketRegion returns the region name of the specified bucket.
// It does so by making an API call to AWS.
func GetS3BucketRegion(ctx context.Context, bucket string) (string, error) {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("us-west-2"),
	})
	if err != nil {
		return "", nil
	}

	out, err := s3.New(sess).GetBucketLocationWithContext(ctx, &s3.GetBucketLocationInput{
		Bucket: &bucket,
	})
	if err != nil {
		return "", err
	}

	return *out.LocationConstraint, nil
}

// S3Downloader implements downloading a checkpoint from S3
// and sends it to the client in an archive file.
type S3Downloader struct {
	aw     archive.ArchiveWriter
	bucket string
	prefix string
}

// Download downloads the checkpoint.
func (d *S3Downloader) Download(ctx context.Context) error {
	region, err := GetS3BucketRegion(ctx, d.bucket)
	if err != nil {
		return err
	}
	sess, err := session.NewSession(&aws.Config{
		Region: &region,
	})
	if err != nil {
		return err
	}
	// We do not pass in credentials explicitly. Instead, we reply on
	// the existing AWS credentials.
	s3client := s3.New(sess)

	var merr error
	downloader := s3manager.NewDownloader(sess, func(d *s3manager.Downloader) {
		d.Concurrency = 1 // Setting concurrency to 1 to use seqWriterAt
	})
	funcReadPage := func(output *s3.ListObjectsV2Output, lastPage bool) bool {
		iter := newBatchDownloadIterator(d.aw, d.bucket, d.prefix, output.Contents)
		// Download every bucket in this page
		err = downloader.DownloadWithIterator(ctx, iter)
		if iter.Err() != nil {
			merr = multierror.Append(merr, iter.Err())
		}
		if err != nil {
			merr = multierror.Append(merr, err)
		}

		// Return False to stop paging
		return merr == nil
	}
	err = s3client.ListObjectsV2PagesWithContext(
		ctx,
		&s3.ListObjectsV2Input{
			Bucket: &d.bucket,
			Prefix: &d.prefix,
		},
		funcReadPage,
	)
	if err != nil {
		merr = multierror.Append(merr, err)
	}
	if merr != nil {
		return fmt.Errorf("checkpoint download failed: %w", merr)
	}
	return nil
}

// Close closes the underlying ArchiveWriter.
func (d *S3Downloader) Close() error {
	return d.aw.Close()
}

// NewS3Downloader returns a new S3Downloader.
func NewS3Downloader(aw archive.ArchiveWriter, bucket string, prefix string) *S3Downloader {
	return &S3Downloader{
		aw:     aw,
		bucket: bucket,
		prefix: prefix,
	}
}

// seqWriterAt satisfies S3 APIs' io.WriterAt interface while staying sequential.
// To use it with s3manager.Downloader, its concurrency needs be set to 1.
// Ref: https://docs.aws.amazon.com/sdk-for-go/api/service/s3/s3manager/#Downloader
type seqWriterAt struct {
	next    io.Writer
	written int64
}

func newSeqWriterAt(w io.Writer) *seqWriterAt {
	return &seqWriterAt{next: w}
}

// BatchDownloadIterator implements s3's BatchDownloadIterator API.
type batchDownloadIterator struct {
	// S3 config
	bucket string
	prefix string
	// The objects we are writing
	objects []*s3.Object
	// The output we are writing to
	aw archive.ArchiveWriter
	// Internal states
	err error
	pos int
}

// Next() returns true if the next item is available.
func (i *batchDownloadIterator) Next() bool {
	i.pos++
	if i.pos == len(i.objects) {
		return false
	}
	pathname := strings.TrimPrefix(*i.objects[i.pos].Key, i.prefix)
	err := i.aw.WriteHeader(pathname, *i.objects[i.pos].Size)
	if err != nil {
		i.err = err
		return false
	}
	return true
}

// Err() returns the error if any.
func (i *batchDownloadIterator) Err() error {
	return i.err
}

// DownloadObject() returns a DownloadObject.
func (i *batchDownloadIterator) DownloadObject() s3manager.BatchDownloadObject {
	return s3manager.BatchDownloadObject{
		Object: &s3.GetObjectInput{
			Bucket: &i.bucket,
			Key:    i.objects[i.pos].Key,
		},
		Writer: newSeqWriterAt(i.aw),
	}
}

func newBatchDownloadIterator(aw archive.ArchiveWriter,
	bucket string, prefix string, objs []*s3.Object,
) *batchDownloadIterator {
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	return &batchDownloadIterator{
		aw:      aw,
		bucket:  bucket,
		prefix:  prefix,
		objects: objs,
		pos:     -1,
	}
}
