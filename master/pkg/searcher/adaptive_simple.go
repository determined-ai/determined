package searcher

import (
	"math"
	"sort"

	"github.com/determined-ai/determined/master/pkg/model"
)

func maxTrials(maxTrials, brackets, index int) int {
	count := maxTrials / brackets
	if remainder := maxTrials % brackets; index < remainder {
		return count + 1
	}
	return count
}

func newAdaptiveSimpleSearch(
	config model.AdaptiveSimpleConfig, targetBatchesPerStep int,
) SearchMethod {
	brackets := parseAdaptiveMode(config.Mode)(config.MaxRungs)
	sort.Sort(sort.Reverse(sort.IntSlice(brackets)))

	methods := make([]SearchMethod, 0, len(brackets))
	for i, numRungs := range brackets {
		c := model.SyncHalvingConfig{
			Metric:          config.Metric,
			SmallerIsBetter: config.SmallerIsBetter,
			MaxLength:       config.MaxLength,
			Divisor:         config.Divisor,
			NumRungs:        numRungs,
			TrainStragglers: true,
		}
		numTrials := max(maxTrials(config.MaxTrials, len(brackets), i), 1)
		methods = append(methods, newSyncHalvingSimpleSearch(c, numTrials, targetBatchesPerStep))
	}

	return newTournamentSearch(methods...)
}

func newSyncHalvingSimpleSearch(
	config model.SyncHalvingConfig, trials, targetBatchesPerStep int,
) SearchMethod {
	rungs := make([]*rung, 0, config.NumRungs)
	expectedUnits := 0
	for id := 0; id < config.NumRungs; id++ {
		unitsNeeded := max(int(float64(config.MaxLength.Units)/
			math.Pow(config.Divisor, float64(config.NumRungs-id-1))), 1)
		startTrials := max(int(float64(trials)/math.Pow(config.Divisor, float64(id))), 1)
		if id != 0 {
			prev := rungs[id-1]
			unitsNeeded = max(unitsNeeded, prev.unitsNeeded.Units)
			startTrials = max(startTrials, prev.promoteTrials)
			prev.promoteTrials = startTrials
			expectedUnits += (unitsNeeded - rungs[id-1].unitsNeeded.Units) * startTrials
		} else {
			expectedUnits += unitsNeeded * startTrials
		}
		rungs = append(rungs,
			&rung{
				unitsNeeded: model.NewLength(config.MaxLength.Kind, unitsNeeded),
				startTrials: startTrials,
			},
		)
	}

	config.Budget = model.NewLength(config.MaxLength.Kind, expectedUnits)
	return &syncHalvingSearch{
		SyncHalvingConfig: config,
		rungs:             rungs,
		trialRungs:        make(map[RequestID]int),
		earlyExitTrials:   make(map[RequestID]bool),
		expectedUnits:     model.NewLength(config.MaxLength.Kind, expectedUnits),
	}
}
