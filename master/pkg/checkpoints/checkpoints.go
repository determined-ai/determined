package checkpoints

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/determined-ai/determined/master/pkg/checkpoints/archive"
	"github.com/determined-ai/determined/master/pkg/checkpoints/gcs"
	"github.com/determined-ai/determined/master/pkg/checkpoints/local"
	"github.com/determined-ai/determined/master/pkg/checkpoints/s3"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
)

// CheckpointDownloader defines the interface for downloading checkpoints.
type CheckpointDownloader interface {
	Download(context.Context) error
	Close() error
	ListFiles(context.Context) ([]archive.FileEntry, error)
}

// NewDownloader returns a new CheckpointDownloader that writes to w.
//
//   - w: the underlying Writer that CheckpointDownloader writes to
//   - id: the UUID string of the checkpoint to be downloaded
//   - storageConfig: the CheckpointStorageConfig
//   - archiveType: The ArchiveType (file format) in which the checkpoint shall
//     be downloaded
func NewDownloader(
	ctx context.Context,
	w io.Writer,
	id string,
	storageConfig *expconf.CheckpointStorageConfig,
	aw archive.ArchiveWriter,
) (CheckpointDownloader, error) {
	idPrefix := func(prefix string) string {
		prefix = strings.TrimRight(prefix, "/")
		return prefix + "/" + id
	}
	idPrefixRef := func(prefixRef *string) string {
		prefix := ""
		if prefixRef != nil {
			prefix = *prefixRef
		}
		return idPrefix(prefix)
	}

	switch storage := storageConfig.GetUnionMember().(type) {
	case expconf.S3Config:
		prefix := idPrefixRef(storage.Prefix())
		return s3.NewS3Downloader(ctx, aw, storage.Bucket(), prefix, storage.EndpointURL())

	case expconf.GCSConfig:
		prefix := idPrefixRef(storage.Prefix())
		return gcs.NewGCSDownloader(ctx, aw, storage.Bucket(), prefix)

	case expconf.SharedFSConfig:
		pathPrefix, err := storage.PathInContainerOrHost()
		if err != nil {
			return nil, err
		}
		prefix := idPrefix(pathPrefix)
		return local.NewLocalDownloader(aw, prefix)

	case expconf.DirectoryConfig:
		prefix := idPrefix(storage.ContainerPath())
		return local.NewLocalDownloader(aw, prefix)

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
	case expconf.DirectoryConfig:
		return "directory"
	default:
		return "unknown"
	}
}
