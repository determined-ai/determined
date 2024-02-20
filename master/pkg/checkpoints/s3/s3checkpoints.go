package s3

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"

	"github.com/determined-ai/determined/master/pkg/checkpoints/archive"
	"github.com/determined-ai/determined/master/pkg/ptrs"
)

const (
	// awsEndpointURL is the AWS endpoint format for getting bucket regions.
	awsEndpointURL = "https://%s.s3.amazonaws.com"
)

// S3Downloader implements downloading a checkpoint from S3
// and sends it to the client in an archive file.
type S3Downloader struct {
	aw         archive.ArchiveWriter
	client     *s3.S3
	downloader *s3manager.Downloader
	bucket     string
	prefix     string
	files      []archive.FileEntry
}

func (d *S3Downloader) archiveDownload(ctx context.Context, path string, size int64) error {
	if err := d.aw.WriteHeader(path, size); err != nil {
		return err
	}
	_, err := d.downloader.DownloadWithContext(ctx, newSeqWriterAt(d.aw),
		&s3.GetObjectInput{
			Bucket: &d.bucket,
			Key:    ptrs.Ptr(d.prefix + path),
		})
	return err
}

// Download downloads the checkpoint.
func (d *S3Downloader) Download(ctx context.Context) error {
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
func (d *S3Downloader) Close() error {
	return d.aw.Close()
}

// ListFiles lists the files in the checkpoint.
func (d *S3Downloader) ListFiles(ctx context.Context) ([]archive.FileEntry, error) {
	if d.files != nil {
		return d.files, nil
	}
	files := make([]archive.FileEntry, 0)
	funcReadPage := func(output *s3.ListObjectsV2Output, lastPage bool) bool {
		for _, obj := range output.Contents {
			if strings.HasSuffix(*obj.Key, "/") {
				continue
			}
			files = append(files, archive.FileEntry{
				Path: strings.TrimPrefix(*obj.Key, d.prefix),
				Size: *obj.Size,
			})
		}
		return true
	}
	err := d.client.ListObjectsV2PagesWithContext(
		ctx,
		&s3.ListObjectsV2Input{
			Bucket: &d.bucket,
			Prefix: &d.prefix,
		},
		funcReadPage,
	)
	if err != nil {
		return nil, err
	}
	d.files = files
	return d.files, nil
}

// NewS3Downloader returns a new S3Downloader.
func NewS3Downloader(
	ctx context.Context,
	aw archive.ArchiveWriter,
	bucket string,
	prefix string,
	endpointURL *string,
) (*S3Downloader, error) {
	prefix = strings.TrimLeft(prefix, "/")
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	// We do not pass in credentials explicitly. Instead, we reply on
	// the existing AWS credentials.
	var endpointFormat *string
	if endpointURL != nil {
		format := fmt.Sprint(*endpointURL, "/%s")
		endpointFormat = &format
	}

	// if endpointFormat is nil, defaults to aws endpoint
	region, err := GetS3BucketRegion(ctx, bucket, endpointFormat)
	if err != nil {
		return nil, err
	}

	awsConfig := &aws.Config{Region: &region}

	// configure for non-aws S3 providers
	if endpointURL != nil {
		awsConfig.Endpoint = endpointURL
		awsConfig.DisableSSL = aws.Bool(false)
		awsConfig.S3ForcePathStyle = aws.Bool(true)
	}

	sess, err := session.NewSession(awsConfig)
	if err != nil {
		return nil, err
	}

	return &S3Downloader{
		aw:     aw,
		client: s3.New(sess),
		downloader: s3manager.NewDownloader(sess, func(d *s3manager.Downloader) {
			d.Concurrency = 1 // Setting concurrency to 1 to use seqWriterAt
		}),
		bucket: bucket,
		prefix: prefix,
	}, nil
}

// GetS3BucketRegion returns the region name of the specified bucket.
// It does so by making an API call to either the provided endpoint or AWS.
func GetS3BucketRegion(ctx context.Context, bucket string, endpointURL *string) (string, error) {
	// We can't use the AWS SDK for getting bucket region
	// because we get a 403 when the region in the client is different
	// than the bucket (defeating the whole point of calling bucket location).
	// Instead just use the HEAD API since this is a lot simpler and doesn't require any auth.
	// https://github.com/aws/aws-sdk-go/issues/720

	var url string
	if endpointURL != nil {
		url = fmt.Sprintf(*endpointURL, bucket)
	} else {
		url = fmt.Sprintf(awsEndpointURL, bucket)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
	if err != nil {
		return "", fmt.Errorf("making request to get region of s3 bucket at url %s: %w", url, err)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("getting region of s3 bucket at url %s: %w", url, err)
	}
	if err := res.Body.Close(); err != nil {
		return "", fmt.Errorf("closing s3 bucket request body: %w", err)
	}

	return res.Header.Get("X-Amz-Bucket-Region"), nil
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
