package model

import (
	"encoding/json"

	"github.com/pkg/errors"

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
	SourceTrialID        *int    `json:"source_trial_id"`
	SourceCheckpointUUID *string `json:"source_checkpoint_uuid"`

	SingleConfig         *SingleConfig         `union:"name,single" json:"-"`
	RandomConfig         *RandomConfig         `union:"name,random" json:"-"`
	GridConfig           *GridConfig           `union:"name,grid" json:"-"`
	SyncHalvingConfig    *SyncHalvingConfig    `union:"name,sync_halving" json:"-"`
	AsyncHalvingConfig   *AsyncHalvingConfig   `union:"name,async_halving" json:"-"`
	AdaptiveConfig       *AdaptiveConfig       `union:"name,adaptive" json:"-"`
	AdaptiveSimpleConfig *AdaptiveSimpleConfig `union:"name,adaptive_simple" json:"-"`
	AdaptiveASHAConfig   *AdaptiveASHAConfig   `union:"name,adaptive_asha" json:"-"`
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
	return errors.Wrap(json.Unmarshal(data, DefaultParser(s)), "failed to parse searcher config")
}

// Unit implements the model.InUnits interface.
func (s SearcherConfig) Unit() Unit {
	switch {
	case s.SingleConfig != nil:
		return s.SingleConfig.Unit()
	case s.RandomConfig != nil:
		return s.RandomConfig.Unit()
	case s.GridConfig != nil:
		return s.GridConfig.Unit()
	case s.SyncHalvingConfig != nil:
		return s.SyncHalvingConfig.Unit()
	case s.AdaptiveConfig != nil:
		return s.AdaptiveConfig.Unit()
	case s.AdaptiveSimpleConfig != nil:
		return s.AdaptiveSimpleConfig.Unit()
	case s.AsyncHalvingConfig != nil:
		return s.AsyncHalvingConfig.Unit()
	case s.AdaptiveASHAConfig != nil:
		return s.AdaptiveASHAConfig.Unit()
	case s.PBTConfig != nil:
		return s.PBTConfig.Unit()
	default:
		panic("no searcher type specified")
	}
}

// SingleConfig configures a single trial.
type SingleConfig struct {
	MaxLength Length `json:"max_length"`
}

// Validate implements the check.Validatable interface.
func (s SingleConfig) Validate() (errs []error) {
	return []error{
		check.GreaterThan(s.MaxLength.Units, 0, "max_length must be > 0"),
	}
}

// Unit implements the model.InUnits interface.
func (s SingleConfig) Unit() Unit {
	return s.MaxLength.Unit
}

// RandomConfig configures a random search.
type RandomConfig struct {
	MaxLength           Length `json:"max_length"`
	MaxTrials           int    `json:"max_trials"`
	MaxConcurrentTrials int    `json:"max_concurrent_trials"`
}

// Unit implements the model.InUnits interface.
func (r RandomConfig) Unit() Unit {
	return r.MaxLength.Unit
}

// Validate implements the check.Validatable interface.
func (r RandomConfig) Validate() (errs []error) {
	return []error{
		check.GreaterThan(r.MaxLength.Units, 0, "max_length must be > 0"),
		check.GreaterThan(r.MaxTrials, 0, "max_trials must be > 0"),
		check.GreaterThanOrEqualTo(r.MaxConcurrentTrials, 0, "max_concurrent_trials must be >= 0"),
	}
}

// GridConfig configures a grid search.
type GridConfig struct {
	MaxLength           Length `json:"max_length"`
	MaxConcurrentTrials int    `json:"max_concurrent_trials"`
}

// Unit implements the model.InUnits interface.
func (g GridConfig) Unit() Unit {
	return g.MaxLength.Unit
}

// Validate implements the check.Validatable interface.
func (g GridConfig) Validate() (errs []error) {
	return []error{
		check.GreaterThan(g.MaxLength.Units, 0, "max_length must be > 0"),
		check.GreaterThanOrEqualTo(g.MaxConcurrentTrials, 0, "max_concurrent_trials must be >= 0"),
	}
}

// SyncHalvingConfig configures synchronous successive halving.
type SyncHalvingConfig struct {
	Metric          string  `json:"metric"`
	SmallerIsBetter bool    `json:"smaller_is_better"`
	NumRungs        int     `json:"num_rungs"`
	MaxLength       Length  `json:"max_length"`
	Budget          Length  `json:"budget"`
	Divisor         float64 `json:"divisor"`
	TrainStragglers bool    `json:"train_stragglers"`
}

// Unit implements the model.InUnits interface.
func (s SyncHalvingConfig) Unit() Unit {
	return s.MaxLength.Unit
}

// AsyncHalvingConfig configures asynchronous successive halving.
type AsyncHalvingConfig struct {
	Metric              string  `json:"metric"`
	SmallerIsBetter     bool    `json:"smaller_is_better"`
	NumRungs            int     `json:"num_rungs"`
	MaxLength           Length  `json:"max_length"`
	MaxTrials           int     `json:"max_trials"`
	Divisor             float64 `json:"divisor"`
	MaxConcurrentTrials int     `json:"max_concurrent_trials"`
	StopOnce            bool    `json:"stop_once"`
}

// Validate implements the check.Validatable interface.
func (a AsyncHalvingConfig) Validate() (errs []error) {
	return []error{
		check.GreaterThan(a.MaxLength.Units, 0, "max_length must be > 0"),
		check.GreaterThan(a.MaxTrials, 0, "max_trials must be > 0"),
		check.GreaterThan(a.Divisor, 1.0, "divisor must be > 1.0"),
		check.GreaterThan(a.NumRungs, 0, "num_rungs must be > 0"),
		check.GreaterThanOrEqualTo(a.MaxConcurrentTrials, 0, "max_concurrent_trials must be >= 0"),
	}
}

