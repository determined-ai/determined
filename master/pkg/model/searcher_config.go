package model

import (
	"encoding/json"
	"fmt"

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
	return json.Unmarshal(data, DefaultParser(s))
}

// Kind defines a kind of unit for specifying lengths.
type Kind string

// All the kinds available for Lengths.
const (
	Records Kind = "records"
	Batches Kind = "batches"
	Epochs  Kind = "epoches"
)

// Length a training duration in terms of records, batches or epochs.
type Length struct {
	Kind  Kind
	Units int
}

// MarshalJSON implements the json.Marshaler interface.
func (l Length) MarshalJSON() ([]byte, error) {
	switch l.Kind {
	case Records:
		return json.Marshal(map[string]int{
			"records": l.Units,
		})
	case Batches:
		return json.Marshal(map[string]int{
			"batches": l.Units,
		})
	case Epochs:
		return json.Marshal(map[string]int{
			"epochs": l.Units,
		})
	default:
		panic(fmt.Sprintf("invalid unit passed to NewLength %s", l.Kind))
	}
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (l *Length) UnmarshalJSON(b []byte) error {
	var v map[string]int
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}

	records, rOk := v["records"]
	batches, bOk := v["batches"]
	epochs, eOk := v["epochs"]

	switch {
	case rOk && !bOk && !eOk:
		*l = NewLengthInRecords(records)
	case !rOk && bOk && !eOk:
		*l = NewLengthInBatches(batches)
	case !rOk && !bOk && eOk:
		*l = NewLengthInEpochs(epochs)
	default:
		return errors.New(fmt.Sprintf("invalid length: %s", b))
	}

	return nil
}

// NewLength returns a new length with the specified unit and length.
func NewLength(kind Kind, units int) Length {
	return Length{Kind: kind, Units: units}
}

// NewLengthInRecords returns a new length in terms of records.
func NewLengthInRecords(records int) Length {
	return Length{Kind: Records, Units: records}
}

// NewLengthInBatches returns a new length in terms of batches.
func NewLengthInBatches(batches int) Length {
	return Length{Kind: Batches, Units: batches}
}

// NewLengthInEpochs returns a new Length in terms of epochs.
func NewLengthInEpochs(epochs int) Length {
	return Length{Kind: Epochs, Units: epochs}
}

func (l Length) String() string {
	return fmt.Sprintf("%d %s", l.Units, l.Kind)
}

// Validate implements the check.Validatable interface.
func (l Length) Validate() []error {
	return []error{}
}

// Add adds a length to another length.
func (l Length) Add(other Length) Length {
	check.Panic(check.Equal(l.Kind, other.Kind))
	return NewLength(l.Kind, l.Units+other.Units)
}

// Sub subtracts a length from another length.
func (l Length) Sub(other Length) Length {
	check.Panic(check.Equal(l.Kind, other.Kind))
	return NewLength(l.Kind, l.Units-other.Units)
}

// Mult multiplies a length by another length.
func (l Length) Mult(other Length) Length {
	check.Panic(check.Equal(l.Kind, other.Kind))
	return NewLength(l.Kind, l.Units*other.Units)
}

// MultInt multiplies a length by an int.
func (l Length) MultInt(other int) Length {
	return NewLength(l.Kind, l.Units*other)
}

// Div divides a length by another length.
func (l Length) Div(other Length) Length {
	check.Panic(check.Equal(l.Kind, other.Kind))
	return NewLength(l.Kind, l.Units/other.Units)
}

// DivInt divides a length by an int.
func (l Length) DivInt(other int) Length {
	return NewLength(l.Kind, l.Units/other)
}

// SingleConfig configures a single trial.
type SingleConfig struct {
	MaxLength Length `json:"max_length"`
}

// Validate implements the check.Validatable interface.
func (s SingleConfig) Validate() []error {
	return []error{
		check.GreaterThan(s.MaxLength.Units, 0, "max_length must be > 0"),
	}
}

// RandomConfig configures a random search.
type RandomConfig struct {
	MaxLength Length `json:"max_length"`
	MaxTrials int    `json:"max_trials"`
}

// Validate implements the check.Validatable interface.
func (r RandomConfig) Validate() []error {
	return []error{
		check.GreaterThan(r.MaxLength.Units, 0, "max_length must be > 0"),
		check.GreaterThan(r.MaxTrials, 0, "max_trials must be > 0"),
	}
}

// GridConfig configures a grid search.
type GridConfig struct {
	MaxLength Length `json:"max_length"`
}

// Validate implements the check.Validatable interface.
func (g GridConfig) Validate() []error {
	return []error{
		check.GreaterThan(g.MaxLength.Units, 0, "max_length must be > 0"),
	}
}

// SyncHalvingConfig configures asynchronous successive halving.
type SyncHalvingConfig struct {
	Metric          string  `json:"metric"`
	SmallerIsBetter bool    `json:"smaller_is_better"`
	NumRungs        int     `json:"num_rungs"`
	MaxLength       Length  `json:"max_length"`
	Budget          Length  `json:"budget"`
	Divisor         float64 `json:"divisor"`
	TrainStragglers bool    `json:"train_stragglers"`
}

// Validate implements the check.Validatable interface.
func (c SyncHalvingConfig) Validate() []error {
	return []error{
		check.Equal(c.MaxLength.Kind, c.Budget.Kind,
			"max_length and budget must be specified in terms of the same unit"),
	}
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
}

// Validate implements the check.Validatable interface.
func (a AsyncHalvingConfig) Validate() []error {
	return []error{
		check.GreaterThan(a.MaxLength.Units, 0, "max_length must be > 0"),
		check.GreaterThan(a.MaxTrials, 0, "max_trials must be > 0"),
		check.GreaterThan(a.Divisor, 1.0, "divisor must be > 1.0"),
		check.GreaterThan(a.NumRungs, 0, "num_rungs must be > 0"),
		check.GreaterThanOrEqualTo(a.MaxConcurrentTrials, 0, "max_concurrent_trials must be >= 0"),
	}
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
			"step_budget must be > target_trial_steps"),
		check.GreaterThan(a.MaxLength.Units, 0, "target_trial_steps must be > 0"),
		check.GreaterThan(a.Budget.Units, 0, "step_budget must be > 0"),
		check.LessThanOrEqualTo(a.Budget.Units, 50000, "step_budget must be <= 50000"),
		check.GreaterThan(a.Divisor, 1.0, "divisor must be > 1.0"),
		check.In(string(a.Mode), []string{AggressiveMode, StandardMode, ConservativeMode},
			"invalid adaptive mode"),
		check.GreaterThan(a.MaxRungs, 0, "max_rungs must be > 0"),
		check.Equal(a.MaxLength.Units, a.Budget.Units,
			"max_length and budget must be specified in terms of the same unit"),
	}
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
