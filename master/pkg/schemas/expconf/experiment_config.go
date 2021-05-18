package expconf

import (
	"bytes"
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
	RawBindMounts               BindMountsConfigV0          `json:"bind_mounts"`
	RawCheckpointPolicy         *string                     `json:"checkpoint_policy"`
	RawCheckpointStorage        *CheckpointStorageConfigV0  `json:"checkpoint_storage"`
	RawDataLayer                *DataLayerConfigV0          `json:"data_layer"`
	RawData                     map[string]interface{}      `json:"data"`
	RawDebug                    *bool                       `json:"debug"`
	RawDescription              Description                 `json:"description"`
	RawEntrypoint               *string                     `json:"entrypoint"`
	RawEnvironment              *EnvironmentConfigV0        `json:"environment"`
	RawHyperparameters          HyperparametersV0           `json:"hyperparameters"`
	RawInternal                 *InternalConfigV0           `json:"internal,omitempty"`
	RawLabels                   LabelsV0                    `json:"labels"`
	RawMaxRestarts              *int                        `json:"max_restarts"`
	RawMinCheckpointPeriod      *LengthV0                   `json:"min_checkpoint_period"`
	RawMinValidationPeriod      *LengthV0                   `json:"min_validation_period"`
	RawOptimizations            *OptimizationsConfigV0      `json:"optimizations"`
	RawPerformInitialValidation *bool                       `json:"perform_initial_validation"`
	RawProfiling                *ProfilingConfigV0          `json:"profiling"`
	RawRecordsPerEpoch          *int                        `json:"records_per_epoch"`
	RawReproducibility          *ReproducibilityConfigV0    `json:"reproducibility"`
	RawResources                *ResourcesConfigV0          `json:"resources"`
	RawSchedulingUnit           *int                        `json:"scheduling_unit"`
	RawSearcher                 *SearcherConfigV0           `json:"searcher"`
	RawSecurity                 *SecurityConfigV0           `json:"security,omitempty"`
	RawTensorboardStorage       *TensorboardStorageConfigV0 `json:"tensorboard_storage,omitempty"`
}

// Unit implements the model.InUnits interface.
func (e *ExperimentConfigV0) Unit() Unit {
	return e.RawSearcher.Unit()
}

// Value implements the driver.Valuer interface.
func (e ExperimentConfigV0) Value() (driver.Value, error) {
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
func (e *ExperimentConfigV0) Scan(src interface{}) error {
	byts, ok := src.([]byte)
	if !ok {
		return errors.Errorf("unable to convert to []byte: %v", src)
	}
	config, err := ParseAnyExperimentConfigJSON(byts)
	if err != nil {
		return err
	}
	// This *should* be a copy without any changes, unless perhaps we just shimmed the bytes that
	// were in the database, but to ensure we never allow any un-defaulted experiments anywhere
	// inside the system, we call WithDefaults here.
	*e = schemas.WithDefaults(config).(ExperimentConfigV0)
	return nil
}

// Description is a container struct for handling runtime defaults. It has to be a container so that
// it can be responsible for allocating the nil pointer if one is not provided.  It would be nice if
// you could use `type Description *string` but go won't let you create methods on such a type.
type Description struct {
	RawString *string
}

// WithDefaults implements the Defaultable interface.
func (d Description) WithDefaults() interface{} {
	var s string
	if d.RawString != nil {
		s = *d.RawString
	} else {
		s = fmt.Sprintf(
			"Experiment (%s)",
			petname.Generate(TaskNameGeneratorWords, TaskNameGeneratorSep),
		)
	}
	return Description{&s}
}

// String is part of the Getter/Setter API.
func (d Description) String() string {
	if d.RawString == nil {
		panic("You must call WithDefaults on Description before .String")
	}
	return *d.RawString
}

// SetString is part of the Getter/Setter API.
func (d *Description) SetString(s string) {
	d.RawString = &s
}

// MarshalJSON marshals makes the Description container transparent to marshaling.
func (d Description) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.RawString)
}

// UnmarshalJSON marshals makes the Description container transparent to unmarshaling.
func (d *Description) UnmarshalJSON(bytes []byte) error {
	return json.Unmarshal(bytes, &d.RawString)
}

// InUnits is describes a type that is in terms of a specific unit.
type InUnits interface {
	Unit() Unit
}

// LabelsV0 holds the set of labels on the experiment.
type LabelsV0 map[string]bool

