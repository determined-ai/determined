package expconf

import (
	"encoding/json"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/union"
)

//go:generate ../gen.sh
// DataLayerConfigV0 configures data layer storage.
type DataLayerConfigV0 struct {
	SharedFSConfig *SharedFSDataLayerConfigV0 `union:"type,shared_fs" json:"-"`
	S3Config       *S3DataLayerConfigV0       `union:"type,s3" json:"-"`
	GCSConfig      *GCSDataLayerConfigV0      `union:"type,gcs" json:"-"`
}

// MarshalJSON implements the json.Marshaler interface.
func (d DataLayerConfigV0) MarshalJSON() ([]byte, error) {
	return union.MarshalEx(d, true)
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (d *DataLayerConfigV0) UnmarshalJSON(data []byte) error {
	if err := union.Unmarshal(data, d); err != nil {
		return err
	}
	type DefaultParser *DataLayerConfigV0
	return errors.Wrap(json.Unmarshal(data, DefaultParser(d)), "failed to parse data layer config")
}

//go:generate ../gen.sh
// SharedFSDataLayerConfigV0 configures data layer storage on a local file system.
type SharedFSDataLayerConfigV0 struct {
	ContainerStoragePath *string `json:"container_storage_path"`
	HostStoragePath      *string `json:"host_storage_path"`
}

//go:generate ../gen.sh
// S3DataLayerConfigV0 configures data layer storage on S3.
type S3DataLayerConfigV0 struct {
	Bucket                  string  `json:"bucket"`
	BucketDirectoryPath     string  `json:"bucket_directory_path"`
	LocalCacheContainerPath *string `json:"local_cache_container_path"`
	LocalCacheHostPath      *string `json:"local_cache_host_path"`
	AccessKey               *string `json:"access_key"`
	SecretKey               *string `json:"secret_key"`
	EndpointURL             *string `json:"endpoint_url"`
}

//go:generate ../gen.sh
// GCSDataLayerConfigV0 configures data layer storage on GCS.
type GCSDataLayerConfigV0 struct {
	Bucket                  string  `json:"bucket"`
	BucketDirectoryPath     string  `json:"bucket_directory_path"`
	LocalCacheContainerPath *string `json:"local_cache_container_path"`
	LocalCacheHostPath      *string `json:"local_cache_host_path"`
}
