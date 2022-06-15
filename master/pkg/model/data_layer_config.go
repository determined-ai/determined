package model

import (
	"encoding/json"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/check"
	"github.com/determined-ai/determined/master/pkg/union"
)

// DataLayerConfig configures data layer storage.
type DataLayerConfig struct {
	SharedFSConfig *SharedFSDataLayerConfig `union:"type,shared_fs" json:"-"`
	S3Config       *S3DataLayerConfig       `union:"type,s3"        json:"-"`
	GCSConfig      *GCSDataLayerConfig      `union:"type,gcs"       json:"-"`
}

// Validate implements the check.Validatable interface.
func (DataLayerConfig) Validate() []error { return nil }

// MarshalJSON implements the json.Marshaler interface.
func (d DataLayerConfig) MarshalJSON() ([]byte, error) {
	return union.Marshal(d)
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (d *DataLayerConfig) UnmarshalJSON(data []byte) error {
	if err := union.Unmarshal(data, d); err != nil {
		return err
	}
	type DefaultParser *DataLayerConfig
	return errors.Wrap(json.Unmarshal(data, DefaultParser(d)), "failed to parse data layer config")
}

// SharedFSDataLayerConfig configures data layer storage on a local file system.
type SharedFSDataLayerConfig struct {
	ContainerStoragePath *string `json:"container_storage_path"`
	HostStoragePath      *string `json:"host_storage_path,omitempty"`
}

// Validate implements the check.Validatable interface.
func (s SharedFSDataLayerConfig) Validate() []error {
	if s.HostStoragePath != nil {
		return []error{
			check.True(filepath.IsAbs(*s.HostStoragePath),
				"host_storage_path must be an absolute path"),
			check.True(filepath.IsAbs(*s.ContainerStoragePath),
				"container_storage_path must be an absolute path"),
		}
	}
	return nil
}

// S3DataLayerConfig configures data layer storage on S3.
type S3DataLayerConfig struct {
	Bucket                  string  `json:"bucket"`
	BucketDirectoryPath     string  `json:"bucket_directory_path"`
	LocalCacheContainerPath *string `json:"local_cache_container_path,omitempty"`
	LocalCacheHostPath      *string `json:"local_cache_host_path,omitempty"`
	AccessKey               *string `json:"access_key,omitempty"`
	SecretKey               *string `json:"secret_key,omitempty"`
	EndpointURL             *string `json:"endpoint_url,omitempty"`
}

// Validate implements the check.Validatable interface.
func (s S3DataLayerConfig) Validate() []error {
	validationErrors := make([]error, 0)
	if s.LocalCacheContainerPath != nil {
		validationErrors = append(
			validationErrors,
			check.True(filepath.IsAbs(*s.LocalCacheContainerPath),
				"local_cache_container_path must be an absolute path"),
		)
	}
	if s.LocalCacheHostPath != nil {
		validationErrors = append(validationErrors, []error{
			check.True(s.LocalCacheContainerPath != nil,
				"local_cache_container_path must be specified if local_cache_host_path is set"),
			check.True(filepath.IsAbs(*s.LocalCacheHostPath),
				"local_cache_host_path must be an absolute path"),
		}...,
		)
	}
	return validationErrors
}

// GCSDataLayerConfig configures data layer storage on GCS.
type GCSDataLayerConfig struct {
	Bucket                  string  `json:"bucket"`
	BucketDirectoryPath     string  `json:"bucket_directory_path"`
	LocalCacheContainerPath *string `json:"local_cache_container_path,omitempty"`
	LocalCacheHostPath      *string `json:"local_cache_host_path,omitempty"`
}

// Validate implements the check.Validatable interface.
func (g GCSDataLayerConfig) Validate() []error {
	validationErrors := make([]error, 0)
	if g.LocalCacheContainerPath != nil {
		validationErrors = append(
			validationErrors,
			check.True(filepath.IsAbs(*g.LocalCacheContainerPath),
				"local_cache_container_path must be an absolute path"),
		)
	}
	if g.LocalCacheHostPath != nil {
		validationErrors = append(validationErrors, []error{
			check.True(g.LocalCacheContainerPath != nil,
				"local_cache_container_path must be specified if local_cache_host_path is set"),
			check.True(filepath.IsAbs(*g.LocalCacheHostPath),
				"local_cache_host_path must be an absolute path"),
		}...,
		)
	}
	return validationErrors
}
