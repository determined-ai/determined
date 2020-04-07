package model

import (
	"encoding/json"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/check"
	"github.com/determined-ai/determined/master/pkg/union"
)

const (
	sharedFSContainerPath = "/determined_shared_fs"
	sharedFSPropagation   = "rprivate"
)

// CheckpointStorageConfig has the common checkpoint config params.
type CheckpointStorageConfig struct {
	SaveExperimentBest int `json:"save_experiment_best"`
	SaveTrialBest      int `json:"save_trial_best"`
	SaveTrialLatest    int `json:"save_trial_latest"`

	SharedFSConfig *SharedFSConfig `union:"type,shared_fs" json:"-"`
	HDFSConfig     *HDFSConfig     `union:"type,hdfs" json:"-"`
	S3Config       *S3Config       `union:"type,s3" json:"-"`
	GCSConfig      *GCSConfig      `union:"type,gcs" json:"-"`
}

// Validate implements the check.Validatable interface.
func (c CheckpointStorageConfig) Validate() []error {
	return []error{
		check.GreaterThanOrEqualTo(c.SaveExperimentBest, 0, "save_experiment_best must be >= 0"),
		check.GreaterThanOrEqualTo(c.SaveTrialBest, 0, "save_trial_best must be >= 0"),
		check.GreaterThanOrEqualTo(c.SaveTrialLatest, 0, "save_trial_latest must be >= 0"),
	}
}

// MarshalJSON implements the json.Marshaler interface.
func (c CheckpointStorageConfig) MarshalJSON() ([]byte, error) {
	return union.Marshal(c)
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (c *CheckpointStorageConfig) UnmarshalJSON(data []byte) error {
	if err := union.Unmarshal(data, c); err != nil {
		return err
	}
	type DefaultParser *CheckpointStorageConfig
	return errors.Wrap(json.Unmarshal(data, DefaultParser(c)), "failed to parse checkpoint storage")
}

// TensorboardStorageConfig has the common checkpoint config params.
type TensorboardStorageConfig struct {
	SharedFSConfig *SharedFSConfig `union:"type,shared_fs" json:"-"`
	HDFSConfig     *HDFSConfig     `union:"type,hdfs" json:"-"`
	S3Config       *S3Config       `union:"type,s3" json:"-"`
	GCSConfig      *GCSConfig      `union:"type,gcs" json:"-"`
}

// MarshalJSON implements the json.Marshaler interface.
func (t TensorboardStorageConfig) MarshalJSON() ([]byte, error) {
	return union.Marshal(t)
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (t *TensorboardStorageConfig) UnmarshalJSON(data []byte) error {
	return union.Unmarshal(data, t)
}

// A SharedFSContainerPath is a constant value for the ContainerPath of a
// SharedFSConfig.
type SharedFSContainerPath struct{}

// MarshalJSON implements the json.Marshaler interface.
func (p SharedFSContainerPath) MarshalJSON() ([]byte, error) {
	return json.Marshal(sharedFSContainerPath)
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (p *SharedFSContainerPath) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return nil
	}

	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	if s != sharedFSContainerPath {
		return errors.New("cannot set value")
	}
	return nil
}

func (p SharedFSContainerPath) String() string {
	return sharedFSContainerPath
}

// A SharedFSPropagation is a constant value for the Propagation of a
// SharedFSConfig.
type SharedFSPropagation struct{}

// MarshalJSON implements the json.Marshaler interface.
func (p SharedFSPropagation) MarshalJSON() ([]byte, error) {
	return json.Marshal(sharedFSPropagation)
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (p *SharedFSPropagation) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return nil
	}

	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	if s != sharedFSPropagation {
		return errors.New("cannot set value")
	}
	return nil
}

func (p SharedFSPropagation) String() string {
	return sharedFSPropagation
}

// SharedFSConfig configures storing on a shared filesystem (e.g., NFS).
type SharedFSConfig struct {
	HostPath        string                `json:"host_path"`
	ContainerPath   SharedFSContainerPath `json:"container_path"`
	CheckpointPath  *string               `json:"checkpoint_path,omitempty"`
	TensorboardPath *string               `json:"tensorboard_path,omitempty"`
	StoragePath     *string               `json:"storage_path,omitempty"`
	Propagation     SharedFSPropagation   `json:"propagation"`
}

// Validate implements the check.Validatable interface.
func (s SharedFSConfig) Validate() []error {
	return []error{
		check.True(filepath.IsAbs(s.HostPath), "host_path must be an absolute path"),
	}
}

// HDFSConfig configures storing checkpoints in HDFS.
type HDFSConfig struct {
	URL  string  `json:"hdfs_url"`
	Path string  `json:"hdfs_path"`
	User *string `json:"user,omitempty"`
}

// Validate implements the check.Validatable interface.
func (h HDFSConfig) Validate() []error {
	return []error{
		check.True(filepath.IsAbs(h.Path), "hdfs_path must be an absolute path"),
	}
}

// S3Config configures storing checkpoints on S3.
type S3Config struct {
	Bucket      string  `json:"bucket"`
	AccessKey   *string `json:"access_key,omitempty"`
	SecretKey   *string `json:"secret_key,omitempty"`
	EndpointURL *string `json:"endpoint_url,omitempty"`
}

// Validate implements the check.Validatable interface.
func (S3Config) Validate() []error { return nil }

// GCSConfig configures storing checkpoints on GCS.
type GCSConfig struct {
	Bucket string `json:"bucket"`
}

// Validate implements the check.Validatable interface.
func (GCSConfig) Validate() []error { return nil }
