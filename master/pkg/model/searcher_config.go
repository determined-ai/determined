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
	return json.Unmarshal(data, DefaultParser(s))
}

// Shim converts a config to the new version.
func (s SearcherConfig) Shim(batchesPerStep int) SearcherConfig {
	switch {
	case s.SingleConfig != nil:
		s.SingleConfig = s.SingleConfig.shim(batchesPerStep)
	case s.RandomConfig != nil:
		s.RandomConfig = s.RandomConfig.shim(batchesPerStep)
	case s.GridConfig != nil:
		s.GridConfig = s.GridConfig.shim(batchesPerStep)
	case s.SyncHalvingConfig != nil:
		s.SyncHalvingConfig = s.SyncHalvingConfig.shim(batchesPerStep)
	case s.AdaptiveConfig != nil:
		s.AdaptiveConfig = s.AdaptiveConfig.shim(batchesPerStep)
	case s.AdaptiveSimpleConfig != nil:
		s.AdaptiveSimpleConfig = s.AdaptiveSimpleConfig.shim(batchesPerStep)
	case s.AsyncHalvingConfig != nil:
		s.AsyncHalvingConfig = s.AsyncHalvingConfig.shim(batchesPerStep)
	case s.AdaptiveASHAConfig != nil:
		s.AdaptiveASHAConfig = s.AdaptiveASHAConfig.shim(batchesPerStep)
	case s.PBTConfig != nil:
		s.PBTConfig = s.PBTConfig.shim(batchesPerStep)
	default:
		panic("no searcher type specified")
	}
	return s
}

// SingleConfig configures a single trial.
type SingleConfig struct {
	MaxLength Length `json:"max_length"`

	// Deprecated
	MaxSteps int `json:"max_steps"`
}

// Validate implements the check.Validatable interface.
func (s SingleConfig) Validate() (errs []error) {
	return validate(s)
}

func (s SingleConfig) validateDeprecated() []error {
	return []error{
		check.GreaterThan(s.MaxSteps, 0, "max_steps must be > 0"),
	}
}

func (s SingleConfig) validateNew() []error {
	return []error{
		check.GreaterThan(s.MaxLength.Units, 0, "max_length must be > 0"),
	}
}

func (s SingleConfig) shim(batchesPerStep int) *SingleConfig {
	if s.MaxSteps > 0 {
		s.MaxLength = NewLengthInBatches(s.MaxSteps * batchesPerStep)
	}
	return &s
}

// RandomConfig configures a random search.
type RandomConfig struct {
	MaxLength Length `json:"max_length"`
	MaxTrials int    `json:"max_trials"`

	// Deprecated
	MaxSteps int `json:"max_steps"`
}

// Validate implements the check.Validatable interface.
func (r RandomConfig) Validate() (errs []error) {
	return validate(r)
}

func (r RandomConfig) validateDeprecated() []error {
	return []error{
		check.GreaterThan(r.MaxSteps, 0, "max_steps must be > 0"),
		check.GreaterThan(r.MaxTrials, 0, "max_trials must be > 0"),
	}
}

func (r RandomConfig) validateNew() []error {
	return []error{
		check.GreaterThan(r.MaxSteps, 0, "max_steps must be > 0"),
		check.GreaterThan(r.MaxTrials, 0, "max_trials must be > 0"),
	}
}

func (r RandomConfig) shim(batchesPerStep int) *RandomConfig {
	if r.MaxSteps > 0 {
		r.MaxLength = NewLengthInBatches(r.MaxSteps * batchesPerStep)
	}
	return &r
}

// GridConfig configures a grid search.
type GridConfig struct {
	MaxLength Length `json:"max_length"`

	// Deprecated
	MaxSteps int `json:"max_steps"`
}

// Validate implements the check.Validatable interface.
func (g GridConfig) Validate() (errs []error) {
	return validate(g)
}

func (g GridConfig) validateDeprecated() []error {
	return []error{
		check.GreaterThan(g.MaxSteps, 0, "max_steps must be > 0"),
	}
}

func (g GridConfig) validateNew() []error {
	return []error{
		check.GreaterThan(g.MaxLength.Units, 0, "max_length must be > 0"),
	}
}

