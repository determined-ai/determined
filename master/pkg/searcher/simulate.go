package searcher

import (
	"sort"

	"github.com/determined-ai/determined/master/pkg/ptrs"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/mathx"
	"github.com/determined-ai/determined/master/pkg/protoutils"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/proto/pkg/experimentv1"
)

// SearchSummary describes a summary of planned trials and the associated expconf.SearcherConfig.
type SearchSummary struct {
	Trials []TrialSummary
	Config expconf.SearcherConfig
}

// SearchUnit is a length unit. If MaxLength is true, Name and Value will be ignored.
type SearchUnit struct {
	Name      *string
	Value     *int32
	MaxLength bool
}

// Proto converts the SearchUnit to its protobuf representation.
func (su SearchUnit) Proto() *experimentv1.SearchUnit {
	return &experimentv1.SearchUnit{
		Name:      su.Name,
		Value:     su.Value,
		MaxLength: su.MaxLength,
	}
}

// TrialSummary is a summary of the number of trials that will train for Unit length.
type TrialSummary struct {
	Count int
	Unit  SearchUnit
}

// Proto converts the TrialSummary to its protobuf representation.
func (rs TrialSummary) Proto() *experimentv1.TrialSummary {
	return &experimentv1.TrialSummary{
		Count: int32(rs.Count),
		Unit:  rs.Unit.Proto(),
	}
}

// Proto converts the SearchSummary to its protobuf representation.
func (s SearchSummary) Proto() *experimentv1.SearchSummary {
	var trialSummaries []*experimentv1.TrialSummary
	for _, v := range s.Trials {
		trialSummaries = append(trialSummaries, v.Proto())
	}
	return &experimentv1.SearchSummary{
		Config: protoutils.ToStruct(s.Config),
		Trials: trialSummaries,
	}
}

// Simulate generates the intended training plan for the searcher.
func Simulate(conf expconf.SearcherConfig, hparams expconf.Hyperparameters) (SearchSummary, error) {
	searchSummary := SearchSummary{
		Trials: []TrialSummary{},
		Config: conf,
	}
	switch {
	case conf.RawSingleConfig != nil:
		searchSummary.Trials = append(searchSummary.Trials, TrialSummary{Count: 1, Unit: SearchUnit{MaxLength: true}})
		return searchSummary, nil
	case conf.RawRandomConfig != nil:
		maxTrials := conf.RawRandomConfig.MaxTrials()
		searchSummary.Trials = append(searchSummary.Trials, TrialSummary{Count: maxTrials, Unit: SearchUnit{MaxLength: true}})
		return searchSummary, nil
	case conf.RawGridConfig != nil:
		hparamGrid := newHyperparameterGrid(hparams)
		searchSummary.Trials = append(searchSummary.Trials, TrialSummary{Count: len(hparamGrid), Unit: SearchUnit{MaxLength: true}})
		return searchSummary, nil
	case conf.RawAdaptiveASHAConfig != nil:
		ashaConfig := conf.RawAdaptiveASHAConfig
		brackets := makeBrackets(*ashaConfig)
		unitsPerTrial := make(map[int32]int)
		for _, bracket := range brackets {
			rungs := makeRungs(bracket.numRungs, ashaConfig.Divisor(), ashaConfig.Length().Units)
			rungTrials := bracket.maxTrials
			// For each rung, calculate number of runs that will be stopped before next rung
			// to determine the number of runs that will only train to the current rung.
			for i, rung := range rungs {
				rungUnits := int(rung.UnitsNeeded)
				trialsContinued := mathx.Max(int(float64(rungTrials)/ashaConfig.Divisor()), 1)
				trialsStopped := rungTrials - trialsContinued
				if i == len(rungs)-1 {
					trialsStopped = rungTrials
				}
				unitsPerTrial[int32(rungUnits)] += trialsStopped
				rungTrials = trialsContinued
			}
		}
		for units, numTrials := range unitsPerTrial {
			searchSummary.Trials = append(searchSummary.Trials, TrialSummary{
				Count: numTrials,
				Unit: SearchUnit{
					Name:  ptrs.Ptr(string(ashaConfig.Length().Unit)),
					Value: &units,
				},
			})
		}
		// Sort by target units for consistency in output.
		sort.Slice(searchSummary.Trials, func(i, j int) bool {
			return *searchSummary.Trials[i].Unit.Value < *searchSummary.Trials[j].Unit.Value
		})
		return searchSummary, nil
	default:
		return SearchSummary{}, errors.New("invalid searcher configuration")
	}
}
