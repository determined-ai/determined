package searcher

import (
	"fmt"
	"github.com/determined-ai/determined/master/pkg/mathx"
	"github.com/determined-ai/determined/master/pkg/protoutils"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/proto/pkg/experimentv1"
	"github.com/pkg/errors"
	"math/rand"
)

// ValidationFunction calculates the validation metric for the validation step.
type ValidationFunction func(random *rand.Rand, trialID, idx int) float64

// ConstantValidation returns the same validation metric for all validation steps.
func ConstantValidation(_ *rand.Rand, _, _ int) float64 { return 1 }

// RandomValidation returns a random validation metric for each validation step.
func RandomValidation(rand *rand.Rand, _, _ int) float64 { return rand.Float64() }

// TrialIDMetric returns the trialID as the metric for all validation steps.
func TrialIDMetric(_ *rand.Rand, trialID, _ int) float64 {
	return float64(trialID)
}

type SearchSummary struct {
	Runs   map[int]SearchUnit
	Config expconf.SearcherConfig
}

type SearchUnit struct {
	Name      string
	Value     int
	Undefined bool
}

func (su SearchUnit) Proto() *experimentv1.SearchUnit {
	return &experimentv1.SearchUnit{
		Name:      su.Name,
		Value:     int32(su.Value),
		Undefined: su.Undefined,
	}
}

func (su SearchUnit) String() string {
	return fmt.Sprintf("%s(%d)", su.Name, su.Value)
}

func (s SearchSummary) Proto() *experimentv1.SearchSummary {
	runSummaries := make(map[int32]*experimentv1.SearchUnit)
	for k, v := range s.Runs {
		runSummaries[int32(k)] = v.Proto()
	}
	return &experimentv1.SearchSummary{
		Config: protoutils.ToStruct(s.Config),
		Runs:   runSummaries,
	}
}

// Simulate generates the intended training plan for the searcher.
func Simulate(conf expconf.SearcherConfig, hparams expconf.Hyperparameters) (SearchSummary, error) {
	searchSummary := SearchSummary{
		Runs:   make(map[int]SearchUnit),
		Config: conf,
	}
	switch {
	case conf.RawSingleConfig != nil:
		searchSummary.Runs[1] = SearchUnit{Undefined: true}
		return searchSummary, nil
	case conf.RawRandomConfig != nil:
		maxRuns := conf.RawRandomConfig.MaxTrials()
		searchSummary.Runs[maxRuns] = SearchUnit{Undefined: true}
		return searchSummary, nil
	case conf.RawGridConfig != nil:
		hparamGrid := NewHyperparameterGrid(hparams)
		searchSummary.Runs[len(hparamGrid)] = SearchUnit{Undefined: true}
		return searchSummary, nil
	case conf.RawAdaptiveASHAConfig != nil:
		ashaConfig := conf.RawAdaptiveASHAConfig
		brackets := makeBrackets(*ashaConfig)
		unitsPerRun := make(map[int]int)
		for _, bracket := range brackets {
			rungs := makeRungs(bracket.numRungs, ashaConfig.Divisor(), ashaConfig.Length().Units)
			rungRuns := bracket.maxTrials
			// For each rung, calculate number of runs that will be stopped before next rung.
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
			searchSummary.Runs[numRuns] = SearchUnit{
				Name:  string(ashaConfig.Length().Unit),
				Value: units,
			}
		}
		return searchSummary, nil
	default:
		return SearchSummary{}, errors.New("invalid searcher configuration")
	}
}