func (g GridConfig) shim(batchesPerStep int) *GridConfig {
	if g.MaxSteps > 0 {
		g.MaxLength = NewLengthInBatches(g.MaxSteps * batchesPerStep)
	}
	return &g
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

	// Deprecated
	TargetTrialSteps int `json:"target_trial_steps"`
	StepBudget       int `json:"step_budget"`
}

func (s SyncHalvingConfig) shim(batchesPerStep int) *SyncHalvingConfig {
	if s.TargetTrialSteps > 0 {
		s.MaxLength = NewLengthInBatches(s.TargetTrialSteps * batchesPerStep)
		s.Budget = NewLengthInBatches(s.StepBudget * batchesPerStep)
	}
	return &s
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

	// Deprecated
	TargetTrialSteps int `json:"target_trial_steps"`
}

// Validate implements the check.Validatable interface.
func (a AsyncHalvingConfig) Validate() (errs []error) {
	return validate(a)
}

func (a AsyncHalvingConfig) validateDeprecated() []error {
	return []error{
		check.GreaterThan(a.TargetTrialSteps, 0, "target_trial_steps must be > 0"),
		check.GreaterThan(a.MaxTrials, 0, "max_trials must be > 0"),
		check.GreaterThan(a.Divisor, 1.0, "divisor must be > 1.0"),
		check.GreaterThan(a.NumRungs, 0, "num_rungs must be > 0"),
		check.GreaterThanOrEqualTo(a.MaxConcurrentTrials, 0, "max_concurrent_trials must be >= 0"),
	}
}

func (a AsyncHalvingConfig) validateNew() []error {
	return []error{
		check.GreaterThan(a.MaxLength.Units, 0, "max_length must be > 0"),
		check.GreaterThan(a.MaxTrials, 0, "max_trials must be > 0"),
		check.GreaterThan(a.Divisor, 1.0, "divisor must be > 1.0"),
		check.GreaterThan(a.NumRungs, 0, "num_rungs must be > 0"),
		check.GreaterThanOrEqualTo(a.MaxConcurrentTrials, 0, "max_concurrent_trials must be >= 0"),
	}
}

func (a AsyncHalvingConfig) shim(batchesPerStep int) *AsyncHalvingConfig {
	if a.TargetTrialSteps > 0 {
		a.MaxLength = NewLengthInBatches(a.TargetTrialSteps * batchesPerStep)
	}
	return &a
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

	// Deprecated
	TargetTrialSteps int `json:"target_trial_steps"`
	StepBudget       int `json:"step_budget"`
}

// Validate implements the check.Validatable interface.
func (a AdaptiveConfig) Validate() (errs []error) {
	return validate(a)
}

func (a AdaptiveConfig) validateDeprecated() []error {
	return []error{
		check.GreaterThan(a.StepBudget, a.TargetTrialSteps,
			"step_budget must be > target_trial_steps"),
		check.GreaterThan(a.TargetTrialSteps, 0, "target_trial_steps must be > 0"),
		check.GreaterThan(a.StepBudget, 0, "step_budget must be > 0"),
		check.LessThanOrEqualTo(a.StepBudget, 50000, "step_budget must be <= 50000"),
		check.GreaterThan(a.Divisor, 1.0, "divisor must be > 1.0"),
		check.In(string(a.Mode), []string{AggressiveMode, StandardMode, ConservativeMode},
			"invalid adaptive mode"),
		check.GreaterThan(a.MaxRungs, 0, "max_rungs must be > 0"),
	}
}

func (a AdaptiveConfig) validateNew() []error {
	return []error{
		check.GreaterThan(a.Budget.Units, a.MaxLength.Units,
			"budget must be > max_length"),
		check.GreaterThan(a.MaxLength.Units, 0, "max_length must be > 0"),
		check.GreaterThan(a.Budget.Units, 0, "budget must be > 0"),
		check.GreaterThan(a.Divisor, 1.0, "divisor must be > 1.0"),
		check.In(string(a.Mode), []string{AggressiveMode, StandardMode, ConservativeMode},
			"invalid adaptive mode"),
		check.GreaterThan(a.MaxRungs, 0, "max_rungs must be > 0"),
		check.Equal(a.MaxLength.Units, a.Budget.Units,
			"max_length and budget must be specified in terms of the same unit"),
	}
}

