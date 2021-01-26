package expconf

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	petname "github.com/dustinkirkland/golang-petname"
	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/schemas"
)

// ExperimentConfigV0 is a versioned experiment config.
type ExperimentConfigV0 struct {
	// Fields which can be omitted or defined at the cluster level.
	BindMounts               *BindMountsConfigV0         `json:"bind_mounts"`
	CheckpointPolicy         *string                     `json:"checkpoint_policy"`
	CheckpointStorage        *CheckpointStorageConfigV0  `json:"checkpoint_storage"`
	DataLayer                *DataLayerConfigV0          `json:"data_layer"`
	Data                     *map[string]interface{}     `json:"data"`
	Debug                    *bool                       `json:"debug"`
	Description              *string                     `json:"description"`
	Entrypoint               *string                     `json:"entrypoint"`
	Environment              *EnvironmentConfigV0        `json:"environment"`
	Internal                 *InternalConfigV0           `json:"internal,omitempty"`
	Labels                   *LabelsV0                   `json:"labels"`
	MaxRestarts              *int                        `json:"max_restarts"`
	MinCheckpointPeriod      *LengthV0                   `json:"min_checkpoint_period"`
	MinValidationPeriod      *LengthV0                   `json:"min_validation_period"`
	Optimizations            *OptimizationsConfigV0      `json:"optimizations"`
	PerformInitialValidation *bool                       `json:"perform_initial_validation"`
	RecordsPerEpoch          *int                        `json:"records_per_epoch"`
	Reproducibility          *ReproducibilityConfigV0    `json:"reproducibility"`
	Resources                *ResourcesConfigV0          `json:"resources"`
	SchedulingUnit           *int                        `json:"scheduling_unit"`
	Security                 *SecurityConfigV0           `json:"security,omitempty"`
	TensorboardStorage       *TensorboardStorageConfigV0 `json:"tensorboard_storage,omitempty"`

	// Fields which must be defined by the user.
	Hyperparameters HyperparametersV0 `json:"hyperparameters"`
	Searcher        SearcherConfigV0  `json:"searcher"`
}

func runtimeDefaultDescription(descriptionField **string) {
	if *descriptionField == nil {
		s := fmt.Sprintf(
			"Experiment (%s)",
			petname.Generate(TaskNameGeneratorWords, TaskNameGeneratorSep),
		)
		*descriptionField = &s
	}
}

// RuntimeDefaults implements the RuntimeDefaultable interface.
func (e *ExperimentConfigV0) RuntimeDefaults() {
	runtimeDefaultDescription(&e.Description)
}

// shim will convert a valid-but-maybe-incomplete V0 config into a V1 config.
func (e *ExperimentConfigV0) shim(out *ExperimentConfigV1) error {
	// Set obsolete fields to nil so they do not show up in the marshaled config.
	// This works because the only transitions between V0 and V1 were removing obselete fields.
	e.Security = nil
	e.TensorboardStorage = nil

	if e.CheckpointStorage != nil && e.CheckpointStorage.SharedFSConfig != nil {
		e.CheckpointStorage.SharedFSConfig.ContainerPath = nil
		e.CheckpointStorage.SharedFSConfig.CheckpointPath = nil
		e.CheckpointStorage.SharedFSConfig.TensorboardPath = nil
	}

	if e.Optimizations != nil {
		e.Optimizations.MixedPrecision = nil
	}

	byts, err := json.Marshal(e)
	if err != nil {
		return errors.Wrap(err, "unable to marshal v0 config")
	}

	// Ensure that the v1 config is valid; it may be stricter.
	err = schemas.SaneBytes(out, byts)
	if err != nil {
		return errors.Wrap(err, "shimmed v0 config is not a valid v1 config")
	}

	err = json.Unmarshal(byts, out)
	if err != nil {
		return errors.Wrap(err, "unable to unmarshal v0 config into v1")
	}

	return nil
}

// ExperimentConfigV1 is a versioned experiment config.
type ExperimentConfigV1 struct {
	// Fields which can be omitted or defined at the cluster level.
	BindMounts               *BindMountsConfigV0        `json:"bind_mounts"`
	CheckpointPolicy         *string                    `json:"checkpoint_policy"`
	CheckpointStorage        *CheckpointStorageConfigV1 `json:"checkpoint_storage"`
	DataLayer                *DataLayerConfigV0         `json:"data_layer"`
	Data                     *map[string]interface{}    `json:"data"`
	Debug                    *bool                      `json:"debug"`
	Description              *string                    `json:"description"`
	Environment              *EnvironmentConfigV0       `json:"environment"`
	Internal                 *InternalConfigV0          `json:"internal"`
	Labels                   *LabelsV0                  `json:"labels"`
	MaxRestarts              *int                       `json:"max_restarts"`
	MinCheckpointPeriod      *LengthV0                  `json:"min_checkpoint_period"`
	MinValidationPeriod      *LengthV0                  `json:"min_validation_period"`
	Optimizations            *OptimizationsConfigV0     `json:"optimizations"`
	PerformInitialValidation *bool                      `json:"perform_initial_validation"`
	RecordsPerEpoch          *int                       `json:"records_per_epoch"`
	Reproducibility          *ReproducibilityConfigV0   `json:"reproducibility"`
	Resources                *ResourcesConfigV0         `json:"resources"`
	SchedulingUnit           *int                       `json:"scheduling_unit"`

	// Fields which must be defined by the user.
	Entrypoint      string            `json:"entrypoint"`
	Hyperparameters HyperparametersV0 `json:"hyperparameters"`
	Searcher        SearcherConfigV0  `json:"searcher"`
}

