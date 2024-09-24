package expconf

import (
	"encoding/json"

	log "github.com/sirupsen/logrus"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/schemas"
	"github.com/determined-ai/determined/master/pkg/union"
)

// SearcherConfigV0 holds the searcher configurations.
//
//go:generate ../gen.sh
type SearcherConfigV0 struct {
	RawSingleConfig       *SingleConfigV0       `union:"name,single" json:"-"`
	RawRandomConfig       *RandomConfigV0       `union:"name,random" json:"-"`
	RawGridConfig         *GridConfigV0         `union:"name,grid" json:"-"`
	RawAsyncHalvingConfig *AsyncHalvingConfigV0 `union:"name,async_halving" json:"-"`
	RawAdaptiveASHAConfig *AdaptiveASHAConfigV0 `union:"name,adaptive_asha" json:"-"`

	// TODO(DET-8577): There should not be a need to parse EOL searchers if we get rid of parsing
	//                 active experiment configs unnecessarily.
	// These searchers are allowed only to help parse old experiment configs.
	RawSyncHalvingConfig    *SyncHalvingConfigV0    `union:"name,sync_halving" json:"-"`
	RawAdaptiveConfig       *AdaptiveConfigV0       `union:"name,adaptive" json:"-"`
	RawAdaptiveSimpleConfig *AdaptiveSimpleConfigV0 `union:"name,adaptive_simple" json:"-"`
	RawCustomConfig         *CustomConfigV0         `union:"name,custom" json:"-"`

	RawMetric               *string `json:"metric"`
	RawSmallerIsBetter      *bool   `json:"smaller_is_better"`
	RawSourceTrialID        *int    `json:"source_trial_id"`
	RawSourceCheckpointUUID *string `json:"source_checkpoint_uuid"`
}

