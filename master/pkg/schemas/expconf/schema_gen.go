// This is a generated file.  Editing it will make you sad.

package expconf

import (
	"github.com/santhosh-tekuri/jsonschema/v2"

	"github.com/determined-ai/determined/master/pkg/schemas"
)

func (x *BindMountV0) ParsedSchema() interface{} {
	return schemas.ParsedBindMountV0()
}

func (x *BindMountV0) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v0/bind-mount.json")
}

func (x *BindMountV0) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v0/bind-mount.json")
}

func (x *CheckpointStorageConfigV0) ParsedSchema() interface{} {
	return schemas.ParsedCheckpointStorageConfigV0()
}

func (x *CheckpointStorageConfigV0) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v0/checkpoint-storage.json")
}

func (x *CheckpointStorageConfigV0) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v0/checkpoint-storage.json")
}

func (x *GCSDataLayerConfigV0) ParsedSchema() interface{} {
	return schemas.ParsedGCSDataLayerConfigV0()
}

func (x *GCSDataLayerConfigV0) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v0/data-layer-gcs.json")
}

func (x *GCSDataLayerConfigV0) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v0/data-layer-gcs.json")
}

func (x *S3DataLayerConfigV0) ParsedSchema() interface{} {
	return schemas.ParsedS3DataLayerConfigV0()
}

func (x *S3DataLayerConfigV0) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v0/data-layer-s3.json")
}

func (x *S3DataLayerConfigV0) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v0/data-layer-s3.json")
}

func (x *SharedFSDataLayerConfigV0) ParsedSchema() interface{} {
	return schemas.ParsedSharedFSDataLayerConfigV0()
}

func (x *SharedFSDataLayerConfigV0) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v0/data-layer-shared-fs.json")
}

func (x *SharedFSDataLayerConfigV0) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v0/data-layer-shared-fs.json")
}

func (x *DataLayerConfigV0) ParsedSchema() interface{} {
	return schemas.ParsedDataLayerConfigV0()
}

func (x *DataLayerConfigV0) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v0/data-layer.json")
}

func (x *DataLayerConfigV0) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v0/data-layer.json")
}

func (x *EnvironmentImageMapV0) ParsedSchema() interface{} {
	return schemas.ParsedEnvironmentImageMapV0()
}

func (x *EnvironmentImageMapV0) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v0/environment-image-map.json")
}

func (x *EnvironmentImageMapV0) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v0/environment-image-map.json")
}

func (x *EnvironmentImageV0) ParsedSchema() interface{} {
	return schemas.ParsedEnvironmentImageV0()
}

func (x *EnvironmentImageV0) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v0/environment-image.json")
}

func (x *EnvironmentImageV0) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v0/environment-image.json")
}

func (x *EnvironmentVariablesMapV0) ParsedSchema() interface{} {
	return schemas.ParsedEnvironmentVariablesMapV0()
}

func (x *EnvironmentVariablesMapV0) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v0/environment-variables-map.json")
}

func (x *EnvironmentVariablesMapV0) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v0/environment-variables-map.json")
}

func (x *EnvironmentVariablesV0) ParsedSchema() interface{} {
	return schemas.ParsedEnvironmentVariablesV0()
}

func (x *EnvironmentVariablesV0) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v0/environment-variables.json")
}

func (x *EnvironmentVariablesV0) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v0/environment-variables.json")
}

func (x *EnvironmentConfigV0) ParsedSchema() interface{} {
	return schemas.ParsedEnvironmentConfigV0()
}

func (x *EnvironmentConfigV0) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v0/environment.json")
}

func (x *EnvironmentConfigV0) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v0/environment.json")
}

func (x *ExperimentConfigV0) ParsedSchema() interface{} {
	return schemas.ParsedExperimentConfigV0()
}

func (x *ExperimentConfigV0) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v0/experiment.json")
}

func (x *ExperimentConfigV0) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v0/experiment.json")
}

func (x *GCSConfigV0) ParsedSchema() interface{} {
	return schemas.ParsedGCSConfigV0()
}

func (x *GCSConfigV0) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v0/gcs.json")
}

func (x *GCSConfigV0) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v0/gcs.json")
}

func (x *HDFSConfigV0) ParsedSchema() interface{} {
	return schemas.ParsedHDFSConfigV0()
}

func (x *HDFSConfigV0) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v0/hdfs.json")
}

func (x *HDFSConfigV0) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v0/hdfs.json")
}

