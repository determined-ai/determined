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

//go:generate ../gen.sh
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

func runtimeDefaultDescription() *string {
	s := fmt.Sprintf(
		"Experiment (%s)",
		petname.Generate(TaskNameGeneratorWords, TaskNameGeneratorSep),
	)
	return &s
}

// RuntimeDefaults implements the RuntimeDefaultable interface.
func (e ExperimentConfigV0) RuntimeDefaults() interface{} {
	// Description has runtime defaults.
	if e.Description == nil {
		e.Description = runtimeDefaultDescription()
	}
	return e
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
	// var labels []string
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

//go:generate ../gen.sh
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

//go:generate ../gen.sh
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

// BindMountsConfigV0 is the configuration for bind mounts.
type BindMountsConfigV0 []BindMountV0

// Merge implements the Mergable interface.
func (b BindMountsConfig) Merge(other interface{}) interface{} {
	tOther := other.(BindMountsConfig)
	// Merge by appending.
	out := BindMountsConfig{}
	out = append(out, b...)
	out = append(out, tOther...)
	return out
}

//go:generate ../gen.sh
// BindMountV0 configures trial runner filesystem bind mounts.
type BindMountV0 struct {
	HostPath      string  `json:"host_path"`
	ContainerPath string  `json:"container_path"`
	ReadOnly      *bool   `json:"read_only"`
	Propagation   *string `json:"propagation"`
}

//go:generate ../gen.sh
// ReproducibilityConfigV0 configures parameters related to reproducibility.
type ReproducibilityConfigV0 struct {
	ExperimentSeed *uint32 `json:"experiment_seed"`
}

// RuntimeDefaults implements the RuntimeDefaultable interface.
func (r ReproducibilityConfigV0) RuntimeDefaults() interface{} {
	if r.ExperimentSeed == nil {
		t := uint32(time.Now().Unix())
		r.ExperimentSeed = &t
	}
	return r
}

//go:generate ../gen.sh
// SecurityConfigV0 is a legacy config.
type SecurityConfigV0 struct {
	Kerberos *KerberosConfigV0 `json:"kerberos"`
}

//go:generate ../gen.sh
// KerberosConfigV0 is a legacy config.
type KerberosConfigV0 struct {
	ConfigFile *string `json:"config_file"`
}

//go:generate ../gen.sh
// InternalConfigV0 is a legacy config.
type InternalConfigV0 struct {
	Native *NativeConfigV0 `json:"native"`
}

//go:generate ../gen.sh
// NativeConfigV0 is a legacy config.
type NativeConfigV0 struct {
	Command *[]string `json:"command"`
}
