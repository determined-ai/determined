package internal

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
)

type archiveWriter interface {
	WriteFileHeader(fname string, size int64) error
	Write(b []byte) (int, error)
}

type tarArchiveWriter struct {
	tw *tar.Writer
}

func (aw *tarArchiveWriter) WriteFileHeader(fname string, size int64) error {
	hdr := tar.Header{
		Name: fname,
		Mode: 0666,
		Size: size,
	}
	if strings.HasSuffix(fname, "/") {
		// This a directory
		hdr.Mode = 0777
	}
	return aw.tw.WriteHeader(&hdr)
}

func (aw *tarArchiveWriter) Write(p []byte) (int, error) {
	return aw.tw.Write(p)
}

type zipArchiveWriter struct {
	zw        *zip.Writer
	zwContent io.Writer
}

func (aw *zipArchiveWriter) WriteFileHeader(fname string, size int64) error {
	// Zip by default sets mode 0666 and 077 for files and folders respectively
	zwc, err := aw.zw.Create(fname)
	if err != nil {
		return err
	}
	aw.zwContent = zwc
	return nil
}

func (aw *zipArchiveWriter) Write(p []byte) (int, error) {
	var w io.Writer
	if aw.zwContent == nil {
		return 0, nil
	}
	w = aw.zwContent
	return w.Write(p)
}

type delayWriter struct {
	delayBytes int
	buf        []byte
	next       io.Writer
}

func (w *delayWriter) Write(p []byte) (int, error) {
	if w.buf == nil {
		return w.next.Write(p)
	}

	w.buf = append(w.buf, p...)
	if len(w.buf) < w.delayBytes {
		return len(p), nil
	}

	n, err := w.next.Write(w.buf)
	w.buf = nil
	return n, err
}

// Flush the buffer if it is nonempty.
func (w *delayWriter) Close() error {
	if w.buf != nil && len(w.buf) > 0 {
		_, err := w.next.Write(w.buf)
		return err
	}
	return nil
}

func newDelayWriter(w io.Writer, delayBytes int) *delayWriter {
	return &delayWriter{
		delayBytes: delayBytes,
		buf:        make([]byte, 0, delayBytes),
		next:       w,
	}
}

// S3SeqWriterAt satisfies S3 APIs' io.WriterAt interface while staying sequential.
// Ref: https://docs.aws.amazon.com/sdk-for-go/api/service/s3/s3manager/#Downloader
type s3SeqWriterAt struct {
	next    io.Writer
	written int64
}

func newS3SeqWriterAt(w io.Writer) *s3SeqWriterAt {
	return &s3SeqWriterAt{next: w}
}

// WriteAt writes the content in buffer p.
func (w *s3SeqWriterAt) WriteAt(p []byte, off int64) (int, error) {
	if off != w.written {
		return 0, fmt.Errorf(
			"only supporting sequential writes,"+
				" writing at offset %d while %d bytes have been written",
			off, w.written)
	}
	n, err := w.next.Write(p)
	if err != nil {
		return 0, err
	}
	w.written += int64(n)

	return n, err
}

// BatchDownloadIterator implements s3's BatchDownloadIterator API.
type batchDownloadIterator struct {
	// The objects we are writing
	objects []*s3.Object
	// The output we are writing to
	aw archiveWriter
	// Internal states
	err    error
	pos    int
	bucket string
}

// Next() returns true if the next item is available.
func (i *batchDownloadIterator) Next() bool {
	i.pos++
	if i.pos == len(i.objects) {
		return false
	}
	err := i.aw.WriteFileHeader(*i.objects[i.pos].Key, *i.objects[i.pos].Size)
	if err != nil {
		i.err = err
		return false
	}
	return true
}

// Err() eturns the error if any.
func (i *batchDownloadIterator) Err() error {
	return i.err
}

// DownloadObject() eturns a DownloadObject.
func (i *batchDownloadIterator) DownloadObject() s3manager.BatchDownloadObject {
	return s3manager.BatchDownloadObject{
		Object: ptrs.Ptr(s3.GetObjectInput{
			Bucket: ptrs.Ptr(i.bucket),
			Key:    i.objects[i.pos].Key,
		}),
		Writer: newS3SeqWriterAt(i.aw),
	}
}

func newBatchDownloadIterator(aw archiveWriter,
	bucket string, objs []*s3.Object) *batchDownloadIterator {
	return &batchDownloadIterator{
		aw:      aw,
		bucket:  bucket,
		objects: objs,
		pos:     -1,
	}
}

func getAwsRegion() string {
	defaultRegion := "us-west-2"
	client := http.Client{
		Timeout: 100 * time.Millisecond,
	}

	// Per AWS: https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/instance-identity-documents.html
	resp, err := client.Get("http://169.254.169.254/latest/dynamic/instance-identity/document")
	if err != nil || resp.StatusCode != http.StatusOK {
		return defaultRegion
	}

	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return defaultRegion
	}
	_ = resp.Body.Close()

	var jsonObj map[string]interface{}
	err = json.Unmarshal(bytes, &jsonObj)
	if err != nil {
		return defaultRegion
	}

	if _, ok := jsonObj["region"]; !ok {
		return defaultRegion
	}

	region, ok := jsonObj["region"].(string)
	if !ok {
		return defaultRegion
	}

	return region
}