func (x *CategoricalHyperparameterV0) ParsedSchema() interface{} {
	return schemas.ParsedCategoricalHyperparameterV0()
}

func (x *CategoricalHyperparameterV0) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v0/hyperparameter-categorical.json")
}

func (x *CategoricalHyperparameterV0) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v0/hyperparameter-categorical.json")
}

func (x *ConstHyperparameterV0) ParsedSchema() interface{} {
	return schemas.ParsedConstHyperparameterV0()
}

func (x *ConstHyperparameterV0) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v0/hyperparameter-const.json")
}

func (x *ConstHyperparameterV0) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v0/hyperparameter-const.json")
}

func (x *DoubleHyperparameterV0) ParsedSchema() interface{} {
	return schemas.ParsedDoubleHyperparameterV0()
}

func (x *DoubleHyperparameterV0) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v0/hyperparameter-double.json")
}

func (x *DoubleHyperparameterV0) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v0/hyperparameter-double.json")
}

func (x *IntHyperparameterV0) ParsedSchema() interface{} {
	return schemas.ParsedIntHyperparameterV0()
}

func (x *IntHyperparameterV0) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v0/hyperparameter-int.json")
}

func (x *IntHyperparameterV0) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v0/hyperparameter-int.json")
}

func (x *LogHyperparameterV0) ParsedSchema() interface{} {
	return schemas.ParsedLogHyperparameterV0()
}

func (x *LogHyperparameterV0) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v0/hyperparameter-log.json")
}

func (x *LogHyperparameterV0) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v0/hyperparameter-log.json")
}

func (x *HyperparameterV0) ParsedSchema() interface{} {
	return schemas.ParsedHyperparameterV0()
}

func (x *HyperparameterV0) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v0/hyperparameter.json")
}

func (x *HyperparameterV0) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v0/hyperparameter.json")
}

func (x *HyperparametersV0) ParsedSchema() interface{} {
	return schemas.ParsedHyperparametersV0()
}

func (x *HyperparametersV0) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v0/hyperparameters.json")
}

func (x *HyperparametersV0) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v0/hyperparameters.json")
}

func (x *InternalConfigV0) ParsedSchema() interface{} {
	return schemas.ParsedInternalConfigV0()
}

func (x *InternalConfigV0) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v0/internal.json")
}

func (x *InternalConfigV0) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v0/internal.json")
}

func (x *KerberosConfigV0) ParsedSchema() interface{} {
	return schemas.ParsedKerberosConfigV0()
}

func (x *KerberosConfigV0) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v0/kerberos.json")
}

func (x *KerberosConfigV0) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v0/kerberos.json")
}

func (x *LengthV0) ParsedSchema() interface{} {
	return schemas.ParsedLengthV0()
}

func (x *LengthV0) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v0/length.json")
}

func (x *LengthV0) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v0/length.json")
}

func (x *OptimizationsConfigV0) ParsedSchema() interface{} {
	return schemas.ParsedOptimizationsConfigV0()
}

func (x *OptimizationsConfigV0) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v0/optimizations.json")
}

func (x *OptimizationsConfigV0) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v0/optimizations.json")
}

func (x *ReproducibilityConfigV0) ParsedSchema() interface{} {
	return schemas.ParsedReproducibilityConfigV0()
}

func (x *ReproducibilityConfigV0) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v0/reproducibility.json")
}

func (x *ReproducibilityConfigV0) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v0/reproducibility.json")
}

func (x *ResourcesConfigV0) ParsedSchema() interface{} {
	return schemas.ParsedResourcesConfigV0()
}

func (x *ResourcesConfigV0) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v0/resources.json")
}

func (x *ResourcesConfigV0) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v0/resources.json")
}

func (x *S3ConfigV0) ParsedSchema() interface{} {
	return schemas.ParsedS3ConfigV0()
}

func (x *S3ConfigV0) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v0/s3.json")
}

func (x *S3ConfigV0) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v0/s3.json")
}

func (x *AdaptiveASHAConfigV0) ParsedSchema() interface{} {
	return schemas.ParsedAdaptiveASHAConfigV0()
}

func (x *AdaptiveASHAConfigV0) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v0/searcher-adaptive-asha.json")
}

func (x *AdaptiveASHAConfigV0) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v0/searcher-adaptive-asha.json")
}

func (x *AdaptiveSimpleConfigV0) ParsedSchema() interface{} {
	return schemas.ParsedAdaptiveSimpleConfigV0()
}

func (x *AdaptiveSimpleConfigV0) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v0/searcher-adaptive-simple.json")
}

