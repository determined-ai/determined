package model

import (
	"database/sql/driver"
	"encoding/json"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/check"
)

// ExperimentConfig is the defaulted configuration.
type ExperimentConfig struct {
	Description        string                    `json:"description"`
	Labels             Labels                    `json:"labels,omitempty"`
	Data               map[string]interface{}    `json:"data,omitempty"`
	CheckpointStorage  CheckpointStorageConfig   `json:"checkpoint_storage"`
	TensorboardStorage *TensorboardStorageConfig `json:"tensorboard_storage,omitempty"`
	CheckpointPeriod   Length                    `json:"checkpoint_period"`
	ValidationPeriod   Length                    `json:"validation_period"`
	CheckpointPolicy   string                    `json:"checkpoint_policy"`
	Hyperparameters    Hyperparameters           `json:"hyperparameters"`
	Searcher           SearcherConfig            `json:"searcher"`
	Resources          ResourcesConfig           `json:"resources"`
	Optimizations      OptimizationsConfig       `json:"optimizations"`
	RecordsPerEpoch    int                       `json:"records_per_epoch"`
	BatchesPerStep     int                       `json:"batches_per_step"`
	BindMounts         []BindMount               `json:"bind_mounts,omitempty"`
	Environment        Environment               `json:"environment"`
	Reproducibility    ReproducibilityConfig     `json:"reproducibility"`
	MaxRestarts        int                       `json:"max_restarts"`
	Security           *SecurityConfig           `json:"security,omitempty"`
	Debug              bool                      `json:"debug"`
	Internal           *InternalConfig           `json:"internal"`
	Entrypoint         string                    `json:"entrypoint"`
	DataLayer          DataLayerConfig           `json:"data_layer"`

	// Deprecated
	MinCheckpointPeriod *int `json:"min_checkpoint_period"`
	MinValidationPeriod *int `json:"min_validation_period"`
}

// Validate implements the check.Validatable interface.
func (e ExperimentConfig) Validate() []error {
	// Do some checks for grid search; since this involves looking at both the searcher config and the
	// hyperparameter config, we have to do it at this level.
	// - Check that counts are specified for all parameters.
	// - Compute the total number of trials that would be created and check that it is not too large.
	gridTrials := 1
	noCountParams := make([]string, 0)
	if e.Searcher.GridConfig != nil {
		e.Hyperparameters.Each(func(name string, param Hyperparameter) {
			mult := 1
			switch {
			case param.IntHyperparameter != nil:
				p := param.IntHyperparameter
				switch {
				case p.Count == nil:
					noCountParams = append(noCountParams, name)
				case *p.Count > p.Maxval-p.Minval:
					// If the count is greater than the number of possible values, grid search will clamp it down.
					mult = p.Maxval - p.Minval
				default:
					mult = *p.Count
				}
			case param.DoubleHyperparameter != nil:
				p := param.DoubleHyperparameter
				if p.Count == nil {
					noCountParams = append(noCountParams, name)
				} else {
					mult = *p.Count
				}
			case param.LogHyperparameter != nil:
				p := param.LogHyperparameter
				if p.Count == nil {
					noCountParams = append(noCountParams, name)
				} else {
					mult = *p.Count
				}
			case param.CategoricalHyperparameter != nil:
				p := param.CategoricalHyperparameter
				mult = len(p.Vals)
			}
			gridTrials *= mult
		})
	}

	errs := []error{}

	// If the configuration is not a native submission, the user must specify an
	// entrypoint in the configuration.
	if e.Internal == nil || e.Internal.Native == nil {
		errs = append(errs, check.NotEmpty(
			e.Entrypoint, "Must specify an entrypoint that references the trial class."))
	}

	return append(errs, []error{
		check.TrueSilent(len(noCountParams) == 0,
			"these hyperparameters must specify counts for grid search: %s",
			strings.Join(noCountParams, ", ")),
		check.LessThanOrEqualTo(gridTrials, MaxAllowedTrials,
			"number of trials for grid search must be <= %d", MaxAllowedTrials),
		check.GreaterThanOrEqualTo(e.MaxRestarts, 0, "max_restarts must be >= 0"),
		check.True(e.MinCheckpointPeriod == nil,
			"min_checkpoint_period is deprecated, please use checkpoint_period"),
		check.True(e.MinValidationPeriod == nil,
			"min_validation_period is deprecated, please use validation_period"),
		check.GreaterThan(e.CheckpointPeriod.Units, 0, "checkpoint_period must be > 0"),
		check.GreaterThan(e.ValidationPeriod.Units, 0, "validation_period must be > 0"),
		check.Equal(e.ValidationPeriod.Unit, e.CheckpointPeriod.Unit,
			"checkpoint_period and validation_period must use the same unit"),
		check.Equal(e.CheckpointPeriod.Unit, e.Searcher.Unit(),
			"checkpoint_period and searcher must use the same unit"),
	}...)
}

// Value implements the driver.Valuer interface.
func (e ExperimentConfig) Value() (driver.Value, error) {
	if err := check.Validate(e); err != nil {
		return nil, err
	}
	return json.Marshal(e)
}

