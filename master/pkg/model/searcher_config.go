package model

import (
	"encoding/json"

	"github.com/determined-ai/determined/master/pkg/check"
	"github.com/determined-ai/determined/master/pkg/union"
)

// MaxAllowedTrials is the maximum number of trials that we allow to be created for a single
// experiment. The limitation is not fundamental, but we start running into issues with performance,
// memory, and crashes at some point, and this is a defense against that sort of thing. Currently,
// the limit is only enforced for grid and simple adaptive searchers.
const MaxAllowedTrials = 2000

// SearcherConfig holds the searcher configurations.
type SearcherConfig struct {
	Metric               string  `json:"metric"`
	SmallerIsBetter      bool    `json:"smaller_is_better"`
	MaxConcurrentTrials  *int    `json:"max_concurrent_trials"`
	SourceTrialID        *int    `json:"source_trial_id"`
	SourceCheckpointUUID *string `json:"source_checkpoint_uuid"`

	SingleConfig         *SingleConfig         `union:"name,single" json:"-"`
	RandomConfig         *RandomConfig         `union:"name,random" json:"-"`
	GridConfig           *GridConfig           `union:"name,grid" json:"-"`
	AsyncHalvingConfig   *AsyncHalvingConfig   `union:"name,async_halving" json:"-"`
	SyncHalvingConfig    *SyncHalvingConfig    `union:"name,sync_halving" json:"-"`
	AdaptiveConfig       *AdaptiveConfig       `union:"name,adaptive" json:"-"`
	AdaptiveSimpleConfig *AdaptiveSimpleConfig `union:"name,adaptive_simple" json:"-"`
	PBTConfig            *PBTConfig            `union:"name,pbt" json:"-"`
}