func (x *AdaptiveSimpleConfigV0) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v0/searcher-adaptive-simple.json")
}

func (x *AdaptiveConfigV0) ParsedSchema() interface{} {
	return schemas.ParsedAdaptiveConfigV0()
}

func (x *AdaptiveConfigV0) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v0/searcher-adaptive.json")
}

func (x *AdaptiveConfigV0) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v0/searcher-adaptive.json")
}

func (x *AsyncHalvingConfigV0) ParsedSchema() interface{} {
	return schemas.ParsedAsyncHalvingConfigV0()
}

func (x *AsyncHalvingConfigV0) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v0/searcher-async-halving.json")
}

func (x *AsyncHalvingConfigV0) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v0/searcher-async-halving.json")
}

func (x *GridConfigV0) ParsedSchema() interface{} {
	return schemas.ParsedGridConfigV0()
}

func (x *GridConfigV0) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v0/searcher-grid.json")
}

func (x *GridConfigV0) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v0/searcher-grid.json")
}

func (x *PBTConfigV0) ParsedSchema() interface{} {
	return schemas.ParsedPBTConfigV0()
}

func (x *PBTConfigV0) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v0/searcher-pbt.json")
}

func (x *PBTConfigV0) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v0/searcher-pbt.json")
}

func (x *RandomConfigV0) ParsedSchema() interface{} {
	return schemas.ParsedRandomConfigV0()
}

func (x *RandomConfigV0) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v0/searcher-random.json")
}

func (x *RandomConfigV0) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v0/searcher-random.json")
}

func (x *SingleConfigV0) ParsedSchema() interface{} {
	return schemas.ParsedSingleConfigV0()
}

func (x *SingleConfigV0) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v0/searcher-single.json")
}

func (x *SingleConfigV0) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v0/searcher-single.json")
}

func (x *SyncHalvingConfigV0) ParsedSchema() interface{} {
	return schemas.ParsedSyncHalvingConfigV0()
}

func (x *SyncHalvingConfigV0) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v0/searcher-sync-halving.json")
}

func (x *SyncHalvingConfigV0) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v0/searcher-sync-halving.json")
}

func (x *SearcherConfigV0) ParsedSchema() interface{} {
	return schemas.ParsedSearcherConfigV0()
}

func (x *SearcherConfigV0) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v0/searcher.json")
}

func (x *SearcherConfigV0) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v0/searcher.json")
}

func (x *SecurityConfigV0) ParsedSchema() interface{} {
	return schemas.ParsedSecurityConfigV0()
}

func (x *SecurityConfigV0) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v0/security.json")
}

func (x *SecurityConfigV0) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v0/security.json")
}

func (x *SharedFSConfigV0) ParsedSchema() interface{} {
	return schemas.ParsedSharedFSConfigV0()
}

func (x *SharedFSConfigV0) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v0/shared-fs.json")
}

func (x *SharedFSConfigV0) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v0/shared-fs.json")
}

func (x *TensorboardStorageConfigV0) ParsedSchema() interface{} {
	return schemas.ParsedTensorboardStorageConfigV0()
}

func (x *TensorboardStorageConfigV0) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v0/tensorboard-storage.json")
}

func (x *TensorboardStorageConfigV0) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v0/tensorboard-storage.json")
}

func (x *CheckpointStorageConfigV1) ParsedSchema() interface{} {
	return schemas.ParsedCheckpointStorageConfigV1()
}

func (x *CheckpointStorageConfigV1) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v1/checkpoint-storage.json")
}

func (x *CheckpointStorageConfigV1) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v1/checkpoint-storage.json")
}

func (x *ExperimentConfigV1) ParsedSchema() interface{} {
	return schemas.ParsedExperimentConfigV1()
}

func (x *ExperimentConfigV1) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v1/experiment.json")
}

func (x *ExperimentConfigV1) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v1/experiment.json")
}

func (x *OptimizationsConfigV1) ParsedSchema() interface{} {
	return schemas.ParsedOptimizationsConfigV1()
}

func (x *OptimizationsConfigV1) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v1/optimizations.json")
}

func (x *OptimizationsConfigV1) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v1/optimizations.json")
}

func (x *SharedFSConfigV1) ParsedSchema() interface{} {
	return schemas.ParsedSharedFSConfigV1()
}

func (x *SharedFSConfigV1) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v1/shared-fs.json")
}

func (x *SharedFSConfigV1) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v1/shared-fs.json")
}