// MarshalJSON implements the json.Marshaler interface.
func (l LabelsV0) MarshalJSON() ([]byte, error) {
	if l == nil {
		return []byte("null"), nil
	}
	labels := make([]string, 0, len(l))
	// var labels []string
	for label := range l {
		labels = append(labels, label)
	}
	return json.Marshal(labels)
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (l *LabelsV0) UnmarshalJSON(data []byte) error {
	if data == nil || bytes.Equal(data, []byte("null")) {
		return nil
	}
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
	// Slots is used by commands while trials use SlotsPerTrial.
	RawSlots *int `json:"slots,omitempty"`

	RawMaxSlots       *int     `json:"max_slots"`
	RawSlotsPerTrial  *int     `json:"slots_per_trial"`
	RawWeight         *float64 `json:"weight"`
	RawNativeParallel *bool    `json:"native_parallel,omitempty"`
	RawShmSize        *int     `json:"shm_size"`
	RawAgentLabel     *string  `json:"agent_label"`
	RawResourcePool   *string  `json:"resource_pool"`
	RawPriority       *int     `json:"priority"`

	RawDevices DevicesConfigV0 `json:"devices"`
}

//go:generate ../gen.sh
// OptimizationsConfigV0 is a legacy config value.
type OptimizationsConfigV0 struct {
	RawAggregationFrequency       *int    `json:"aggregation_frequency"`
	RawAverageAggregatedGradients *bool   `json:"average_aggregated_gradients"`
	RawAverageTrainingMetrics     *bool   `json:"average_training_metrics"`
	RawGradientCompression        *bool   `json:"gradient_compression"`
	RawGradUpdateSizeFile         *string `json:"grad_updates_size_file"`
	RawMixedPrecision             *string `json:"mixed_precision,omitempty"`
	RawTensorFusionThreshold      *int    `json:"tensor_fusion_threshold"`
	RawTensorFusionCycleTime      *int    `json:"tensor_fusion_cycle_time"`
	RawAutoTuneTensorFusion       *bool   `json:"auto_tune_tensor_fusion"`
}

//go:generate ../gen.sh
// BindMountsConfigV0 is the configuration for bind mounts.
type BindMountsConfigV0 []BindMountV0

// Merge implements the Mergable interface.
func (b BindMountsConfigV0) Merge(other interface{}) interface{} {
	tOther := other.(BindMountsConfigV0)
	// Merge by appending.
	out := BindMountsConfigV0{}
	out = append(out, b...)
	out = append(out, tOther...)
	return out
}

//go:generate ../gen.sh
// BindMountV0 configures trial runner filesystem bind mounts.
type BindMountV0 struct {
	RawHostPath      string  `json:"host_path"`
	RawContainerPath string  `json:"container_path"`
	RawReadOnly      *bool   `json:"read_only"`
	RawPropagation   *string `json:"propagation"`
}

//go:generate ../gen.sh
// DevicesConfigV0 is the configuration for devices.
type DevicesConfigV0 []DeviceV0

// Merge implements the Mergable interface.
func (b DevicesConfigV0) Merge(other interface{}) interface{} {
	tOther := other.(DevicesConfigV0)
	// Merge by appending.
	out := DevicesConfigV0{}
	out = append(out, b...)
	out = append(out, tOther...)
	return out
}

//go:generate ../gen.sh
// DeviceV0 configures trial runner filesystem bind mounts.
type DeviceV0 struct {
	RawHostPath      string  `json:"host_path"`
	RawContainerPath string  `json:"container_path"`
	RawMode          *string `json:"mode"`
}

//go:generate ../gen.sh
// ReproducibilityConfigV0 configures parameters related to reproducibility.
type ReproducibilityConfigV0 struct {
	RawExperimentSeed *uint32 `json:"experiment_seed"`
}

// WithDefaults implements the Defaultable interface.
func (r ReproducibilityConfigV0) WithDefaults() interface{} {
	var seed uint32
	if r.RawExperimentSeed != nil {
		seed = *r.RawExperimentSeed
	} else {
		seed = uint32(time.Now().Unix())
	}
	return ReproducibilityConfigV0{&seed}
}

//go:generate ../gen.sh
// SecurityConfigV0 is a legacy config.
type SecurityConfigV0 struct {
	RawKerberos KerberosConfigV0 `json:"kerberos"`
}

//go:generate ../gen.sh
// KerberosConfigV0 is a legacy config.
type KerberosConfigV0 struct {
	RawConfigFile string `json:"config_file"`
}

//go:generate ../gen.sh
// InternalConfigV0 is a legacy config.
type InternalConfigV0 struct {
	RawNative NativeConfigV0 `json:"native"`
}

//go:generate ../gen.sh
// NativeConfigV0 is a legacy config.
type NativeConfigV0 struct {
	RawCommand []string `json:"command"`
}
