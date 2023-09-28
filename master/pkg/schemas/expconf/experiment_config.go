package expconf

import (
	"bytes"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	petname "github.com/dustinkirkland/golang-petname"
	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/schemas"
)

// ExperimentConfigV0 is a versioned experiment config.
//
//go:generate ../gen.sh
type ExperimentConfigV0 struct {
	RawBindMounts               BindMountsConfigV0          `json:"bind_mounts"`
	RawCheckpointPolicy         *string                     `json:"checkpoint_policy"`
	RawCheckpointStorage        *CheckpointStorageConfigV0  `json:"checkpoint_storage"`
	RawData                     map[string]interface{}      `json:"data"`
	RawDebug                    *bool                       `json:"debug"`
	RawDescription              *string                     `json:"description"`
	RawEntrypoint               *EntrypointV0               `json:"entrypoint"`
	RawEnvironment              *EnvironmentConfigV0        `json:"environment"`
	RawHyperparameters          HyperparametersV0           `json:"hyperparameters"`
	RawLabels                   LabelsV0                    `json:"labels"`
	RawLogPatternPolicies       LogPatternPoliciesConfigV0  `json:"log_pattern_policies"`
	RawMaxRestarts              *int                        `json:"max_restarts"`
	RawMinCheckpointPeriod      *LengthV0                   `json:"min_checkpoint_period"`
	RawMinValidationPeriod      *LengthV0                   `json:"min_validation_period"`
	RawName                     Name                        `json:"name"`
	RawOptimizations            *OptimizationsConfigV0      `json:"optimizations"`
	RawPerformInitialValidation *bool                       `json:"perform_initial_validation"`
	RawProfiling                *ProfilingConfigV0          `json:"profiling"`
	RawProject                  *string                     `json:"project"`
	RawRecordsPerEpoch          *int                        `json:"records_per_epoch"`
	RawReproducibility          *ReproducibilityConfigV0    `json:"reproducibility"`
	RawResources                *ResourcesConfigV0          `json:"resources"`
	RawSchedulingUnit           *int                        `json:"scheduling_unit"`
	RawSearcher                 *SearcherConfigV0           `json:"searcher"`
	RawSecurity                 *SecurityConfigV0           `json:"security,omitempty"`
	RawTensorboardStorage       *TensorboardStorageConfigV0 `json:"tensorboard_storage,omitempty"`
	RawWorkspace                *string                     `json:"workspace"`
	RawSlurmConfig              *SlurmConfigV0              `json:"slurm,omitempty"`
	RawPbsConfig                *PbsConfigV0                `json:"pbs,omitempty"`
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
	*e = schemas.WithDefaults(config)
	return nil
}

// AsLegacy converts a current ExperimentConfig to a (limited capacity) LegacyConfig.
func (e ExperimentConfig) AsLegacy() LegacyConfig {
	return LegacyConfig{
		CheckpointStorage: schemas.Copy(e.CheckpointStorage()),
		BindMounts:        schemas.Copy(e.BindMounts()),
		Environment:       schemas.Copy(e.Environment()),
		Hyperparameters:   schemas.Copy(e.Hyperparameters()),
		Searcher:          e.Searcher().AsLegacy(),
	}
}

// Name is a container struct for handling runtime defaults. It has to be a container so that
// it can be responsible for allocating the nil pointer if one is not provided.  It would be nice if
// you could use `type Name *string` but go won't let you create methods on such a type.
type Name struct {
	RawString *string
}

// WithDefaults implements the Defaultable psuedointerface.
func (d Name) WithDefaults() Name {
	var s string
	if d.RawString != nil {
		s = *d.RawString
	} else {
		s = fmt.Sprintf(
			"Experiment (%s)",
			petname.Generate(TaskNameGeneratorWords, TaskNameGeneratorSep),
		)
	}
	return Name{&s}
}

// String is part of the Getter/Setter API.
func (d Name) String() string {
	if d.RawString == nil {
		panic("You must call WithDefaults on Name before .String")
	}
	return *d.RawString
}

// SetString is part of the Getter/Setter API.
func (d *Name) SetString(s string) {
	d.RawString = &s
}

// MarshalJSON marshals makes the Name container transparent to marshaling.
func (d Name) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.RawString)
}

