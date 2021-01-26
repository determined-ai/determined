// This is a generated file.  Editing it will make you sad.

package expconf

import (
	"github.com/santhosh-tekuri/jsonschema/v2"

	"github.com/determined-ai/determined/master/pkg/schemas"
)

func (b *BindMountV0) ParsedSchema() interface{} {
	return schemas.ParsedBindMountV0()
}

func (b *BindMountV0) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v0/bind-mount.json")
}

func (b *BindMountV0) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v0/bind-mount.json")
}

func (c *CheckpointStorageConfigV0) ParsedSchema() interface{} {
	return schemas.ParsedCheckpointStorageConfigV0()
}

func (c *CheckpointStorageConfigV0) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v0/checkpoint-storage.json")
}

func (c *CheckpointStorageConfigV0) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v0/checkpoint-storage.json")
}

func (g *GCSDataLayerConfigV0) ParsedSchema() interface{} {
	return schemas.ParsedGCSDataLayerConfigV0()
}

func (g *GCSDataLayerConfigV0) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v0/data-layer-gcs.json")
}

func (g *GCSDataLayerConfigV0) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v0/data-layer-gcs.json")
}

func (s *S3DataLayerConfigV0) ParsedSchema() interface{} {
	return schemas.ParsedS3DataLayerConfigV0()
}

func (s *S3DataLayerConfigV0) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v0/data-layer-s3.json")
}

func (s *S3DataLayerConfigV0) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v0/data-layer-s3.json")
}

func (s *SharedFSDataLayerConfigV0) ParsedSchema() interface{} {
	return schemas.ParsedSharedFSDataLayerConfigV0()
}

func (s *SharedFSDataLayerConfigV0) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v0/data-layer-shared-fs.json")
}

func (s *SharedFSDataLayerConfigV0) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v0/data-layer-shared-fs.json")
}

func (d *DataLayerConfigV0) ParsedSchema() interface{} {
	return schemas.ParsedDataLayerConfigV0()
}

func (d *DataLayerConfigV0) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v0/data-layer.json")
}

func (d *DataLayerConfigV0) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v0/data-layer.json")
}

func (e *EnvironmentImageMapV0) ParsedSchema() interface{} {
	return schemas.ParsedEnvironmentImageMapV0()
}

func (e *EnvironmentImageMapV0) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v0/environment-image-map.json")
}

func (e *EnvironmentImageMapV0) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v0/environment-image-map.json")
}

func (e *EnvironmentImageV0) ParsedSchema() interface{} {
	return schemas.ParsedEnvironmentImageV0()
}

func (e *EnvironmentImageV0) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v0/environment-image.json")
}

func (e *EnvironmentImageV0) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v0/environment-image.json")
}

func (e *EnvironmentVariablesMapV0) ParsedSchema() interface{} {
	return schemas.ParsedEnvironmentVariablesMapV0()
}

func (e *EnvironmentVariablesMapV0) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v0/environment-variables-map.json")
}

func (e *EnvironmentVariablesMapV0) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v0/environment-variables-map.json")
}

func (e *EnvironmentVariablesV0) ParsedSchema() interface{} {
	return schemas.ParsedEnvironmentVariablesV0()
}

func (e *EnvironmentVariablesV0) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v0/environment-variables.json")
}

func (e *EnvironmentVariablesV0) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v0/environment-variables.json")
}

func (e *EnvironmentConfigV0) ParsedSchema() interface{} {
	return schemas.ParsedEnvironmentConfigV0()
}

func (e *EnvironmentConfigV0) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v0/environment.json")
}

func (e *EnvironmentConfigV0) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v0/environment.json")
}

func (e *ExperimentConfigV0) ParsedSchema() interface{} {
	return schemas.ParsedExperimentConfigV0()
}

func (e *ExperimentConfigV0) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v0/experiment.json")
}

func (e *ExperimentConfigV0) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v0/experiment.json")
}

func (g *GCSConfigV0) ParsedSchema() interface{} {
	return schemas.ParsedGCSConfigV0()
}

func (g *GCSConfigV0) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v0/gcs.json")
}

func (g *GCSConfigV0) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v0/gcs.json")
}

func (h *HDFSConfigV0) ParsedSchema() interface{} {
	return schemas.ParsedHDFSConfigV0()
}

func (h *HDFSConfigV0) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v0/hdfs.json")
}

func (h *HDFSConfigV0) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v0/hdfs.json")
}

