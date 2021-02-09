// This is a generated file.  Editing it will make you sad.

package expconf

import (
	"github.com/santhosh-tekuri/jsonschema/v2"

	"github.com/determined-ai/determined/master/pkg/schemas"
)

func (b *BindMountV1) ParsedSchema() interface{} {
	return schemas.ParsedBindMountV1()
}

func (b *BindMountV1) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v1/bind-mount.json")
}

func (b *BindMountV1) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v1/bind-mount.json")
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

func (d *DataLayerGCSConfigV1) ParsedSchema() interface{} {
	return schemas.ParsedDataLayerGCSConfigV1()
}

func (d *DataLayerGCSConfigV1) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v1/data-layer-gcs.json")
}

func (d *DataLayerGCSConfigV1) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v1/data-layer-gcs.json")
}

func (d *DataLayerS3ConfigV1) ParsedSchema() interface{} {
	return schemas.ParsedDataLayerS3ConfigV1()
}

func (d *DataLayerS3ConfigV1) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v1/data-layer-s3.json")
}

func (d *DataLayerS3ConfigV1) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v1/data-layer-s3.json")
}

func (d *DataLayerSharedFSConfigV1) ParsedSchema() interface{} {
	return schemas.ParsedDataLayerSharedFSConfigV1()
}

func (d *DataLayerSharedFSConfigV1) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v1/data-layer-shared-fs.json")
}

func (d *DataLayerSharedFSConfigV1) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v1/data-layer-shared-fs.json")
}

func (d *DataLayerConfigV1) ParsedSchema() interface{} {
	return schemas.ParsedDataLayerConfigV1()
}

func (d *DataLayerConfigV1) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v1/data-layer.json")
}

func (d *DataLayerConfigV1) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v1/data-layer.json")
}

func (e *EnvironmentImageV1) ParsedSchema() interface{} {
	return schemas.ParsedEnvironmentImageV1()
}

func (e *EnvironmentImageV1) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v1/environment-image.json")
}

func (e *EnvironmentImageV1) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v1/environment-image.json")
}

func (e *EnvironmentVariablesV1) ParsedSchema() interface{} {
	return schemas.ParsedEnvironmentVariablesV1()
}

func (e *EnvironmentVariablesV1) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v1/environment-variables.json")
}

func (e *EnvironmentVariablesV1) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v1/environment-variables.json")
}

func (e *EnvironmentConfigV1) ParsedSchema() interface{} {
	return schemas.ParsedEnvironmentConfigV1()
}

func (e *EnvironmentConfigV1) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v1/environment.json")
}

func (e *EnvironmentConfigV1) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v1/environment.json")
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

func (g *GCSConfigV1) ParsedSchema() interface{} {
	return schemas.ParsedGCSConfigV1()
}

func (g *GCSConfigV1) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v1/gcs.json")
}

func (g *GCSConfigV1) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v1/gcs.json")
}

func (h *HDFSConfigV1) ParsedSchema() interface{} {
	return schemas.ParsedHDFSConfigV1()
}

func (h *HDFSConfigV1) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v1/hdfs.json")
}

func (h *HDFSConfigV1) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v1/hdfs.json")
}

func (c *CategoricalHyperparameterV1) ParsedSchema() interface{} {
	return schemas.ParsedCategoricalHyperparameterV1()
}

func (c *CategoricalHyperparameterV1) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v1/hyperparameter-categorical.json")
}

func (c *CategoricalHyperparameterV1) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v1/hyperparameter-categorical.json")
}

func (c *ConstHyperparameterV1) ParsedSchema() interface{} {
	return schemas.ParsedConstHyperparameterV1()
}

func (c *ConstHyperparameterV1) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v1/hyperparameter-const.json")
}

func (c *ConstHyperparameterV1) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v1/hyperparameter-const.json")
}

func (d *DoubleHyperparameterV1) ParsedSchema() interface{} {
	return schemas.ParsedDoubleHyperparameterV1()
}

func (d *DoubleHyperparameterV1) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v1/hyperparameter-double.json")
}

func (d *DoubleHyperparameterV1) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v1/hyperparameter-double.json")
}

func (i *IntHyperparameterV1) ParsedSchema() interface{} {
	return schemas.ParsedIntHyperparameterV1()
}

func (i *IntHyperparameterV1) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v1/hyperparameter-int.json")
}

func (i *IntHyperparameterV1) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v1/hyperparameter-int.json")
}

func (l *LogHyperparameterV1) ParsedSchema() interface{} {
	return schemas.ParsedLogHyperparameterV1()
}

func (l *LogHyperparameterV1) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v1/hyperparameter-log.json")
}

func (l *LogHyperparameterV1) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v1/hyperparameter-log.json")
}

func (h *HyperparameterV1) ParsedSchema() interface{} {
	return schemas.ParsedHyperparameterV1()
}

func (h *HyperparameterV1) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v1/hyperparameter.json")
}

func (h *HyperparameterV1) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v1/hyperparameter.json")
}

func (i *InternalConfigV1) ParsedSchema() interface{} {
	return schemas.ParsedInternalConfigV1()
}

func (i *InternalConfigV1) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v1/internal.json")
}

func (i *InternalConfigV1) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v1/internal.json")
}

