package checkpoints

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/determined-ai/determined/master/pkg/checkpoints/archive"
	"github.com/determined-ai/determined/master/pkg/checkpoints/gcs"
	"github.com/determined-ai/determined/master/pkg/checkpoints/s3"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
)

// CheckpointDownloader defines the interface for downloading checkpoints.
type CheckpointDownloader interface {
	Download(ctx context.Context) error
	Close() error
}

// NewDownloader returns a new CheckpointDownloader that writes to w.
//
//   - w: the underlying Writer that CheckpointDownloader writes to
//   - id: the UUID string of the checkpoint to be downloaded
//   - storageConfig: the CheckpointStorageConfig
//   - archiveType: The ArchiveType (file format) in which the checkpoint shall
//     be downloaded
func NewDownloader(
	w io.Writer,
	id string,
	storageConfig *expconf.CheckpointStorageConfig,
	archiveType archive.ArchiveType,
) (CheckpointDownloader, error) {
	aw, err := archive.NewArchiveWriter(w, archiveType)
	if err != nil {
		return nil, err
	}

	prefix := ""
	switch storage := storageConfig.GetUnionMember().(type) {
	case expconf.S3Config:
		if storage.Prefix() != nil {
			prefix = *storage.Prefix()
		}
		return s3.NewS3Downloader(
			aw, storage.Bucket(), strings.TrimLeft(prefix+"/"+id, "/")), nil
	case expconf.GCSConfig:
		if storage.Prefix() != nil {
			prefix = *storage.Prefix()
		}
		return gcs.NewGCSDownloader(
			aw, storage.Bucket(), strings.TrimLeft(prefix+"/"+id, "/")), nil
	default:
		return nil,
			fmt.Errorf("checkpoint download via master is not supported for %s",
				storageConfig2Str(storage))
	}
}

func storageConfig2Str(config any) string {
	switch config.(type) {
	case expconf.AzureConfig:
		return "azure"
	case expconf.GCSConfig:
		return "gcs"
	case expconf.S3Config:
		return "s3"
	case expconf.SharedFSConfig:
		return "shared_fs"
	default:
		return "unknown"
	}
}