// RuntimeDefaults implements the RuntimeDefaultable interface.
func (e *ExperimentConfigV1) RuntimeDefaults() {
	runtimeDefaultDescription(&e.Description)
}

// Unit implements the model.InUnits interface.
func (e *ExperimentConfig) Unit() Unit {
	return e.Searcher.Unit()
}

// Value implements the driver.Valuer interface.
func (e ExperimentConfig) Value() (driver.Value, error) {
	// Validate the object before passing it to the database.
	err := schemas.IsComplete(&e)
	if err != nil {
		return nil, errors.Wrap(err, "refusing to save invalid experiment config")
	}

	byts, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}

	return byts, nil
}

// Scan implements the db.Scanner interface.
func (e *ExperimentConfig) Scan(src interface{}) error {
	byts, ok := src.([]byte)
	if !ok {
		return errors.Errorf("unable to convert to []byte: %v", src)
	}
	config, err := ParseAnyExperimentConfigJSON(byts)
	if err != nil {
		return err
	}
	*e = config
	return nil
}

// InUnits is describes a type that is in terms of a specific unit.
type InUnits interface {
	Unit() Unit
}

// LabelsV0 holds the set of labels on the experiment.
type LabelsV0 map[string]bool

// MarshalJSON implements the json.Marshaler interface.
func (l LabelsV0) MarshalJSON() ([]byte, error) {
	labels := make([]string, 0, len(l))
	for label := range l {
		labels = append(labels, label)
	}
	return json.Marshal(labels)
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (l *LabelsV0) UnmarshalJSON(data []byte) error {
	if *l == nil {
		*l = make(map[string]bool)
	}
	labels := make([]string, 0)
	if err := json.Unmarshal(data, &labels); err == nil {
		for _, label := range labels {
			(*l)[label] = true
		}
		return nil
	}
	labelMap := make(map[string]bool)
	err := json.Unmarshal(data, &labelMap)
	for label := range labelMap {
		(*l)[label] = true
	}
	return err
}

// ResourcesConfigV0 configures experiment resource usage.
type ResourcesConfigV0 struct {
	MaxSlots       *int     `json:"max_slots"`
	SlotsPerTrial  *int     `json:"slots_per_trial"`
	Weight         *float64 `json:"weight"`
	NativeParallel *bool    `json:"native_parallel,omitempty"`
	ShmSize        *int     `json:"shm_size,omitempty"`
	AgentLabel     *string  `json:"agent_label"`
	ResourcePool   *string  `json:"resource_pool"`
	Priority       *int     `json:"priority"`
}

// OptimizationsConfigV0 is a legacy config value.
type OptimizationsConfigV0 struct {
	AggregationFrequency       *int    `json:"aggregation_frequency"`
	AverageAggregatedGradients *bool   `json:"average_aggregated_gradients"`
	AverageTrainingMetrics     *bool   `json:"average_training_metrics"`
	GradientCompression        *bool   `json:"gradient_compression"`
	GradUpdateSizeFile         *string `json:"grad_updates_size_file"`
	MixedPrecision             *string `json:"mixed_precision,omitempty"`
	TensorFusionThreshold      *int    `json:"tensor_fusion_threshold"`
	TensorFusionCycleTime      *int    `json:"tensor_fusion_cycle_time"`
	AutoTuneTensorFusion       *bool   `json:"auto_tune_tensor_fusion"`
}

// OptimizationsConfigV1 configures performance optimizations for Horovod training.
type OptimizationsConfigV1 struct {
	AggregationFrequency       *int    `json:"aggregation_frequency"`
	AverageAggregatedGradients *bool   `json:"average_aggregated_gradients"`
	AverageTrainingMetrics     *bool   `json:"average_training_metrics"`
	GradientCompression        *bool   `json:"gradient_compression"`
	GradUpdateSizeFile         *string `json:"grad_updates_size_file"`
	TensorFusionThreshold      *int    `json:"tensor_fusion_threshold"`
	TensorFusionCycleTime      *int    `json:"tensor_fusion_cycle_time"`
	AutoTuneTensorFusion       *bool   `json:"auto_tune_tensor_fusion"`
}

// BindMountsConfigV0 is the configuration for bind mounts.
type BindMountsConfigV0 []BindMountV0

// Merge implements the mergable interface.
func (b *BindMountsConfig) Merge(src interface{}) {
	*b = append(*b, src.(BindMountsConfig)...)
}

// BindMountV0 configures trial runner filesystem bind mounts.
type BindMountV0 struct {
	HostPath      string  `json:"host_path"`
	ContainerPath string  `json:"container_path"`
	ReadOnly      *bool   `json:"read_only"`
	Propagation   *string `json:"propagation"`
}

// ReproducibilityConfigV0 configures parameters related to reproducibility.
type ReproducibilityConfigV0 struct {
	ExperimentSeed *uint32 `json:"experiment_seed"`
}

// RuntimeDefaults implements the RuntimeDefaultable interface.
func (r *ReproducibilityConfigV0) RuntimeDefaults() {
	if r.ExperimentSeed == nil {
		t := uint32(time.Now().Unix())
		r.ExperimentSeed = &t
	}
}

// SecurityConfigV0 is a legacy config.
type SecurityConfigV0 struct {
	Kerberos *KerberosConfigV0 `json:"kerberos"`
}

// KerberosConfigV0 is a legacy config.
type KerberosConfigV0 struct {
	ConfigFile *string `json:"config_file"`
}

// InternalConfigV0 is a legacy config.
type InternalConfigV0 struct {
	Native *NativeConfigV0 `json:"native"`
}

// NativeConfigV0 is a legacy config.
type NativeConfigV0 struct {
	Command *[]string `json:"command"`
}