// UnmarshalJSON marshals makes the Name container transparent to unmarshaling.
func (d *Name) UnmarshalJSON(bytes []byte) error {
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

// SlurmConfigV0 configures experiment resource usage.
//
//go:generate ../gen.sh
type SlurmConfigV0 struct {
	RawSlotsPerNode *int     `json:"slots_per_node,omitempty"`
	RawGpuType      *string  `json:"gpu_type,omitempty"`
	RawSbatchArgs   []string `json:"sbatch_args,omitempty"`
}

// PbsConfigV0 configures experiment resource usage.
//
//go:generate ../gen.sh
type PbsConfigV0 struct {
	RawSlotsPerNode *int     `json:"slots_per_node,omitempty"`
	RawSbatchArgs   []string `json:"pbsbatch_args,omitempty"`
}

// ResourcesConfigV0 configures experiment resource usage.
//
//go:generate ../gen.sh
type ResourcesConfigV0 struct {
	// Slots is used by commands while trials use SlotsPerTrial.
	RawSlots *int `json:"slots,omitempty"`

	RawMaxSlots       *int     `json:"max_slots"`
	RawSlotsPerTrial  *int     `json:"slots_per_trial"`
	RawWeight         *float64 `json:"weight"`
	RawNativeParallel *bool    `json:"native_parallel,omitempty"`
	RawShmSize        *int     `json:"shm_size"`
	RawResourcePool   *string  `json:"resource_pool"`
	RawPriority       *int     `json:"priority"`

	RawDevices DevicesConfigV0 `json:"devices"`
}

// OptimizationsConfigV0 is a legacy config value.
//
//go:generate ../gen.sh
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

// BindMountsConfigV0 is the configuration for bind mounts.
//
//go:generate ../gen.sh
type BindMountsConfigV0 []BindMountV0

// Merge is just merge-by-appending, with a specific form of deduplication.
//
// If other contains entries where ContainerPath() would conflict with any entry in the receiver,
// those entries from other are omitted.  However, there is no deduplication of the receiver's
// entries relative to each other, or of the other's entries relative to each other.  The reasoning
// is that if either the user-provided config or the template config is broken, it would be
// confusing that Merge() would silently fix them.  However it would also be confusing if two valid
// configs got merged together and resulted in a clearly invalid config.
func (b BindMountsConfigV0) Merge(other BindMountsConfigV0) BindMountsConfigV0 {
	out := BindMountsConfigV0{}
	out = append(out, b...)

	// Prevent duplicate container paths as a result of the merge.
	paths := map[string]bool{}
	for _, mount := range b {
		paths[mount.ContainerPath()] = true
	}
	for _, mount := range other {
		if _, ok := paths[mount.ContainerPath()]; !ok {
			out = append(out, mount)
		}
	}
	return out
}

// EntrypointV0 configures the entrypoint script for the experiment.
type EntrypointV0 struct {
	RawEntrypoint interface{}
}

// MarshalJSON marshals makes the EntrypointV0 container transparent to marshaling.
func (e EntrypointV0) MarshalJSON() ([]byte, error) {
	return json.Marshal(e.RawEntrypoint)
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (e *EntrypointV0) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &e.RawEntrypoint)
}

// BindMountV0 configures trial runner filesystem bind mounts.
//
//go:generate ../gen.sh
type BindMountV0 struct {
	RawHostPath      string  `json:"host_path"`
	RawContainerPath string  `json:"container_path"`
	RawReadOnly      *bool   `json:"read_only"`
	RawPropagation   *string `json:"propagation"`
}

// DevicesConfigV0 is the configuration for devices.
//
//go:generate ../gen.sh
type DevicesConfigV0 []DeviceV0

// Merge is just merge-by-appending, with a specific form of deduplication.
// See the comment on BindMountsConfigV0.Merge() for details.
func (d DevicesConfigV0) Merge(other DevicesConfigV0) DevicesConfigV0 {
	out := DevicesConfigV0{}
	out = append(out, d...)

	// Prevent duplicate container paths as a result of the merge.
	paths := map[string]bool{}
	for _, mount := range d {
		paths[mount.ContainerPath()] = true
	}
	for _, mount := range other {
		if _, ok := paths[mount.ContainerPath()]; !ok {
			out = append(out, mount)
		}
	}
	return out
}

// DeviceV0 configures trial runner filesystem bind mounts.
//
//go:generate ../gen.sh
type DeviceV0 struct {
	RawHostPath      string  `json:"host_path"`
	RawContainerPath string  `json:"container_path"`
	RawMode          *string `json:"mode"`
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (d *DeviceV0) UnmarshalJSON(data []byte) error {
	var plain string
	if err := json.Unmarshal(data, &plain); err == nil {
		fields := strings.Split(plain, ":")
		if len(fields) < 2 || len(fields) > 3 {
			return errors.Errorf("invalid device string: %q", plain)
		}
		d.RawHostPath = fields[0]
		d.RawContainerPath = fields[1]
		if len(fields) > 2 {
			d.RawMode = &fields[2]
		}
		return nil
	}

	type DefaultParser *DeviceV0
	return json.Unmarshal(data, DefaultParser(d))
}

// ReproducibilityConfigV0 configures parameters related to reproducibility.
//
//go:generate ../gen.sh
type ReproducibilityConfigV0 struct {
	RawExperimentSeed *uint32 `json:"experiment_seed"`
}

// WithDefaults implements the Defaultable psuedointerface.
func (r ReproducibilityConfigV0) WithDefaults() ReproducibilityConfigV0 {
	var seed uint32
	if r.RawExperimentSeed != nil {
		seed = *r.RawExperimentSeed
	} else {
		seed = uint32(time.Now().Unix())
	}
	return ReproducibilityConfigV0{&seed}
}

// SecurityConfigV0 is a legacy config.
//
//go:generate ../gen.sh
type SecurityConfigV0 struct {
	RawKerberos KerberosConfigV0 `json:"kerberos"`
}

// KerberosConfigV0 is a legacy config.
//
//go:generate ../gen.sh
type KerberosConfigV0 struct {
	RawConfigFile string `json:"config_file"`
}
