package internal

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/determined-ai/determined/master/pkg/checkpoints"
	"github.com/determined-ai/determined/master/pkg/checkpoints/archive"

	"github.com/determined-ai/determined/master/internal/api"
	detContext "github.com/determined-ai/determined/master/internal/context"
	expauth "github.com/determined-ai/determined/master/internal/experiment"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
)

const (
	// MIMEApplicationGZip is GZip's MIME type.
	MIMEApplicationGZip = "application/gzip"
	// MIMEApplicationZip is Zip's MIME type.
	MIMEApplicationZip = "application/zip"
)

func mimeToArchiveType(mimeType string) archive.ArchiveType {
	switch mimeType {
	case MIMEApplicationGZip:
		return archive.ArchiveTgz
	case MIMEApplicationZip:
		return archive.ArchiveZip
	default:
		return archive.ArchiveUnknown
	}
}

// Since Echo does not send an http status code until the first write to the ResponseWriter,
// we use delayWriter to buffer our writes, which effectively delays sending the status code
// until we are more confident the download will succeed. delayWriter wraps bufio.Writer
// and adds Close().
type delayWriter struct {
	next *bufio.Writer
}

func (w *delayWriter) Write(p []byte) (int, error) {
	return w.next.Write(p)
}

// Close flushes the buffer if it is nonempty.
func (w *delayWriter) Close() error {
	return w.next.Flush()
}

func newDelayWriter(w io.Writer, delayBytes int) *delayWriter {
	return &delayWriter{
		next: bufio.NewWriterSize(w, delayBytes),
	}
}

func (m *Master) getCheckpointStorageConfig(id uuid.UUID) (
	*expconf.CheckpointStorageConfig, error,
) {
	checkpoint, err := m.db.CheckpointByUUID(id)
	if err != nil || checkpoint == nil {
		return nil, err
	}

	bytes, err := json.Marshal(checkpoint.CheckpointTrainingMetadata.ExperimentConfig)
	if err != nil {
		return nil, err
	}

	legacyConfig, err := expconf.ParseLegacyConfigJSON(bytes)
	if err != nil {
		return nil, err
	}

	return ptrs.Ptr(legacyConfig.CheckpointStorage), nil
}

func (m *Master) getCheckpointImpl(
	ctx context.Context, id uuid.UUID, mimeType string, content io.Writer,
) error {
	// Assume a checkpoint always has experiment configs
	storageConfig, err := m.getCheckpointStorageConfig(id)
	switch {
	case err != nil:
		return echo.NewHTTPError(http.StatusInternalServerError,
			fmt.Sprintf("unable to retrieve experiment config for checkpoint %s: %s",
				id.String(), err.Error()))
	case storageConfig == nil:
		return api.NotFoundErrs("checkpoint", id.String(), false)
	}

	// DelayWriter delays the first write until we have successfully downloaded
	// some bytes and are more confident that the download will succeed.
	dw := newDelayWriter(content, 16*1024)
	downloader, err := checkpoints.NewDownloader(
		dw, id.String(), storageConfig, mimeToArchiveType(mimeType))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	err = downloader.Download(ctx)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			fmt.Sprintf("unable to download checkpoint %s: %s", id.String(), err.Error()))
	}

	// Closing the writers will cause Echo to send a 200 response to the client. Hence we
	// cannot use defer, and we close the writers only when there has been no error.
	for _, v := range []io.Closer{downloader, dw} {
		if err := v.Close(); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError,
				fmt.Sprintf("failed to complete checkpoint download: %s", err.Error()))
		}
	}

	return nil
}

//	@Summary	Get a checkpoint's contents in a tgz or zip file.
//	@Tags		Checkpoints
//	@ID			get-checkpoint
//	@Accept		json
//	@Produce	application/gzip,application/zip
//	@Param		checkpoint_uuid	path	string	true	"Checkpoint UUID"
//	@Success	200				{}		string	""
//	@Router		/checkpoints/{checkpoint_uuid} [get]
//
// Read why this line exists on the comment on getAggregatedResourceAllocation in core.go.
func (m *Master) getCheckpoint(c echo.Context) error {
	// Get the MIME type. Only a single type is accepted.
	mimeType := c.Request().Header.Get("Accept")
	if mimeType != MIMEApplicationGZip &&
		mimeType != MIMEApplicationZip {
		return echo.NewHTTPError(http.StatusUnsupportedMediaType,
			fmt.Sprintf("unsupported media type to download a checkpoint: '%s'", mimeType))
	}

	args := struct {
		CheckpointUUID string `path:"checkpoint_uuid"`
	}{}
	if err := api.BindArgs(&args, c); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid checkpoint_uuid: "+err.Error())
	}
	id, err := uuid.Parse(args.CheckpointUUID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest,
			fmt.Sprintf("unable to parse checkpoint UUID %s: %s",
				args.CheckpointUUID, err))
	}

	curUser := c.(*detContext.DetContext).MustGetUser()
	errE := m.canDoActionOnCheckpoint(c.Request().Context(), curUser, args.CheckpointUUID,
		expauth.AuthZProvider.Get().CanGetExperimentArtifacts)
	if errE != nil {
		errM := m.canDoActionOnCheckpointThroughModel(c.Request().Context(), curUser, args.CheckpointUUID)
		if errM != nil {
			s, ok := status.FromError(errE)
			if !ok {
				return errE
			}
			switch s.Code() {
			case codes.NotFound:
				return echo.NewHTTPError(http.StatusNotFound, s.Message())
			case codes.PermissionDenied:
				return echo.NewHTTPError(http.StatusForbidden, s.Message())
			default:
				return fmt.Errorf(s.Message())
			}
		}
	}
	c.Response().Header().Set(echo.HeaderContentType, mimeType)
	return m.getCheckpointImpl(c.Request().Context(), id, mimeType, c.Response())
}
