package searcher

import (
	"sort"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/mathx"
	"github.com/determined-ai/determined/master/pkg/protoutils"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/proto/pkg/experimentv1"
)

type SearchSummary struct {
	Runs   []RunSummary
	Config expconf.SearcherConfig
}

type SearchUnit struct {
	Name      string
	Value     int
	MaxLength bool
}

func (su SearchUnit) Proto() *experimentv1.SearchUnit {
	return &experimentv1.SearchUnit{
		Name:      su.Name,
		Value:     int32(su.Value),
		MaxLength: su.MaxLength,
	}
}

type RunSummary struct {
	Count int
	Unit  SearchUnit
}

func (rs RunSummary) Proto() *experimentv1.RunSummary {
	return &experimentv1.RunSummary{
		Count: int32(rs.Count),
		Unit:  rs.Unit.Proto(),
	}
}

func (s SearchSummary) Proto() *experimentv1.SearchSummary {
	var runSummaries []*experimentv1.RunSummary
	for _, v := range s.Runs {
		runSummaries = append(runSummaries, v.Proto())
	}
	return &experimentv1.SearchSummary{
		Config: protoutils.ToStruct(s.Config),
		Runs:   runSummaries,
	}
}

// Simulate generates the intended training plan for the searcher.
func Simulate(conf expconf.SearcherConfig, hparams expconf.Hyperparameters) (SearchSummary, error) {
	searchSummary := SearchSummary{
		Runs:   []RunSummary{},
		Config: conf,
	}
	switch {
	case conf.RawSingleConfig != nil:
		searchSummary.Runs = append(searchSummary.Runs, RunSummary{Count: 1, Unit: SearchUnit{MaxLength: true}})
		return searchSummary, nil
	case conf.RawRandomConfig != nil:
		maxRuns := conf.RawRandomConfig.MaxTrials()
		searchSummary.Runs = append(searchSummary.Runs, RunSummary{Count: maxRuns, Unit: SearchUnit{MaxLength: true}})
		return searchSummary, nil
	case conf.RawGridConfig != nil:
		hparamGrid := newHyperparameterGrid(hparams)
		searchSummary.Runs = append(searchSummary.Runs, RunSummary{Count: len(hparamGrid), Unit: SearchUnit{MaxLength: true}})
		return searchSummary, nil
	case conf.RawAdaptiveASHAConfig != nil:
		ashaConfig := conf.RawAdaptiveASHAConfig
		brackets := makeBrackets(*ashaConfig)
		unitsPerRun := make(map[int]int)
		for _, bracket := range brackets {
			rungs := makeRungs(bracket.numRungs, ashaConfig.Divisor(), ashaConfig.Length().Units)
			rungRuns := bracket.maxRuns
			// For each rung, calculate number of runs that will be stopped before next rung
			// to determine the number of runs that will only train to the current rung.
			for i, rung := range rungs {
				rungUnits := int(rung.UnitsNeeded)
				runsContinued := mathx.Max(int(float64(rungRuns)/ashaConfig.Divisor()), 1)
				runsStopped := rungRuns - runsContinued
				if i == len(rungs)-1 {
					runsStopped = rungRuns
				}
				unitsPerRun[rungUnits] += runsStopped
				rungRuns = runsContinued
			}
		}
		for units, numRuns := range unitsPerRun {
			searchSummary.Runs = append(searchSummary.Runs, RunSummary{
				Count: numRuns,
				Unit: SearchUnit{
					Name:  string(ashaConfig.Length().Unit),
					Value: units,
				},
			})
		}
		// Sort by target units for consistency in output.
		sort.Slice(searchSummary.Runs, func(i, j int) bool {
			return searchSummary.Runs[i].Unit.Value < searchSummary.Runs[j].Unit.Value
		})
		return searchSummary, nil
	default:
		return SearchSummary{}, errors.New("invalid searcher configuration")
	}
}