// MarshalJSON implements the json.Marshaler interface.
func (s SearcherConfig) MarshalJSON() ([]byte, error) {
	return union.Marshal(s)
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (s *SearcherConfig) UnmarshalJSON(data []byte) error {
	if err := union.Unmarshal(data, s); err != nil {
		return err
	}
	type DefaultParser *SearcherConfig
	return json.Unmarshal(data, DefaultParser(s))
}

// SingleConfig configures a single trial.
type SingleConfig struct {
	MaxSteps int `json:"max_steps"`
}

// Validate implements the check.Validatable interface.
func (s SingleConfig) Validate() []error {
	return []error{
		check.GreaterThan(s.MaxSteps, 0, "max_steps must be > 0"),
	}
}

// RandomConfig configures a random search.
type RandomConfig struct {
	MaxSteps  int `json:"max_steps"`
	MaxTrials int `json:"max_trials"`
}

// Validate implements the check.Validatable interface.
func (r RandomConfig) Validate() []error {
	return []error{
		check.GreaterThan(r.MaxSteps, 0, "max_steps must be > 0"),
		check.GreaterThan(r.MaxTrials, 0, "max_trials must be > 0"),
	}
}

// GridConfig configures a grid search.
type GridConfig struct {
	MaxSteps int `json:"max_steps"`
}

// Validate implements the check.Validatable interface.
func (g GridConfig) Validate() []error {
	return []error{
		check.GreaterThan(g.MaxSteps, 0, "max_steps must be > 0"),
	}
}

// AsyncHalvingConfig configures asynchronous successive halving.
type AsyncHalvingConfig struct {
	Metric              string  `json:"metric"`
	SmallerIsBetter     bool    `json:"smaller_is_better"`
	NumRungs            int     `json:"num_rungs"`
	TargetTrialSteps    int     `json:"target_trial_steps"`
	MaxTrials           int     `json:"max_trials"`
	Divisor             float64 `json:"divisor"`
	MaxConcurrentTrials *int    `json:"max_concurrent_trials"`
	TrainStragglers     bool    `json:"train_stragglers"`
}

// SyncHalvingConfig configures synchronous successive halving.
type SyncHalvingConfig struct {
	Metric           string  `json:"metric"`
	SmallerIsBetter  bool    `json:"smaller_is_better"`
	NumRungs         int     `json:"num_rungs"`
	TargetTrialSteps int     `json:"target_trial_steps"`
	StepBudget       int     `json:"step_budget"`
	Divisor          float64 `json:"divisor"`
	TrainStragglers  bool    `json:"train_stragglers"`
}

// AdaptiveMode specifies how aggressively to perform early stopping.
type AdaptiveMode string

const (
	// AggressiveMode quickly stops underperforming trials, which enables the searcher to explore
	// more hyperparameter configurations.
	AggressiveMode = "aggressive"
	// StandardMode provides a balance between downsampling and hyperparameter exploration.
	StandardMode = "standard"
	// ConservativeMode performs minimal downsampling at the cost of not exploring as many
	// configurations.
	ConservativeMode = "conservative"
)

// AdaptiveConfig configures an adaptive search.
type AdaptiveConfig struct {
	Metric              string       `json:"metric"`
	SmallerIsBetter     bool         `json:"smaller_is_better"`
	TargetTrialSteps    int          `json:"target_trial_steps"`
	MaxTrials           int          `json:"max_trials"`
	BracketRungs        []int        `json:"bracket_rungs"`
	Divisor             float64      `json:"divisor"`
	Mode                AdaptiveMode `json:"mode"`
	MaxRungs            int          `json:"max_rungs"`
	MaxConcurrentTrials *int         `json:"max_concurrent_trials"`
}

// Validate implements the check.Validatable interface.
func (a AdaptiveConfig) Validate() []error {
	return []error{
		check.GreaterThan(a.TargetTrialSteps, 0, "target_trial_steps must be > 0"),
		check.GreaterThan(a.MaxTrials, 0, "max_trials must be > 0"),
		check.GreaterThan(a.Divisor, 1.0, "divisor must be > 1.0"),
		check.In(string(a.Mode), []string{AggressiveMode, StandardMode, ConservativeMode},
			"invalid adaptive mode"),
		check.GreaterThan(a.MaxRungs, 0, "max_rungs must be > 0"),
	}
}

// AdaptiveSimpleConfig configures an simplified adaptive search.
type AdaptiveSimpleConfig struct {
	Metric              string       `json:"metric"`
	SmallerIsBetter     bool         `json:"smaller_is_better"`
	MaxSteps            int          `json:"max_steps"`
	MaxTrials           int          `json:"max_trials"`
	Divisor             float64      `json:"divisor"`
	Mode                AdaptiveMode `json:"mode"`
	MaxRungs            int          `json:"max_rungs"`
	MaxConcurrentTrials *int         `json:"max_concurrent_trials"`
}

// Validate implements the check.Validatable interface.
func (a AdaptiveSimpleConfig) Validate() []error {
	return []error{
		check.GreaterThan(a.MaxSteps, 0, "max_steps must be > 0"),
		check.GreaterThan(a.MaxTrials, 0, "max_trials must be > 0"),
		check.LessThanOrEqualTo(a.MaxTrials, MaxAllowedTrials,
			"max_trials must be <= %d", MaxAllowedTrials),
		check.GreaterThan(a.Divisor, 1.0, "divisor must be > 1.0"),
		check.In(string(a.Mode), []string{AggressiveMode, StandardMode, ConservativeMode},
			"invalid adaptive mode"),
		check.GreaterThan(a.MaxRungs, 0, "max_rungs must be > 0"),
	}
}

// PBTReplaceConfig configures replacement for a PBT search.
type PBTReplaceConfig struct {
	TruncateFraction float64 `json:"truncate_fraction"`
}

// Validate implements the check.Validatable interface.
func (r PBTReplaceConfig) Validate() []error {
	return []error{
		check.GreaterThanOrEqualTo(r.TruncateFraction, 0.0, "truncate_fraction must be >= 0"),
		check.LessThanOrEqualTo(r.TruncateFraction, 0.5, "truncate_fraction must be <= 0.5"),
	}
}

// PBTExploreConfig configures exploration for a PBT search.
type PBTExploreConfig struct {
	ResampleProbability float64 `json:"resample_probability"`
	PerturbFactor       float64 `json:"perturb_factor"`
}

// Validate implements the check.Validatable interface.
func (e PBTExploreConfig) Validate() []error {
	return []error{
		check.GreaterThanOrEqualTo(e.ResampleProbability, 0.0, "resample_probability must be >= 0"),
		check.LessThanOrEqualTo(e.ResampleProbability, 1.0, "resample_probability must be <= 1"),
		check.GreaterThanOrEqualTo(e.PerturbFactor, 0.0, "perturb_factor must be >= 0"),
		check.LessThanOrEqualTo(e.PerturbFactor, 1.0, "perturb_factor must be <= 1"),
	}
}

// PBTConfig configures a PBT search.
type PBTConfig struct {
	Metric          string `json:"metric"`
	SmallerIsBetter bool   `json:"smaller_is_better"`
	PopulationSize  int    `json:"population_size"`
	NumRounds       int    `json:"num_rounds"`
	StepsPerRound   int    `json:"steps_per_round"`

	PBTReplaceConfig `json:"replace_function"`
	PBTExploreConfig `json:"explore_function"`
}

// Validate implements the check.Validatable interface.
func (p PBTConfig) Validate() []error {
	return []error{
		check.GreaterThan(p.PopulationSize, 0, "population_size must be > 0"),
		check.GreaterThan(p.NumRounds, 0, "num_rounds must be > 0"),
		check.GreaterThan(p.StepsPerRound, 0, "steps_per_round must be > 0"),
	}
}