func s3DownloadCheckpoint(
	c echo.Context, aw archiveWriter, bucket string, prefix string,
) error {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(getAwsRegion()),
	})
	if err != nil {
		return err
	}
	// We do not pass in credentials explicitly. Instead, we reply on
	// the existing AWS credentials.
	s3client := s3.New(sess)

	var errs []error
	downloader := s3manager.NewDownloader(sess, func(d *s3manager.Downloader) {
		d.Concurrency = 1 // Setting concurrency to 1 to use s3SeqWriterAt
	})
	funcReadPage := func(output *s3.ListObjectsV2Output, lastPage bool) bool {
		iter := newBatchDownloadIterator(aw, bucket, output.Contents)
		// Download every bucket in this page
		err = downloader.DownloadWithIterator(c.Request().Context(), iter)
		if iter.Err() != nil {
			errs = append(errs, iter.Err())
		}
		if err != nil {
			errs = append(errs, err)
		}

		// Return False to stop paging
		return len(errs) == 0
	}
	err = s3client.ListObjectsV2PagesWithContext(
		c.Request().Context(),
		&s3.ListObjectsV2Input{
			Bucket: ptrs.Ptr(bucket),
			Prefix: ptrs.Ptr(prefix),
		},
		funcReadPage,
	)
	if err != nil {
		errs = append(errs, err)
	}
	if len(errs) > 0 {
		msg := "one or more errors encountered during checkpoint download:"
		for _, v := range errs {
			msg += fmt.Sprintf("\n  %s;", v.Error())
		}
		return errors.New(msg)
	}
	return nil
}

// It is assumed that a http status code is not sent until the first write to w.
func buildWriterPipeline(w io.Writer, mimeType string) (archiveWriter, []io.Closer, error) {
	// DelayWriter delays the first write until we have successfully downloaded
	// some bytes and are more confident that the download will succeed.
	dw := newDelayWriter(w, 16*1024)
	closers := []io.Closer{dw}
	switch mimeType {
	case "application/gzip":
		gz := gzip.NewWriter(dw)
		closers = append(closers, gz)

		tw := tar.NewWriter(gz)
		closers = append(closers, tw)

		return &tarArchiveWriter{tw}, closers, nil

	case "application/zip":
		zw := zip.NewWriter(dw)
		closers = append(closers, zw)

		return &zipArchiveWriter{zw, nil}, closers, nil

	default:
		return nil, nil, fmt.Errorf(
			"MIME type must be applicatoin/gzip or application/zip but got " + mimeType)
	}
}

func (m *Master) getCheckpoint(c echo.Context, mimeType string) error {
	args := struct {
		CheckpointUUID string `path:"checkpoint_uuid"`
	}{}
	if err := api.BindArgs(&args, c); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest,
			"invalid checkpoint_uuid: "+err.Error())
	}

	checkpointUUID, err := uuid.Parse(args.CheckpointUUID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest,
			fmt.Sprintf("unable to parse checkpoint_uuid %s: %s",
				args.CheckpointUUID, err))
	}

	// Assume a checkpoint always has experiment configs
	expConfig, err := m.db.ExperimentConfigForCheckpoint(checkpointUUID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			fmt.Sprintf("unable to retrieve experiment config for checkpoint %s: %s",
				args.CheckpointUUID, err.Error()))
	}
	if expConfig == nil {
		return echo.NewHTTPError(http.StatusNotFound,
			fmt.Sprintf("checkpoint %s does not exist", args.CheckpointUUID))
	}
	storageUnion := expConfig.CheckpointStorage()

	c.Response().Header().Set(echo.HeaderContentType, mimeType)
	writerPipe, closers, err := buildWriterPipeline(c.Response(), mimeType)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	switch storage := storageUnion.GetUnionMember().(type) {
	case expconf.S3Config:
		err = s3DownloadCheckpoint(c, writerPipe, storage.Bucket(),
			strings.TrimLeft(*storage.Prefix()+"/"+args.CheckpointUUID, "/"))

	default:
		return echo.NewHTTPError(http.StatusNotImplemented,
			fmt.Sprintf("checkpoint download via master is only supported on S3"+
				", but the checkpoint's storage config type is %T", storage))
	}
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			fmt.Sprintf("unable to download checkpoint %s: %s",
				args.CheckpointUUID, err.Error()))
	}

	for i := len(closers) - 1; i >= 0; i-- {
		err = closers[i].Close()
		if err != nil {
			return err
		}
	}
	c.Response().Flush()

	return nil
}

// @Summary Get a tarball of checkpoint contents.
// @Tags Checkpoints
// @ID get-checkpoint-tgz
// @Accept  json
// @Produce  application/gzip; charset=utf-8
// @Param   checkpoint_uuid path string  true  "Checkpoint UUID"
// @Success 200 {} string ""
//nolint:godot
// @Router /checkpoints/{checkpoint_uuid}/tgz [get]
func (m *Master) getCheckpointTgz(c echo.Context) error {
	return m.getCheckpoint(c, "application/gzip")
}

// @Summary Get a zip of checkpoint contents.
// @Tags Checkpoints
// @ID get-checkpoint-zip
// @Accept  json
// @Produce  application/zip; charset=utf-8
// @Param   checkpoint_uuid path string  true  "Checkpoint UUID"
// @Success 200 {} string ""
//nolint:godot
// @Router /checkpoints/{checkpoint_uuid}/zip [get]
func (m *Master) getCheckpointZip(c echo.Context) error {
	return m.getCheckpoint(c, "application/zip")
}