func (c *CategoricalHyperparameterV0) ParsedSchema() interface{} {
	return schemas.ParsedCategoricalHyperparameterV0()
}

func (c *CategoricalHyperparameterV0) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v0/hyperparameter-categorical.json")
}

func (c *CategoricalHyperparameterV0) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v0/hyperparameter-categorical.json")
}

func (c *ConstHyperparameterV0) ParsedSchema() interface{} {
	return schemas.ParsedConstHyperparameterV0()
}

func (c *ConstHyperparameterV0) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v0/hyperparameter-const.json")
}

func (c *ConstHyperparameterV0) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v0/hyperparameter-const.json")
}

func (d *DoubleHyperparameterV0) ParsedSchema() interface{} {
	return schemas.ParsedDoubleHyperparameterV0()
}

func (d *DoubleHyperparameterV0) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v0/hyperparameter-double.json")
}

func (d *DoubleHyperparameterV0) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v0/hyperparameter-double.json")
}

func (i *IntHyperparameterV0) ParsedSchema() interface{} {
	return schemas.ParsedIntHyperparameterV0()
}

func (i *IntHyperparameterV0) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v0/hyperparameter-int.json")
}

func (i *IntHyperparameterV0) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v0/hyperparameter-int.json")
}

func (l *LogHyperparameterV0) ParsedSchema() interface{} {
	return schemas.ParsedLogHyperparameterV0()
}

func (l *LogHyperparameterV0) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v0/hyperparameter-log.json")
}

func (l *LogHyperparameterV0) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v0/hyperparameter-log.json")
}

func (h *HyperparameterV0) ParsedSchema() interface{} {
	return schemas.ParsedHyperparameterV0()
}

func (h *HyperparameterV0) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v0/hyperparameter.json")
}

func (h *HyperparameterV0) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v0/hyperparameter.json")
}

func (h *HyperparametersV0) ParsedSchema() interface{} {
	return schemas.ParsedHyperparametersV0()
}

func (h *HyperparametersV0) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v0/hyperparameters.json")
}

func (h *HyperparametersV0) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v0/hyperparameters.json")
}

func (i *InternalConfigV0) ParsedSchema() interface{} {
	return schemas.ParsedInternalConfigV0()
}

func (i *InternalConfigV0) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v0/internal.json")
}

func (i *InternalConfigV0) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v0/internal.json")
}

func (k *KerberosConfigV0) ParsedSchema() interface{} {
	return schemas.ParsedKerberosConfigV0()
}

func (k *KerberosConfigV0) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v0/kerberos.json")
}

func (k *KerberosConfigV0) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v0/kerberos.json")
}

func (l *LengthV0) ParsedSchema() interface{} {
	return schemas.ParsedLengthV0()
}

func (l *LengthV0) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v0/length.json")
}

func (l *LengthV0) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v0/length.json")
}

func (o *OptimizationsConfigV0) ParsedSchema() interface{} {
	return schemas.ParsedOptimizationsConfigV0()
}

func (o *OptimizationsConfigV0) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v0/optimizations.json")
}

func (o *OptimizationsConfigV0) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v0/optimizations.json")
}

func (r *ReproducibilityConfigV0) ParsedSchema() interface{} {
	return schemas.ParsedReproducibilityConfigV0()
}

func (r *ReproducibilityConfigV0) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v0/reproducibility.json")
}

func (r *ReproducibilityConfigV0) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v0/reproducibility.json")
}

func (r *ResourcesConfigV0) ParsedSchema() interface{} {
	return schemas.ParsedResourcesConfigV0()
}

func (r *ResourcesConfigV0) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v0/resources.json")
}

func (r *ResourcesConfigV0) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v0/resources.json")
}

func (s *S3ConfigV0) ParsedSchema() interface{} {
	return schemas.ParsedS3ConfigV0()
}

func (s *S3ConfigV0) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v0/s3.json")
}

func (s *S3ConfigV0) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v0/s3.json")
}

func (a *AdaptiveASHAConfigV0) ParsedSchema() interface{} {
	return schemas.ParsedAdaptiveASHAConfigV0()
}

func (a *AdaptiveASHAConfigV0) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v0/searcher-adaptive-asha.json")
}

func (a *AdaptiveASHAConfigV0) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v0/searcher-adaptive-asha.json")
}

func (a *AdaptiveSimpleConfigV0) ParsedSchema() interface{} {
	return schemas.ParsedAdaptiveSimpleConfigV0()
}

func (a *AdaptiveSimpleConfigV0) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v0/searcher-adaptive-simple.json")
}