// Unit implements the model.InUnits interface.
func (a AsyncHalvingConfig) Unit() Unit {
	return a.MaxLength.Unit
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
	Metric          string       `json:"metric"`
	SmallerIsBetter bool         `json:"smaller_is_better"`
	MaxLength       Length       `json:"max_length"`
	Budget          Length       `json:"budget"`
	BracketRungs    []int        `json:"bracket_rungs"`
	Divisor         float64      `json:"divisor"`
	TrainStragglers bool         `json:"train_stragglers"`
	Mode            AdaptiveMode `json:"mode"`
	MaxRungs        int          `json:"max_rungs"`
}

// Validate implements the check.Validatable interface.
func (a AdaptiveConfig) Validate() []error {
	return []error{
		check.GreaterThan(a.Budget.Units, a.MaxLength.Units,
			"budget must be > max_length"),
		check.GreaterThan(a.MaxLength.Units, 0, "max_length must be > 0"),
		check.GreaterThan(a.Budget.Units, 0, "budget must be > 0"),
		check.GreaterThan(a.Divisor, 1.0, "divisor must be > 1.0"),
		check.In(string(a.Mode), []string{AggressiveMode, StandardMode, ConservativeMode},
			"invalid adaptive mode"),
		check.GreaterThan(a.MaxRungs, 0, "max_rungs must be > 0"),
		check.Equal(a.MaxLength.Unit, a.Budget.Unit,
			"max_length and budget must be specified in terms of the same unit"),
	}
}

// Unit implements the model.InUnits interface.
func (a AdaptiveConfig) Unit() Unit {
	return a.MaxLength.Unit
}

// AdaptiveSimpleConfig configures an simplified adaptive search.
type AdaptiveSimpleConfig struct {
	Metric          string       `json:"metric"`
	SmallerIsBetter bool         `json:"smaller_is_better"`
	MaxLength       Length       `json:"max_length"`
	MaxTrials       int          `json:"max_trials"`
	Divisor         float64      `json:"divisor"`
	Mode            AdaptiveMode `json:"mode"`
	MaxRungs        int          `json:"max_rungs"`
}

// Validate implements the check.Validatable interface.
func (a AdaptiveSimpleConfig) Validate() []error {
	return []error{
		check.GreaterThan(a.MaxLength.Units, 0, "max_length must be > 0"),
		check.GreaterThan(a.MaxTrials, 0, "max_trials must be > 0"),
		check.LessThanOrEqualTo(a.MaxTrials, MaxAllowedTrials,
			"max_trials must be <= %d", MaxAllowedTrials),
		check.GreaterThan(a.Divisor, 1.0, "divisor must be > 1.0"),
		check.In(string(a.Mode), []string{AggressiveMode, StandardMode, ConservativeMode},
			"invalid adaptive mode"),
		check.GreaterThan(a.MaxRungs, 0, "max_rungs must be > 0"),
	}
}

// Unit implements the model.InUnits interface.
func (a AdaptiveSimpleConfig) Unit() Unit {
	return a.MaxLength.Unit
}

// AdaptiveASHAConfig configures an adaptive searcher for use with ASHA.
type AdaptiveASHAConfig struct {
	Metric              string       `json:"metric"`
	SmallerIsBetter     bool         `json:"smaller_is_better"`
	MaxLength           Length       `json:"max_length"`
	MaxTrials           int          `json:"max_trials"`
	BracketRungs        []int        `json:"bracket_rungs"`
	Divisor             float64      `json:"divisor"`
	Mode                AdaptiveMode `json:"mode"`
	MaxRungs            int          `json:"max_rungs"`
	MaxConcurrentTrials int          `json:"max_concurrent_trials"`
	StopOnce            bool         `json:"stop_once"`
}

// Validate implements the check.Validatable interface.
func (a AdaptiveASHAConfig) Validate() []error {
	return []error{
		check.GreaterThan(a.MaxLength.Units, 0, "max_length must be > 0"),
		check.GreaterThan(a.MaxTrials, 0, "max_trials must be > 0"),
		check.GreaterThan(a.Divisor, 1.0, "divisor must be > 1.0"),
		check.In(string(a.Mode), []string{AggressiveMode, StandardMode, ConservativeMode},
			"invalid adaptive mode"),
		check.GreaterThan(a.MaxRungs, 0, "max_rungs must be > 0"),
		check.GreaterThanOrEqualTo(a.MaxConcurrentTrials, 0, "max_concurrent_trials must be >= 0"),
	}
}

// Unit implements the model.InUnits interface.
func (a AdaptiveASHAConfig) Unit() Unit {
	return a.MaxLength.Unit
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
	LengthPerRound  Length `json:"length_per_round"`

	PBTReplaceConfig `json:"replace_function"`
	PBTExploreConfig `json:"explore_function"`
}

// Validate implements the check.Validatable interface.
func (p PBTConfig) Validate() []error {
	return []error{
		check.GreaterThan(p.PopulationSize, 0, "population_size must be > 0"),
		check.GreaterThan(p.NumRounds, 0, "num_rounds must be > 0"),
		check.GreaterThan(p.LengthPerRound.Units, 0, "length_per_round must be > 0"),
	}
}

// Unit implements the model.InUnits interface.
func (p PBTConfig) Unit() Unit {
	return p.LengthPerRound.Unit
}
