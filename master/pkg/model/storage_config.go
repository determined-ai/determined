package model

import (
	"encoding/json"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/union"
)

const (
	// DefaultSharedFSContainerPath is the base storage path inside containers for SharedFS storage.
	DefaultSharedFSContainerPath = "/determined_shared_fs"
	// DefaultSharedFSPropagation is the propagation setting for SharedFS storage.
	DefaultSharedFSPropagation = "rprivate"
)

// CheckpointStorageConfigV0 has the common checkpoint config params.
type CheckpointStorageConfigV0 struct {
	SaveExperimentBest int `json:"save_experiment_best"`
	SaveTrialBest      int `json:"save_trial_best"`
	SaveTrialLatest    int `json:"save_trial_latest"`

	SharedFSConfig *SharedFSConfigV0 `union:"type,shared_fs" json:"-"`
	HDFSConfig     *HDFSConfig       `union:"type,hdfs" json:"-"`
	S3Config       *S3Config         `union:"type,s3" json:"-"`
	GCSConfig      *GCSConfig        `union:"type,gcs" json:"-"`
}

// CheckpointStorageConfigV1 has the common checkpoint config params.
type CheckpointStorageConfigV1 struct {
	SaveExperimentBest int `json:"save_experiment_best"`
	SaveTrialBest      int `json:"save_trial_best"`
	SaveTrialLatest    int `json:"save_trial_latest"`

	SharedFSConfig *SharedFSConfig `union:"type,shared_fs" json:"-"`
	HDFSConfig     *HDFSConfig     `union:"type,hdfs" json:"-"`
	S3Config       *S3Config       `union:"type,s3" json:"-"`
	GCSConfig      *GCSConfig      `union:"type,gcs" json:"-"`
}

// MarshalJSON implements the json.Marshaler interface.
func (c CheckpointStorageConfigV1) MarshalJSON() ([]byte, error) {
	return union.MarshalEx(c, true)
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (c *CheckpointStorageConfigV1) UnmarshalJSON(data []byte) error {
	if err := union.Unmarshal(data, c); err != nil {
		return err
	}
	type DefaultParser *CheckpointStorageConfigV1
	return errors.Wrap(json.Unmarshal(data, DefaultParser(c)), "failed to parse checkpoint storage")
}

// TensorboardStorageConfigV0 has the common checkpoint config params.
type TensorboardStorageConfigV0 struct {
	SharedFSConfigV0 *SharedFSConfigV0 `union:"type,shared_fs" json:"-"`
	HDFSConfig       *HDFSConfig       `union:"type,hdfs" json:"-"`
	S3Config         *S3Config         `union:"type,s3" json:"-"`
	GCSConfig        *GCSConfig        `union:"type,gcs" json:"-"`
}

// MarshalJSON implements the json.Marshaler interface.
func (t TensorboardStorageConfigV0) MarshalJSON() ([]byte, error) {
	return union.Marshal(t)
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (t *TensorboardStorageConfigV0) UnmarshalJSON(data []byte) error {
	return union.Unmarshal(data, t)
}

// SharedFSConfigV0 is a legacy config.
type SharedFSConfigV0 struct {
	HostPath        string  `json:"host_path"`
	ContainerPath   *string `json:"container_path,omitempty"`
	CheckpointPath  *string `json:"checkpoint_path,omitempty"`
	TensorboardPath *string `json:"tensorboard_path,omitempty"`
	StoragePath     *string `json:"storage_path,omitempty"`
	Propagation     *string `json:"propagation,omitempty"`
}

// SharedFSConfig configures storing on a shared filesystem (e.g., NFS).
type SharedFSConfig struct {
	HostPath      string  `json:"host_path"`
	ContainerPath *string `json:"container_path,omitempty"`
	StoragePath   *string `json:"storage_path,omitempty"`
	Propagation   *string `json:"propagation,omitempty"`
}

// PathInContainer caclulates where the full StoragePath will be inside the container.
func (s SharedFSConfig) PathInContainer() string {
	if s.StoragePath == nil {
		return DefaultSharedFSContainerPath
	}
	if filepath.IsAbs(*s.StoragePath) {
		relPath, err := filepath.Rel(s.HostPath, *s.StoragePath)
		if err != nil {
			panic("detected unvalidated sharedfs config")
		}
		return filepath.Join(DefaultSharedFSContainerPath, relPath)
	}
	return filepath.Join(DefaultSharedFSContainerPath, *s.StoragePath)
}

// HDFSConfig configures storing checkpoints in HDFS.
type HDFSConfig struct {
	URL  string  `json:"hdfs_url"`
	Path string  `json:"hdfs_path"`
	User *string `json:"user,omitempty"`
}

// S3Config configures storing checkpoints on S3.
type S3Config struct {
	Bucket      string  `json:"bucket"`
	AccessKey   *string `json:"access_key,omitempty"`
	SecretKey   *string `json:"secret_key,omitempty"`
	EndpointURL *string `json:"endpoint_url,omitempty"`
}

// GCSConfig configures storing checkpoints on GCS.
type GCSConfig struct {
	Bucket string `json:"bucket"`
}