func (l *LengthV1) ParsedSchema() interface{} {
	return schemas.ParsedLengthV1()
}

func (l *LengthV1) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v1/length.json")
}

func (l *LengthV1) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v1/length.json")
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

func (r *ResourcesConfigV1) ParsedSchema() interface{} {
	return schemas.ParsedResourcesConfigV1()
}

func (r *ResourcesConfigV1) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v1/resources.json")
}

func (r *ResourcesConfigV1) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v1/resources.json")
}

func (s *S3ConfigV1) ParsedSchema() interface{} {
	return schemas.ParsedS3ConfigV1()
}

func (s *S3ConfigV1) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v1/s3.json")
}

func (s *S3ConfigV1) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v1/s3.json")
}

func (a *AdaptiveASHASearcherConfigV1) ParsedSchema() interface{} {
	return schemas.ParsedAdaptiveASHASearcherConfigV1()
}

func (a *AdaptiveASHASearcherConfigV1) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v1/searcher-adaptive-asha.json")
}

func (a *AdaptiveASHASearcherConfigV1) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v1/searcher-adaptive-asha.json")
}

func (a *AdaptiveSimpleSearcherConfigV1) ParsedSchema() interface{} {
	return schemas.ParsedAdaptiveSimpleSearcherConfigV1()
}

func (a *AdaptiveSimpleSearcherConfigV1) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v1/searcher-adaptive-simple.json")
}

func (a *AdaptiveSimpleSearcherConfigV1) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v1/searcher-adaptive-simple.json")
}

func (a *AdaptiveSearcherConfigV1) ParsedSchema() interface{} {
	return schemas.ParsedAdaptiveSearcherConfigV1()
}

func (a *AdaptiveSearcherConfigV1) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v1/searcher-adaptive.json")
}

func (a *AdaptiveSearcherConfigV1) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v1/searcher-adaptive.json")
}

func (a *AsyncHalvingSearcherConfigV1) ParsedSchema() interface{} {
	return schemas.ParsedAsyncHalvingSearcherConfigV1()
}

func (a *AsyncHalvingSearcherConfigV1) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v1/searcher-async-halving.json")
}

func (a *AsyncHalvingSearcherConfigV1) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v1/searcher-async-halving.json")
}

func (g *GridSearcherConfigV1) ParsedSchema() interface{} {
	return schemas.ParsedGridSearcherConfigV1()
}

func (g *GridSearcherConfigV1) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v1/searcher-grid.json")
}

func (g *GridSearcherConfigV1) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v1/searcher-grid.json")
}

func (p *PBTSearcherConfigV1) ParsedSchema() interface{} {
	return schemas.ParsedPBTSearcherConfigV1()
}

func (p *PBTSearcherConfigV1) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v1/searcher-pbt.json")
}

func (p *PBTSearcherConfigV1) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v1/searcher-pbt.json")
}

func (r *RandomSearcherConfigV1) ParsedSchema() interface{} {
	return schemas.ParsedRandomSearcherConfigV1()
}

func (r *RandomSearcherConfigV1) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v1/searcher-random.json")
}

func (r *RandomSearcherConfigV1) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v1/searcher-random.json")
}

func (s *SingleSearcherConfigV1) ParsedSchema() interface{} {
	return schemas.ParsedSingleSearcherConfigV1()
}

func (s *SingleSearcherConfigV1) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v1/searcher-single.json")
}

func (s *SingleSearcherConfigV1) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v1/searcher-single.json")
}

func (s *SyncHalvingSearcherConfigV1) ParsedSchema() interface{} {
	return schemas.ParsedSyncHalvingSearcherConfigV1()
}

func (s *SyncHalvingSearcherConfigV1) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v1/searcher-sync-halving.json")
}

func (s *SyncHalvingSearcherConfigV1) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v1/searcher-sync-halving.json")
}

func (s *SearcherConfigV1) ParsedSchema() interface{} {
	return schemas.ParsedSearcherConfigV1()
}

func (s *SearcherConfigV1) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v1/searcher.json")
}

func (s *SearcherConfigV1) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v1/searcher.json")
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

func (t *TestRootV1) ParsedSchema() interface{} {
	return schemas.ParsedTestRootV1()
}

func (t *TestRootV1) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v1/test-root.json")
}

func (t *TestRootV1) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v1/test-root.json")
}

func (t *TestSubV1) ParsedSchema() interface{} {
	return schemas.ParsedTestSubV1()
}

func (t *TestSubV1) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v1/test-sub.json")
}

func (t *TestSubV1) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v1/test-sub.json")
}

func (t *TestUnionAV1) ParsedSchema() interface{} {
	return schemas.ParsedTestUnionAV1()
}

func (t *TestUnionAV1) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v1/test-union-a.json")
}

func (t *TestUnionAV1) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v1/test-union-a.json")
}

func (t *TestUnionBV1) ParsedSchema() interface{} {
	return schemas.ParsedTestUnionBV1()
}

func (t *TestUnionBV1) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v1/test-union-b.json")
}

func (t *TestUnionBV1) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v1/test-union-b.json")
}

func (t *TestUnionV1) ParsedSchema() interface{} {
	return schemas.ParsedTestUnionV1()
}

func (t *TestUnionV1) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v1/test-union.json")
}

func (t *TestUnionV1) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v1/test-union.json")
}
