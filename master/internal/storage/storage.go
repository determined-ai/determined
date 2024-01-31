package storage

import (
	"fmt"

	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
)

type storageBackendRow struct {
	bun.BaseModel `bun:"table:storage_backend"`
	ID            model.StorageBackendID `bun:",pk,autoincrement"`
	SharedFSID    *int                   `bun:"shared_fs_id"`
	S3ID          *int                   `bun:"s3_id"`
	GCSID         *int                   `bun:"gcs_id"`
	AzureID       *int                   `bun:"azure_id"`
	DirectoryID   *int                   `bun:"directory_id"`
}

func (p *storageBackendRow) toChildRowOnlyIDPopulated() storageBackend {
	switch {
	case p.SharedFSID != nil:
		return &storageBackendSharedFS{ID: *p.SharedFSID}
	case p.S3ID != nil:
		return &storageBackendS3{ID: *p.S3ID}
	case p.GCSID != nil:
		return &storageBackendGCS{ID: *p.GCSID}
	case p.AzureID != nil:
		return &storageBackendAzure{ID: *p.AzureID}
	case p.DirectoryID != nil:
		return &storageBackendDirectory{ID: *p.DirectoryID}
	default:
		panic(fmt.Sprintf("expected one of p to be nil %+v", p))
	}
}

type storageBackend interface {
	toExpconf() *expconf.CheckpointStorageConfig
	id() int
}

type storageBackendSharedFS struct {
	bun.BaseModel `bun:"table:storage_backend_shared_fs"`
	ID            int `bun:",pk,autoincrement"`

	HostPath        string  `bun:"host_path"`
	ContainerPath   *string `bun:"container_path"`
	CheckpointPath  *string `bun:"checkpoint_path"`
	TensorboardPath *string `bun:"tensorboard_path"`
	StoragePath     *string `bun:"storage_path"`
	Propagation     string  `bun:"propagation"`
}

func (s *storageBackendSharedFS) id() int {
	return s.ID
}

//nolint:exhaustruct
func (s *storageBackendSharedFS) toExpconf() *expconf.CheckpointStorageConfig {
	return &expconf.CheckpointStorageConfig{
		RawSharedFSConfig: &expconf.SharedFSConfig{
			RawHostPath:        &s.HostPath,
			RawContainerPath:   s.ContainerPath,
			RawCheckpointPath:  s.CheckpointPath,
			RawTensorboardPath: s.TensorboardPath,
			RawStoragePath:     s.StoragePath,
			RawPropagation:     &s.Propagation,
		},
	}
}

type storageBackendS3 struct {
	bun.BaseModel `bun:"table:storage_backend_s3"`
	ID            int `bun:",pk,autoincrement"`

	Bucket      string  `bun:"bucket"`
	AccessKey   *string `bun:"access_key"`
	SecretKey   *string `bun:"secret_key"`
	EndpointURL *string `bun:"endpoint_url"`
	Prefix      *string `bun:"prefix"`
}

func (s *storageBackendS3) id() int {
	return s.ID
}

//nolint:exhaustruct
func (s *storageBackendS3) toExpconf() *expconf.CheckpointStorageConfig {
	return &expconf.CheckpointStorageConfig{
		RawS3Config: &expconf.S3Config{
			RawBucket:      &s.Bucket,
			RawAccessKey:   s.AccessKey,
			RawSecretKey:   s.SecretKey,
			RawEndpointURL: s.EndpointURL,
			RawPrefix:      s.Prefix,
		},
	}
}

type storageBackendGCS struct {
	bun.BaseModel `bun:"table:storage_backend_gcs"`
	ID            int `bun:",pk,autoincrement"`

	Bucket string  `bun:"bucket"`
	Prefix *string `bun:"prefix"`
}

func (s *storageBackendGCS) id() int {
	return s.ID
}

//nolint:exhaustruct
func (s *storageBackendGCS) toExpconf() *expconf.CheckpointStorageConfig {
	return &expconf.CheckpointStorageConfig{
		RawGCSConfig: &expconf.GCSConfig{
			RawBucket: &s.Bucket,
			RawPrefix: s.Prefix,
		},
	}
}

type storageBackendAzure struct {
	bun.BaseModel `bun:"table:storage_backend_azure"`
	ID            int `bun:",pk,autoincrement"`

	Container        string  `bun:"container"`
	ConnectionString *string `bun:"connection_string"`
	AccountURL       *string `bun:"account_url"`
	Credential       *string `bun:"credential"`
}

func (s *storageBackendAzure) id() int {
	return s.ID
}

//nolint:exhaustruct
func (s *storageBackendAzure) toExpconf() *expconf.CheckpointStorageConfig {
	return &expconf.CheckpointStorageConfig{
		RawAzureConfig: &expconf.AzureConfig{
			RawContainer:        &s.Container,
			RawConnectionString: s.ConnectionString,
			RawAccountURL:       s.AccountURL,
			RawCredential:       s.Credential,
		},
	}
}

type storageBackendDirectory struct {
	bun.BaseModel `bun:"table:storage_backend_directory"`
	ID            int `bun:",pk,autoincrement"`

	ContainerPath string `bun:"container_path"`
}

func (s *storageBackendDirectory) id() int {
	return s.ID
}

//nolint:exhaustruct
func (s *storageBackendDirectory) toExpconf() *expconf.CheckpointStorageConfig {
	return &expconf.CheckpointStorageConfig{
		RawDirectoryConfig: &expconf.DirectoryConfig{
			RawContainerPath: &s.ContainerPath,
		},
	}
}

func expconfToStorage(cs *expconf.CheckpointStorageConfig) (storageBackend, string) {
	switch storage := cs.GetUnionMember().(type) {
	case expconf.SharedFSConfig:
		return &storageBackendSharedFS{
			HostPath:        storage.HostPath(),
			ContainerPath:   storage.ContainerPath(),
			CheckpointPath:  storage.CheckpointPath(),
			TensorboardPath: storage.TensorboardPath(),
			StoragePath:     storage.StoragePath(),
			Propagation:     storage.Propagation(),
		}, "shared_fs_id"
	case expconf.S3Config:
		return &storageBackendS3{
			Bucket:      storage.Bucket(),
			AccessKey:   storage.AccessKey(),
			SecretKey:   storage.SecretKey(),
			EndpointURL: storage.EndpointURL(),
			Prefix:      storage.Prefix(),
		}, "s3_id"
	case expconf.GCSConfig:
		return &storageBackendGCS{
			Bucket: storage.Bucket(),
			Prefix: storage.Prefix(),
		}, "gcs_id"
	case expconf.AzureConfig:
		return &storageBackendAzure{
			Container:        storage.Container(),
			ConnectionString: storage.ConnectionString(),
			AccountURL:       storage.AccountURL(),
			Credential:       storage.Credential(),
		}, "azure_id"
	case expconf.DirectoryConfig:
		return &storageBackendDirectory{
			ContainerPath: storage.ContainerPath(),
		}, "directory_id"
	default:
		panic(fmt.Sprintf("unknown type converting expconf to storage backend %T", storage))
	}
}