func (a AdaptiveConfig) shim(batchesPerStep int) *AdaptiveConfig {
	if a.TargetTrialSteps > 0 {
		a.MaxLength = NewLengthInBatches(a.TargetTrialSteps * batchesPerStep)
		a.Budget = NewLengthInBatches(a.StepBudget * batchesPerStep)
	}
	return &a
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

	// Deprecated
	MaxSteps int `json:"max_steps"`
}

// Validate implements the check.Validatable interface.
func (a AdaptiveSimpleConfig) Validate() (errs []error) {
	return validate(a)
}

func (a AdaptiveSimpleConfig) validateDeprecated() []error {
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

func (a AdaptiveSimpleConfig) validateNew() []error {
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

func (a AdaptiveSimpleConfig) shim(batchesPerStep int) *AdaptiveSimpleConfig {
	if a.MaxSteps > 0 {
		a.MaxLength = NewLengthInBatches(a.MaxSteps * batchesPerStep)
	}
	return &a
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

	// Deprecated
	TargetTrialSteps int `json:"target_trial_steps"`
}

// Validate implements the check.Validatable interface.
func (a AdaptiveASHAConfig) Validate() (errs []error) {
	return validate(a)
}

func (a AdaptiveASHAConfig) validateDeprecated() []error {
	return []error{
		check.GreaterThan(a.TargetTrialSteps, 0, "target_trial_steps must be > 0"),
		check.GreaterThan(a.MaxTrials, 0, "max_trials must be > 0"),
		check.GreaterThan(a.Divisor, 1.0, "divisor must be > 1.0"),
		check.In(string(a.Mode), []string{AggressiveMode, StandardMode, ConservativeMode},
			"invalid adaptive mode"),
		check.GreaterThan(a.MaxRungs, 0, "max_rungs must be > 0"),
		check.GreaterThanOrEqualTo(a.MaxConcurrentTrials, 0, "max_concurrent_trials must be >= 0"),
	}
}

func (a AdaptiveASHAConfig) validateNew() []error {
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

func (a AdaptiveASHAConfig) shim(batchesPerStep int) *AdaptiveASHAConfig {
	if a.TargetTrialSteps > 0 {
		a.MaxLength = NewLengthInBatches(a.TargetTrialSteps * batchesPerStep)
	}
	return &a
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

	// Deprecated
	StepsPerRound int `json:"steps_per_round"`
}

// Validate implements the check.Validatable interface.
func (p PBTConfig) Validate() (errs []error) {
	return validate(p)
}

// Validate implements the check.Validatable interface.
func (p PBTConfig) validateDeprecated() []error {
	return []error{
		check.GreaterThan(p.PopulationSize, 0, "population_size must be > 0"),
		check.GreaterThan(p.NumRounds, 0, "num_rounds must be > 0"),
		check.GreaterThan(p.StepsPerRound, 0, "steps_per_round must be > 0"),
	}
}

// Validate implements the check.Validatable interface.
func (p PBTConfig) validateNew() []error {
	return []error{
		check.GreaterThan(p.PopulationSize, 0, "population_size must be > 0"),
		check.GreaterThan(p.NumRounds, 0, "num_rounds must be > 0"),
		check.GreaterThan(p.LengthPerRound.Units, 0, "length_per_round must be > 0"),
	}
}

func (p PBTConfig) shim(batchesPerStep int) *PBTConfig {
	if p.StepsPerRound > 0 {
		p.LengthPerRound = NewLengthInBatches(p.StepsPerRound * batchesPerStep)
	}
	return &p
}

type validatableTwoVersionConfig interface {
	validateDeprecated() []error
	validateNew() []error
}

func validate(config validatableTwoVersionConfig) (errs []error) {
	dErrs := config.validateDeprecated()
	nErrs := config.validateNew()

	if allNil(dErrs...) && allNil(nErrs...) {
		errs = append(errs, errors.New("multiple configurations specified"))
	}

	if !allNil(dErrs...) && !allNil(nErrs...) {
		errs = append(errs, nErrs...)
	}

	return errs
}

func allNil(errs ...error) bool {
	for _, err := range errs {
		if err != nil {
			return false
		}
	}
	return true
}
