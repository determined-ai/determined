package expconf

import (
	"encoding/json"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/schemas"
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
	SaveExperimentBest *int `json:"save_experiment_best"`
	SaveTrialBest      *int `json:"save_trial_best"`
	SaveTrialLatest    *int `json:"save_trial_latest"`

	SharedFSConfig *SharedFSConfigV0 `union:"type,shared_fs" json:"-"`
	HDFSConfig     *HDFSConfigV0     `union:"type,hdfs" json:"-"`
	S3Config       *S3ConfigV0       `union:"type,s3" json:"-"`
	GCSConfig      *GCSConfigV0      `union:"type,gcs" json:"-"`
}

// MarshalJSON implements the json.Marshaler interface.
func (c CheckpointStorageConfigV0) MarshalJSON() ([]byte, error) {
	return union.MarshalEx(c, true)
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (c *CheckpointStorageConfigV0) UnmarshalJSON(data []byte) error {
	if err := union.Unmarshal(data, c); err != nil {
		return err
	}
	type DefaultParser *CheckpointStorageConfigV0
	return errors.Wrap(json.Unmarshal(data, DefaultParser(c)), "failed to parse checkpoint storage")
}

// DefaultSource implements the Defaultable interface.
func (c CheckpointStorageConfigV0) DefaultSource() interface{} {
	return schemas.UnionDefaultSchema(c)
}

// Printable modifies the object with secrets hidden.
func (c *CheckpointStorageConfig) Printable() {
	hiddenValue := "********"
	if c.S3Config != nil {
		if c.S3Config.AccessKey != nil {
			c.S3Config.AccessKey = &hiddenValue
		}
		if c.S3Config.SecretKey != nil {
			c.S3Config.SecretKey = &hiddenValue
		}
	}
}

// TensorboardStorageConfigV0 has the common checkpoint config params.
type TensorboardStorageConfigV0 struct {
	SharedFSConfigV0 *SharedFSConfigV0 `union:"type,shared_fs" json:"-"`
	HDFSConfig       *HDFSConfigV0     `union:"type,hdfs" json:"-"`
	S3Config         *S3ConfigV0       `union:"type,s3" json:"-"`
	GCSConfig        *GCSConfigV0      `union:"type,gcs" json:"-"`
}

// MarshalJSON implements the json.Marshaler interface.
func (t TensorboardStorageConfigV0) MarshalJSON() ([]byte, error) {
	return union.Marshal(t)
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (t *TensorboardStorageConfigV0) UnmarshalJSON(data []byte) error {
	return union.Unmarshal(data, t)
}

// DefaultSource implements the Defaultable interface.
func (t TensorboardStorageConfigV0) DefaultSource() interface{} {
	return schemas.UnionDefaultSchema(t)
}

// SharedFSConfigV0 is a legacy config.
type SharedFSConfigV0 struct {
	HostPath        string  `json:"host_path"`
	ContainerPath   *string `json:"container_path,omitempty"`
	CheckpointPath  *string `json:"checkpoint_path,omitempty"`
	TensorboardPath *string `json:"tensorboard_path,omitempty"`
	StoragePath     *string `json:"storage_path"`
	Propagation     *string `json:"propagation"`
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

// HDFSConfigV0 configures storing checkpoints in HDFS.
type HDFSConfigV0 struct {
	URL  string  `json:"hdfs_url"`
	Path string  `json:"hdfs_path"`
	User *string `json:"user"`
}

// S3ConfigV0 configures storing checkpoints on S3.
type S3ConfigV0 struct {
	Bucket      string  `json:"bucket"`
	AccessKey   *string `json:"access_key"`
	SecretKey   *string `json:"secret_key"`
	EndpointURL *string `json:"endpoint_url"`
}

// GCSConfigV0 configures storing checkpoints on GCS.
type GCSConfigV0 struct {
	Bucket string `json:"bucket"`
}
