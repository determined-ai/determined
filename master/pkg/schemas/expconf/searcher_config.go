package expconf

import (
	"encoding/json"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/union"
)

//go:generate ../gen.sh
// SearcherConfigV0 holds the searcher configurations.
type SearcherConfigV0 struct {
	RawSingleConfig       *SingleConfigV0       `union:"name,single" json:"-"`
	RawRandomConfig       *RandomConfigV0       `union:"name,random" json:"-"`
	RawGridConfig         *GridConfigV0         `union:"name,grid" json:"-"`
	RawAsyncHalvingConfig *AsyncHalvingConfigV0 `union:"name,async_halving" json:"-"`
	RawAdaptiveASHAConfig *AdaptiveASHAConfigV0 `union:"name,adaptive_asha" json:"-"`
	RawPBTConfig          *PBTConfigV0          `union:"name,pbt" json:"-"`
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
	case s.RawSingleConfig != nil:
		return s.RawSingleConfig.Unit()
	case s.RawRandomConfig != nil:
		return s.RawRandomConfig.Unit()
	case s.RawGridConfig != nil:
		return s.RawGridConfig.Unit()
	case s.RawAsyncHalvingConfig != nil:
		return s.RawAsyncHalvingConfig.Unit()
	case s.RawAdaptiveASHAConfig != nil:
		return s.RawAdaptiveASHAConfig.Unit()
	case s.RawPBTConfig != nil:
		return s.RawPBTConfig.Unit()
	default:
		panic("no searcher type specified")
	}
}

//go:generate ../gen.sh
// SingleConfigV0 configures a single trial.
type SingleConfigV0 struct {
	RawMetric               string  `json:"metric"`
	RawSmallerIsBetter      *bool   `json:"smaller_is_better"`
	RawSourceTrialID        *int    `json:"source_trial_id"`
	RawSourceCheckpointUUID *string `json:"source_checkpoint_uuid"`

	RawMaxLength LengthV0 `json:"max_length"`
}

// Unit implements the model.InUnits interface.
func (s SingleConfigV0) Unit() Unit {
	return s.RawMaxLength.Unit
}

//go:generate ../gen.sh
// RandomConfigV0 configures a random search.
type RandomConfigV0 struct {
	RawMetric               string  `json:"metric"`
	RawSmallerIsBetter      *bool   `json:"smaller_is_better"`
	RawSourceTrialID        *int    `json:"source_trial_id"`
	RawSourceCheckpointUUID *string `json:"source_checkpoint_uuid"`

	RawMaxLength           LengthV0 `json:"max_length"`
	RawMaxTrials           int      `json:"max_trials"`
	RawMaxConcurrentTrials *int     `json:"max_concurrent_trials"`
}

// Unit implements the model.InUnits interface.
func (r RandomConfigV0) Unit() Unit {
	return r.RawMaxLength.Unit
}

//go:generate ../gen.sh
// GridConfigV0 configures a grid search.
type GridConfigV0 struct {
	RawMetric               string  `json:"metric"`
	RawSmallerIsBetter      *bool   `json:"smaller_is_better"`
	RawSourceTrialID        *int    `json:"source_trial_id"`
	RawSourceCheckpointUUID *string `json:"source_checkpoint_uuid"`

	RawMaxLength           LengthV0 `json:"max_length"`
	RawMaxConcurrentTrials *int     `json:"max_concurrent_trials"`
}

// Unit implements the model.InUnits interface.
func (g GridConfigV0) Unit() Unit {
	return g.RawMaxLength.Unit
}

//go:generate ../gen.sh
// AsyncHalvingConfigV0 configures asynchronous successive halving.
type AsyncHalvingConfigV0 struct {
	RawMetric               string  `json:"metric"`
	RawSmallerIsBetter      *bool   `json:"smaller_is_better"`
	RawSourceTrialID        *int    `json:"source_trial_id"`
	RawSourceCheckpointUUID *string `json:"source_checkpoint_uuid"`

	RawNumRungs            int      `json:"num_rungs"`
	RawMaxLength           LengthV0 `json:"max_length"`
	RawMaxTrials           int      `json:"max_trials"`
	RawDivisor             *float64 `json:"divisor"`
	RawMaxConcurrentTrials *int     `json:"max_concurrent_trials"`
	RawStopOnce            *bool    `json:"stop_once"`
}

// Unit implements the model.InUnits interface.
func (a AsyncHalvingConfigV0) Unit() Unit {
	return a.RawMaxLength.Unit
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
	RawMetric               string  `json:"metric"`
	RawSmallerIsBetter      *bool   `json:"smaller_is_better"`
	RawSourceTrialID        *int    `json:"source_trial_id"`
	RawSourceCheckpointUUID *string `json:"source_checkpoint_uuid"`

	RawMaxLength           LengthV0      `json:"max_length"`
	RawMaxTrials           int           `json:"max_trials"`
	RawBracketRungs        []int         `json:"bracket_rungs"`
	RawDivisor             *float64      `json:"divisor"`
	RawMode                *AdaptiveMode `json:"mode"`
	RawMaxRungs            *int          `json:"max_rungs"`
	RawMaxConcurrentTrials *int          `json:"max_concurrent_trials"`
	RawStopOnce            *bool         `json:"stop_once"`
}

// Unit implements the model.InUnits interface.
func (a AdaptiveASHAConfigV0) Unit() Unit {
	return a.RawMaxLength.Unit
}

// PBTReplaceConfig configures replacement for a PBT search.
type PBTReplaceConfig struct {
	RawTruncateFraction float64 `json:"truncate_fraction"`
}

// PBTExploreConfig configures exploration for a PBT search.
type PBTExploreConfig struct {
	RawResampleProbability float64 `json:"resample_probability"`
	RawPerturbFactor       float64 `json:"perturb_factor"`
}

//go:generate ../gen.sh
// PBTConfigV0 configures a PBT search.
type PBTConfigV0 struct {
	RawMetric               string  `json:"metric"`
	RawSmallerIsBetter      *bool   `json:"smaller_is_better"`
	RawSourceTrialID        *int    `json:"source_trial_id"`
	RawSourceCheckpointUUID *string `json:"source_checkpoint_uuid"`

	RawPopulationSize int      `json:"population_size"`
	RawNumRounds      int      `json:"num_rounds"`
	RawLengthPerRound LengthV0 `json:"length_per_round"`

	RawReplaceFunction PBTReplaceConfig `json:"replace_function"`
	RawExploreFunction PBTExploreConfig `json:"explore_function"`
}

// Unit implements the model.InUnits interface.
func (p PBTConfigV0) Unit() Unit {
	return p.RawLengthPerRound.Unit
}