func (a *AdaptiveSimpleConfigV0) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v0/searcher-adaptive-simple.json")
}

func (a *AdaptiveConfigV0) ParsedSchema() interface{} {
	return schemas.ParsedAdaptiveConfigV0()
}

func (a *AdaptiveConfigV0) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v0/searcher-adaptive.json")
}

func (a *AdaptiveConfigV0) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v0/searcher-adaptive.json")
}

func (a *AsyncHalvingConfigV0) ParsedSchema() interface{} {
	return schemas.ParsedAsyncHalvingConfigV0()
}

func (a *AsyncHalvingConfigV0) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v0/searcher-async-halving.json")
}

func (a *AsyncHalvingConfigV0) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v0/searcher-async-halving.json")
}

func (g *GridConfigV0) ParsedSchema() interface{} {
	return schemas.ParsedGridConfigV0()
}

func (g *GridConfigV0) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v0/searcher-grid.json")
}

func (g *GridConfigV0) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v0/searcher-grid.json")
}

func (p *PBTConfigV0) ParsedSchema() interface{} {
	return schemas.ParsedPBTConfigV0()
}

func (p *PBTConfigV0) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v0/searcher-pbt.json")
}

func (p *PBTConfigV0) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v0/searcher-pbt.json")
}

func (r *RandomConfigV0) ParsedSchema() interface{} {
	return schemas.ParsedRandomConfigV0()
}

func (r *RandomConfigV0) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v0/searcher-random.json")
}

func (r *RandomConfigV0) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v0/searcher-random.json")
}

func (s *SingleConfigV0) ParsedSchema() interface{} {
	return schemas.ParsedSingleConfigV0()
}

func (s *SingleConfigV0) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v0/searcher-single.json")
}

func (s *SingleConfigV0) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v0/searcher-single.json")
}

func (s *SyncHalvingConfigV0) ParsedSchema() interface{} {
	return schemas.ParsedSyncHalvingConfigV0()
}

func (s *SyncHalvingConfigV0) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v0/searcher-sync-halving.json")
}

func (s *SyncHalvingConfigV0) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v0/searcher-sync-halving.json")
}

func (s *SearcherConfigV0) ParsedSchema() interface{} {
	return schemas.ParsedSearcherConfigV0()
}

func (s *SearcherConfigV0) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v0/searcher.json")
}

func (s *SearcherConfigV0) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v0/searcher.json")
}

func (s *SecurityConfigV0) ParsedSchema() interface{} {
	return schemas.ParsedSecurityConfigV0()
}

func (s *SecurityConfigV0) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v0/security.json")
}

func (s *SecurityConfigV0) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v0/security.json")
}

func (s *SharedFSConfigV0) ParsedSchema() interface{} {
	return schemas.ParsedSharedFSConfigV0()
}

func (s *SharedFSConfigV0) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v0/shared-fs.json")
}

func (s *SharedFSConfigV0) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v0/shared-fs.json")
}

func (t *TensorboardStorageConfigV0) ParsedSchema() interface{} {
	return schemas.ParsedTensorboardStorageConfigV0()
}

func (t *TensorboardStorageConfigV0) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v0/tensorboard-storage.json")
}

func (t *TensorboardStorageConfigV0) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v0/tensorboard-storage.json")
}

func (c *CheckpointStorageConfigV1) ParsedSchema() interface{} {
	return schemas.ParsedCheckpointStorageConfigV1()
}

func (c *CheckpointStorageConfigV1) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v1/checkpoint-storage.json")
}

func (c *CheckpointStorageConfigV1) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v1/checkpoint-storage.json")
}

func (e *ExperimentConfigV1) ParsedSchema() interface{} {
	return schemas.ParsedExperimentConfigV1()
}

func (e *ExperimentConfigV1) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v1/experiment.json")
}

func (e *ExperimentConfigV1) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v1/experiment.json")
}

func (o *OptimizationsConfigV1) ParsedSchema() interface{} {
	return schemas.ParsedOptimizationsConfigV1()
}

func (o *OptimizationsConfigV1) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v1/optimizations.json")
}

func (o *OptimizationsConfigV1) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v1/optimizations.json")
}

func (s *SharedFSConfigV1) ParsedSchema() interface{} {
	return schemas.ParsedSharedFSConfigV1()
}

func (s *SharedFSConfigV1) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v1/shared-fs.json")
}

func (s *SharedFSConfigV1) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v1/shared-fs.json")
}
