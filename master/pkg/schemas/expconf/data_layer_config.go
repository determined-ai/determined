package expconf

import (
	"encoding/json"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/schemas"
	"github.com/determined-ai/determined/master/pkg/union"
)

//go:generate ../gen.sh
// DataLayerConfigV0 configures data layer storage.
type DataLayerConfigV0 struct {
	RawSharedFSConfig *SharedFSDataLayerConfigV0 `union:"type,shared_fs" json:"-"`
	RawS3Config       *S3DataLayerConfigV0       `union:"type,s3" json:"-"`
	RawGCSConfig      *GCSDataLayerConfigV0      `union:"type,gcs" json:"-"`
}

// Merge implements schemas.Mergeable.  This Merge enforces that we can't ever merge two union
// members into one output.
func (d DataLayerConfigV0) Merge(other interface{}) interface{} {
	tOther := other.(DataLayerConfigV0)

	// There are no common members to merge.
	out := DataLayerConfigV0{}

	// Only merge union members based on d, not based on other... unless d has no member at all.
	// The only reason it is valid to have no members is due to common fields on union types.
	useOther := all(
		d.RawSharedFSConfig == nil,
		d.RawS3Config == nil,
		d.RawGCSConfig == nil,
	)
	if useOther || d.RawSharedFSConfig != nil {
		out.RawSharedFSConfig = schemas.Merge(
			d.RawSharedFSConfig, tOther.RawSharedFSConfig,
		).(*SharedFSDataLayerConfigV0)
	}
	if useOther || d.RawS3Config != nil {
		out.RawS3Config = schemas.Merge(d.RawS3Config, tOther.RawS3Config).(*S3DataLayerConfigV0)
	}
	if useOther || d.RawGCSConfig != nil {
		out.RawGCSConfig = schemas.Merge(d.RawGCSConfig, tOther.RawGCSConfig).(*GCSDataLayerConfigV0)
	}
	return out
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
	RawContainerStoragePath *string `json:"container_storage_path"`
	RawHostStoragePath      *string `json:"host_storage_path"`
}

//go:generate ../gen.sh
// S3DataLayerConfigV0 configures data layer storage on S3.
type S3DataLayerConfigV0 struct {
	RawBucket                  *string `json:"bucket"`
	RawBucketDirectoryPath     *string `json:"bucket_directory_path"`
	RawLocalCacheContainerPath *string `json:"local_cache_container_path"`
	RawLocalCacheHostPath      *string `json:"local_cache_host_path"`
	RawAccessKey               *string `json:"access_key"`
	RawSecretKey               *string `json:"secret_key"`
	RawEndpointURL             *string `json:"endpoint_url"`
}

//go:generate ../gen.sh
// GCSDataLayerConfigV0 configures data layer storage on GCS.
type GCSDataLayerConfigV0 struct {
	RawBucket                  *string `json:"bucket"`
	RawBucketDirectoryPath     *string `json:"bucket_directory_path"`
	RawLocalCacheContainerPath *string `json:"local_cache_container_path"`
	RawLocalCacheHostPath      *string `json:"local_cache_host_path"`
}
