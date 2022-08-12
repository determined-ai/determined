package expconf

import (
	"encoding/json"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/schemas"
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
	RawCustomConfig       *CustomConfigV0       `union:"name,custom" json:"-"`

	// These searchers are allowed only to help parse old experiment configs.
	RawSyncHalvingConfig    *SyncHalvingConfigV0    `union:"name,sync_halving" json:"-"`
	RawAdaptiveConfig       *AdaptiveConfigV0       `union:"name,adaptive" json:"-"`
	RawAdaptiveSimpleConfig *AdaptiveSimpleConfigV0 `union:"name,adaptive_simple" json:"-"`

	RawMetric               *string `json:"metric"`
	RawSmallerIsBetter      *bool   `json:"smaller_is_better"`
	RawSourceTrialID        *int    `json:"source_trial_id"`
	RawSourceCheckpointUUID *string `json:"source_checkpoint_uuid"`
}

// Merge implements schemas.Mergeable.
func (s SearcherConfigV0) Merge(other interface{}) interface{} {
	return schemas.UnionMerge(s, other)
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
	case s.RawCustomConfig != nil:
		panic("custom searcher config does not provide Unit()")
	case s.RawSyncHalvingConfig != nil:
		panic("cannot get unit of EOL searcher class")
	case s.RawAdaptiveConfig != nil:
		panic("cannot get unit of EOL searcher class")
	case s.RawAdaptiveSimpleConfig != nil:
		panic("cannot get unit of EOL searcher class")
	default:
		panic("no searcher type specified")
	}
}

//go:generate ../gen.sh
// CustomConfigV0 configures a custom search.
type CustomConfigV0 struct {
}

//go:generate ../gen.sh
// SingleConfigV0 configures a single trial.
type SingleConfigV0 struct {
	RawMaxLength *LengthV0 `json:"max_length"`
}

// Unit implements the model.InUnits interface.
func (s SingleConfigV0) Unit() Unit {
	return s.RawMaxLength.Unit
}

//go:generate ../gen.sh
// RandomConfigV0 configures a random search.
type RandomConfigV0 struct {
	RawMaxLength           *LengthV0 `json:"max_length"`
	RawMaxTrials           *int      `json:"max_trials"`
	RawMaxConcurrentTrials *int      `json:"max_concurrent_trials"`
}

// Unit implements the model.InUnits interface.
func (r RandomConfigV0) Unit() Unit {
	return r.RawMaxLength.Unit
}

//go:generate ../gen.sh
// GridConfigV0 configures a grid search.
type GridConfigV0 struct {
	RawMaxLength           *LengthV0 `json:"max_length"`
	RawMaxConcurrentTrials *int      `json:"max_concurrent_trials"`
}

// Unit implements the model.InUnits interface.
func (g GridConfigV0) Unit() Unit {
	return g.RawMaxLength.Unit
}

//go:generate ../gen.sh
// AsyncHalvingConfigV0 configures asynchronous successive halving.
type AsyncHalvingConfigV0 struct {
	RawNumRungs            *int      `json:"num_rungs"`
	RawMaxLength           *LengthV0 `json:"max_length"`
	RawMaxTrials           *int      `json:"max_trials"`
	RawDivisor             *float64  `json:"divisor"`
	RawMaxConcurrentTrials *int      `json:"max_concurrent_trials"`
	RawStopOnce            *bool     `json:"stop_once"`
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
	RawMaxLength           *LengthV0     `json:"max_length"`
	RawMaxTrials           *int          `json:"max_trials"`
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

//go:generate ../gen.sh
// SyncHalvingConfigV0 is a legacy config.
type SyncHalvingConfigV0 struct {
	RawNumRungs        *int      `json:"num_rungs"`
	RawMaxLength       *LengthV0 `json:"max_length"`
	RawBudget          *LengthV0 `json:"budget"`
	RawDivisor         *float64  `json:"divisor"`
	RawTrainStragglers *bool     `json:"train_stragglers"`
}

//go:generate ../gen.sh
// AdaptiveConfigV0 is a legacy config.
type AdaptiveConfigV0 struct {
	RawMaxLength       *LengthV0     `json:"max_length"`
	RawBudget          *LengthV0     `json:"budget"`
	RawBracketRungs    []int         `json:"bracket_rungs"`
	RawDivisor         *float64      `json:"divisor"`
	RawTrainStragglers *bool         `json:"train_stragglers"`
	RawMode            *AdaptiveMode `json:"mode"`
	RawMaxRungs        *int          `json:"max_rungs"`
}

//go:generate ../gen.sh
// AdaptiveSimpleConfigV0 is a legacy config.
type AdaptiveSimpleConfigV0 struct {
	RawMaxLength *LengthV0     `json:"max_length"`
	RawMaxTrials *int          `json:"max_trials"`
	RawDivisor   *float64      `json:"divisor"`
	RawMode      *AdaptiveMode `json:"mode"`
	RawMaxRungs  *int          `json:"max_rungs"`
}

// AssertCurrent distinguishes configs which are only parsable from those that are runnable.
func (s SearcherConfig) AssertCurrent() error {
	switch {
	case s.RawSyncHalvingConfig != nil:
		return errors.New(
			"the 'sync_halving' searcher has been removed and is not valid for new experiments",
		)
	case s.RawAdaptiveConfig != nil:
		return errors.New(
			"the 'adaptive' searcher has been removed and is not valid for new experiments",
		)
	case s.RawAdaptiveSimpleConfig != nil:
		return errors.New(
			"the 'adaptive_simple' searcher has been removed and is not valid for new experiments",
		)
	}
	return nil
}