// Merge implements schemas.Mergeable.
func (s SearcherConfigV0) Merge(other SearcherConfigV0) SearcherConfigV0 {
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

// AsLegacy converts a current ExperimentConfig to a (limited capacity) LegacySearcher.
func (s SearcherConfigV0) AsLegacy() LegacySearcher {
	var name string
	switch {
	case s.RawSingleConfig != nil:
		name = "single"
	case s.RawRandomConfig != nil:
		name = "random"
	case s.RawGridConfig != nil:
		name = "grid"
	case s.RawAsyncHalvingConfig != nil:
		name = "async_halving"
	case s.RawAdaptiveASHAConfig != nil:
		name = "adaptive_asha"
	case s.RawCustomConfig != nil:
		name = "custom"
	case s.RawSyncHalvingConfig != nil:
		name = "sync_halving"
	case s.RawAdaptiveConfig != nil:
		name = "adaptive"
	case s.RawAdaptiveSimpleConfig != nil:
		name = "adaptive_simple"
	default:
		panic("no searcher type specified")
	}
	return LegacySearcher{
		Name:            name,
		Metric:          s.Metric(),
		SmallerIsBetter: s.SmallerIsBetter(),
	}
}

// SingleConfigV0 configures a single trial.
//
//go:generate ../gen.sh
type SingleConfigV0 struct {
	RawMaxLength *LengthV0 `json:"max_length"`
}

// RandomConfigV0 configures a random search.
//
//go:generate ../gen.sh
type RandomConfigV0 struct {
	RawMaxLength           *LengthV0 `json:"max_length"`
	RawMaxTrials           *int      `json:"max_trials"`
	RawMaxConcurrentTrials *int      `json:"max_concurrent_trials"`
}

// GridConfigV0 configures a grid search.
//
//go:generate ../gen.sh
type GridConfigV0 struct {
	RawMaxLength           *LengthV0 `json:"max_length"`
	RawMaxConcurrentTrials *int      `json:"max_concurrent_trials"`
}

// AsyncHalvingConfigV0 configures asynchronous successive halving.
//
//go:generate ../gen.sh
type AsyncHalvingConfigV0 struct {
	RawNumRungs            *int     `json:"num_rungs"`
	RawMaxTrials           *int     `json:"max_trials"`
	RawDivisor             *float64 `json:"divisor"`
	RawMaxConcurrentTrials *int     `json:"max_concurrent_trials"`
	RawMaxTime             *int     `json:"max_time"`
	RawTimeMetric          *string  `json:"time_metric"`
	// These config options are deprecated and should not be used.
	// They exist to help parse legacy exp configs.
	RawMaxLength *LengthV0 `json:"max_length"`
	RawStopOnce  *bool     `json:"stop_once"`
}

// Length returns the maximum training length.
func (a AsyncHalvingConfigV0) Length() Length {
	if a.RawMaxTime != nil && a.RawTimeMetric != nil {
		return Length{Unit: Unit(*a.RawTimeMetric), Units: uint64(*a.RawMaxTime)}
	}
	// Parse legacy expconfs for backwards compat.
	return *a.RawMaxLength
}

// AdaptiveMode specifies how aggressively to perform early stopping.
type AdaptiveMode string

const (
	// AggressiveMode quickly stops underperforming trials, which enables the searcher to explore
	// more hyperparameter configurations.
	AggressiveMode AdaptiveMode = "aggressive"
	// StandardMode provides a balance between downsampling and hyperparameter exploration.
	StandardMode AdaptiveMode = "standard"
	// ConservativeMode performs minimal downsampling at the cost of not exploring as many
	// configurations.
	ConservativeMode AdaptiveMode = "conservative"
)

// AdaptiveASHAConfigV0 configures an adaptive searcher for use with ASHA.
//
//go:generate ../gen.sh
type AdaptiveASHAConfigV0 struct {
	RawMaxTrials           *int          `json:"max_trials"`
	RawBracketRungs        []int         `json:"bracket_rungs"`
	RawDivisor             *float64      `json:"divisor"`
	RawMode                *AdaptiveMode `json:"mode"`
	RawMaxRungs            *int          `json:"max_rungs"`
	RawMaxConcurrentTrials *int          `json:"max_concurrent_trials"`
	RawMaxTime             *int          `json:"max_time"`
	RawTimeMetric          *string       `json:"time_metric"`
	// These config options are deprecated and should not be used.
	// They exist to help parse legacy exp configs.
	RawMaxLength *LengthV0 `json:"max_length"`
	RawStopOnce  *bool     `json:"stop_once"`
}

// Length returns the maximum training length.
func (a AdaptiveASHAConfigV0) Length() Length {
	if a.RawMaxTime != nil && a.RawTimeMetric != nil {
		return Length{Unit: Unit(*a.RawTimeMetric), Units: uint64(*a.RawMaxTime)}
	}
	// Parse legacy expconfs for backwards compat.
	return *a.RawMaxLength
}

// SyncHalvingConfigV0 is a legacy config.
//
//go:generate ../gen.sh
type SyncHalvingConfigV0 struct {
	RawNumRungs        *int      `json:"num_rungs"`
	RawMaxLength       *LengthV0 `json:"max_length"`
	RawBudget          *LengthV0 `json:"budget"`
	RawDivisor         *float64  `json:"divisor"`
	RawTrainStragglers *bool     `json:"train_stragglers"`
}

// AdaptiveConfigV0 is a legacy config.
//
//go:generate ../gen.sh
type AdaptiveConfigV0 struct {
	RawMaxLength       *LengthV0     `json:"max_length"`
	RawBudget          *LengthV0     `json:"budget"`
	RawBracketRungs    []int         `json:"bracket_rungs"`
	RawDivisor         *float64      `json:"divisor"`
	RawTrainStragglers *bool         `json:"train_stragglers"`
	RawMode            *AdaptiveMode `json:"mode"`
	RawMaxRungs        *int          `json:"max_rungs"`
}

// AdaptiveSimpleConfigV0 is a legacy config.
//
//go:generate ../gen.sh
type AdaptiveSimpleConfigV0 struct {
	RawMaxLength *LengthV0     `json:"max_length"`
	RawMaxTrials *int          `json:"max_trials"`
	RawDivisor   *float64      `json:"divisor"`
	RawMode      *AdaptiveMode `json:"mode"`
	RawMaxRungs  *int          `json:"max_rungs"`
}

// CustomConfigV0 configures a custom search.
//
//go:generate ../gen.sh
type CustomConfigV0 struct {
	RawUnit *Unit `json:"unit"`
}

// AssertCurrent distinguishes configs which are only parsable from those that are runnable and logs deprecation
// warnings for legacy fields.
func (s SearcherConfig) AssertCurrent() error {
	switch {
	case s.RawAdaptiveASHAConfig != nil:
		if s.RawAdaptiveASHAConfig.RawMaxLength != nil {
			log.Warn(
				"the `max_length` field of the searcher config has been deprecated and will be removed in a " +
					"future release.")
		}
		if s.RawAdaptiveASHAConfig.RawStopOnce != nil {
			log.Warn("the `stop_once` field of the searcher config has been deprecated and will be removed in " +
				"a future release.")
		}
		if s.RawAdaptiveASHAConfig.RawMaxTime == nil || s.RawAdaptiveASHAConfig.RawTimeMetric == nil {
			return errors.New("the `adaptive_asha` searcher requires `max_time` and `time_metric` to be set")
		}
	case s.RawAsyncHalvingConfig != nil:
		if s.RawAsyncHalvingConfig.RawMaxLength != nil {
			log.Warn("the `max_length` field of the searcher config has been deprecated and will be removed in " +
				"a future release.")
		}
		if s.RawAsyncHalvingConfig.RawStopOnce != nil {
			log.Warn("the `stop_once` field of the searcher config has been deprecated and will be removed in " +
				"a future release.")
		}
		if s.RawAsyncHalvingConfig.RawMaxTime == nil || s.RawAsyncHalvingConfig.RawTimeMetric == nil {
			return errors.New("the `async_halving` searcher requires `max_time` and `time_metric` to be set")
		}
	case s.RawGridConfig != nil:
		if s.RawGridConfig.RawMaxLength != nil {
			log.Warn("the `max_length` field of the searcher config has been deprecated and will be removed in " +
				"a future release.")
		}
	case s.RawSingleConfig != nil:
		if s.RawSingleConfig.RawMaxLength != nil {
			log.Warn("the `max_length` field of the searcher config has been deprecated and will be removed in " +
				"a future release.")
		}
	case s.RawRandomConfig != nil:
		if s.RawRandomConfig.RawMaxLength != nil {
			log.Warn("the `max_length` field of the searcher config has been deprecated and will be removed in " +
				"a future release.")
		}
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
	case s.RawCustomConfig != nil:
		return errors.New(
			"the 'custom' searcher has been removed and is not valid for new experiments",
		)
	}
	return nil
}