// Scan implements the db.Scanner interface.
func (e *ExperimentConfig) Scan(src interface{}) error {
	data, ok := src.([]byte)
	if !ok {
		return errors.Errorf("unable to convert to []byte: %v", src)
	}
	config := DefaultExperimentConfig()
	if err := json.Unmarshal(data, &config); err != nil {
		return err
	}
	*e = config
	return nil
}

// Unit implements the model.InUnits interface.
func (e ExperimentConfig) Unit() Unit {
	return e.ValidationPeriod.Unit
}

// InUnits is describes a type that is in terms of a specific unit.
type InUnits interface {
	Unit() Unit
}

// Labels holds the set of labels on the experiment.
type Labels map[string]bool

// MarshalJSON implements the json.Marshaler interface.
func (l Labels) MarshalJSON() ([]byte, error) {
	labels := make([]string, 0, len(l))
	for label := range l {
		labels = append(labels, label)
	}
	return json.Marshal(labels)
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (l *Labels) UnmarshalJSON(data []byte) error {
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

// ResourcesConfig configures experiment resource usage.
type ResourcesConfig struct {
	// Slots is used by commands while trials use SlotsPerTrial.
	Slots int `json:"slots,omitempty"`

	MaxSlots       *int    `json:"max_slots,omitempty"`
	SlotsPerTrial  int     `json:"slots_per_trial"`
	Weight         float64 `json:"weight"`
	NativeParallel bool    `json:"native_parallel"`
	ShmSize        *int    `json:"shm_size,omitempty"`
	AgentLabel     string  `json:"agent_label"`
}

// Validate implements the check.Validatable interface.
func (r ResourcesConfig) Validate() []error {
	return []error{
		check.GreaterThan(r.SlotsPerTrial, 0, "slots_per_trial must be > 0"),
		check.GreaterThan(r.Weight, float64(0), "weight must be > 0"),
		check.GreaterThanOrEqualTo(
			r.MaxSlots, r.SlotsPerTrial, "max_slots must be >= slots_per_trial"),
		check.GreaterThanOrEqualTo(r.ShmSize, 0, "shm_size must be >= 0"),
	}
}

// OptimizationsConfig configures performance optimizations for Horovod training.
type OptimizationsConfig struct {
	AggregationFrequency       int    `json:"aggregation_frequency"`
	AverageAggregatedGradients bool   `json:"average_aggregated_gradients"`
	AverageTrainingMetrics     bool   `json:"average_training_metrics"`
	GradientCompression        bool   `json:"gradient_compression"`
	GradUpdateSizeFile         string `json:"grad_updates_size_file,omitempty"`
	MixedPrecision             string `json:"mixed_precision"`
	TensorFusionThreshold      int    `json:"tensor_fusion_threshold"`
	TensorFusionCycleTime      int    `json:"tensor_fusion_cycle_time"`
	AutoTuneTensorFusion       bool   `json:"auto_tune_tensor_fusion"`
}

// Validate implements the check.Validatable interface.
func (r OptimizationsConfig) Validate() []error {
	return []error{
		check.GreaterThan(r.AggregationFrequency, 0, "aggregation_frequency must be > 0"),
		check.In(r.MixedPrecision, []string{"O0", "O1", "O2", "O3"}, "mixed_precision must be set "+
			"to one of the following  options: `O0`, `O1`, `O2`, `O3`. Note that in `O0`, `O1`, etc., "+
			"the prefix O is the capital letter O, not the number zero."),
		check.GreaterThanOrEqualTo(r.TensorFusionThreshold, 0, "tensor_fusion_threshold must be >= 0"),
		check.GreaterThanOrEqualTo(r.TensorFusionCycleTime, 0, "tensor_fusion_cycle_time must be >= 0"),
	}
}

// BindMount configures trial runner filesystem bind mounts.
type BindMount struct {
	HostPath      string `json:"host_path"`
	ContainerPath string `json:"container_path"`
	ReadOnly      bool   `json:"read_only"`
	Propagation   string `json:"propagation"`
}

// Validate implements the check.Validatable interface.
func (b BindMount) Validate() []error {
	return []error{
		check.True(b.ContainerPath != ".", "container_path must not be \".\""),
		check.True(filepath.IsAbs(b.HostPath), "host_path must be an absolute path"),
	}
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (b *BindMount) UnmarshalJSON(data []byte) error {
	b.Propagation = "rprivate"
	type DefaultParser *BindMount
	return errors.Wrap(json.Unmarshal(data, DefaultParser(b)), "failed to parse bind mounts")
}

// ReproducibilityConfig configures parameters related to reproducibility.
type ReproducibilityConfig struct {
	ExperimentSeed uint32 `json:"experiment_seed"`
}

// SecurityConfig configures the security options for the experiment. It is not used at this time.
// TODO(ryan): Remove this when we have an experiment config versioning solution (DET-164).
type SecurityConfig struct {
	Kerberos *KerberosConfig `json:"kerberos"`
}

// KerberosConfig configures Kerberos options for the experiment. It is not used anymore.
// TODO(ryan): Remove this when we have an experiment config versioning solution (DET-164).
type KerberosConfig struct {
	ConfigFile string `json:"config_file"`
}

// InternalConfig represents non-user-facing configuration set by Determined
// interface libraries.
type InternalConfig struct {
	Native *NativeConfig `json:"native"`
}

// NativeConfig represents configuration set by Determined native implementations.
type NativeConfig struct {
	Command []string `json:"command"`
}
