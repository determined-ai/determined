package expconf

import (
	"encoding/json"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/union"
)

//go:generate ../gen.sh
// SearcherConfigV0 holds the searcher configurations.
type SearcherConfigV0 struct {
	SingleConfig       *SingleConfigV0       `union:"name,single" json:"-"`
	RandomConfig       *RandomConfigV0       `union:"name,random" json:"-"`
	GridConfig         *GridConfigV0         `union:"name,grid" json:"-"`
	AsyncHalvingConfig *AsyncHalvingConfigV0 `union:"name,async_halving" json:"-"`
	AdaptiveASHAConfig *AdaptiveASHAConfigV0 `union:"name,adaptive_asha" json:"-"`
	PBTConfig          *PBTConfigV0          `union:"name,pbt" json:"-"`
}

// MarshalJSON implements the json.Marshaler interface.
func (s SearcherConfigV0) MarshalJSON() ([]byte, error) {
	return union.MarshalEx(s, true)
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (s *SearcherConfigV0) UnmarshalJSON(data []byte) error {
	if err := union.Unmarshal(data, s); err != nil {
		return err
	}
	type DefaultParser *SearcherConfigV0
	return errors.Wrap(json.Unmarshal(data, DefaultParser(s)), "failed to parse searcher config")
}

// Unit implements the model.InUnits interface.
func (s SearcherConfigV0) Unit() Unit {
	switch {
	case s.SingleConfig != nil:
		return s.SingleConfig.Unit()
	case s.RandomConfig != nil:
		return s.RandomConfig.Unit()
	case s.GridConfig != nil:
		return s.GridConfig.Unit()
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

//go:generate ../gen.sh
// SingleConfigV0 configures a single trial.
type SingleConfigV0 struct {
	Metric               string  `json:"metric"`
	SmallerIsBetter      *bool   `json:"smaller_is_better"`
	SourceTrialID        *int    `json:"source_trial_id"`
	SourceCheckpointUUID *string `json:"source_checkpoint_uuid"`

	MaxLength LengthV0 `json:"max_length"`
}

// Unit implements the model.InUnits interface.
func (s SingleConfigV0) Unit() Unit {
	return s.MaxLength.Unit
}

//go:generate ../gen.sh
// RandomConfigV0 configures a random search.
type RandomConfigV0 struct {
	Metric               string  `json:"metric"`
	SmallerIsBetter      *bool   `json:"smaller_is_better"`
	SourceTrialID        *int    `json:"source_trial_id"`
	SourceCheckpointUUID *string `json:"source_checkpoint_uuid"`

	MaxLength           LengthV0 `json:"max_length"`
	MaxTrials           int      `json:"max_trials"`
	MaxConcurrentTrials *int     `json:"max_concurrent_trials"`
}

// Unit implements the model.InUnits interface.
func (r RandomConfigV0) Unit() Unit {
	return r.MaxLength.Unit
}

//go:generate ../gen.sh
// GridConfigV0 configures a grid search.
type GridConfigV0 struct {
	Metric               string  `json:"metric"`
	SmallerIsBetter      *bool   `json:"smaller_is_better"`
	SourceTrialID        *int    `json:"source_trial_id"`
	SourceCheckpointUUID *string `json:"source_checkpoint_uuid"`

	MaxLength           LengthV0 `json:"max_length"`
	MaxConcurrentTrials *int     `json:"max_concurrent_trials"`
}

// Unit implements the model.InUnits interface.
func (g GridConfigV0) Unit() Unit {
	return g.MaxLength.Unit
}

//go:generate ../gen.sh
// AsyncHalvingConfigV0 configures asynchronous successive halving.
type AsyncHalvingConfigV0 struct {
	Metric               string  `json:"metric"`
	SmallerIsBetter      *bool   `json:"smaller_is_better"`
	SourceTrialID        *int    `json:"source_trial_id"`
	SourceCheckpointUUID *string `json:"source_checkpoint_uuid"`

	NumRungs            int      `json:"num_rungs"`
	MaxLength           LengthV0 `json:"max_length"`
	MaxTrials           int      `json:"max_trials"`
	Divisor             *float64 `json:"divisor"`
	MaxConcurrentTrials *int     `json:"max_concurrent_trials"`
	StopOnce            *bool    `json:"stop_once"`
}

// Unit implements the model.InUnits interface.
func (a AsyncHalvingConfigV0) Unit() Unit {
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

// AdaptiveModePtr is like &AdaptiveMode("standard"), except it works.
func AdaptiveModePtr(mode string) *AdaptiveMode {
	tmp := AdaptiveMode(mode)
	return &tmp
}

//go:generate ../gen.sh
// AdaptiveASHAConfigV0 configures an adaptive searcher for use with ASHA.
type AdaptiveASHAConfigV0 struct {
	Metric               string  `json:"metric"`
	SmallerIsBetter      *bool   `json:"smaller_is_better"`
	SourceTrialID        *int    `json:"source_trial_id"`
	SourceCheckpointUUID *string `json:"source_checkpoint_uuid"`

	MaxLength           LengthV0      `json:"max_length"`
	MaxTrials           int           `json:"max_trials"`
	BracketRungs        *[]int        `json:"bracket_rungs"`
	Divisor             *float64      `json:"divisor"`
	Mode                *AdaptiveMode `json:"mode"`
	MaxRungs            *int          `json:"max_rungs"`
	MaxConcurrentTrials *int          `json:"max_concurrent_trials"`
	StopOnce            *bool         `json:"stop_once"`
}

// Unit implements the model.InUnits interface.
func (a AdaptiveASHAConfigV0) Unit() Unit {
	return a.MaxLength.Unit
}

// PBTReplaceConfig configures replacement for a PBT search.
type PBTReplaceConfig struct {
	TruncateFraction float64 `json:"truncate_fraction"`
}

// PBTExploreConfig configures exploration for a PBT search.
type PBTExploreConfig struct {
	ResampleProbability float64 `json:"resample_probability"`
	PerturbFactor       float64 `json:"perturb_factor"`
}

//go:generate ../gen.sh
// PBTConfigV0 configures a PBT search.
type PBTConfigV0 struct {
	Metric               string  `json:"metric"`
	SmallerIsBetter      *bool   `json:"smaller_is_better"`
	SourceTrialID        *int    `json:"source_trial_id"`
	SourceCheckpointUUID *string `json:"source_checkpoint_uuid"`

	PopulationSize int      `json:"population_size"`
	NumRounds      int      `json:"num_rounds"`
	LengthPerRound LengthV0 `json:"length_per_round"`

	ReplaceFunction PBTReplaceConfig `json:"replace_function"`
	ExploreFunction PBTExploreConfig `json:"explore_function"`
}

// Unit implements the model.InUnits interface.
func (p PBTConfig) Unit() Unit {
	return p.LengthPerRound.Unit
}
