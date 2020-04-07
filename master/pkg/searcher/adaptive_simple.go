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

func newAdaptiveSimpleSearch(config model.AdaptiveSimpleConfig) SearchMethod {
	brackets := parseAdaptiveMode(config.Mode)(config.MaxRungs)
	sort.Sort(sort.Reverse(sort.IntSlice(brackets)))

	methods := make([]SearchMethod, 0, len(brackets))
	for i, numRungs := range brackets {
		c := model.AsyncHalvingConfig{
			Metric:           config.Metric,
			SmallerIsBetter:  config.SmallerIsBetter,
			TargetTrialSteps: config.MaxSteps,
			Divisor:          config.Divisor,
			NumRungs:         numRungs,
			TrainStragglers:  true,
		}
		numTrials := max(maxTrials(config.MaxTrials, len(brackets), i), 1)
		methods = append(methods, newAsyncHalvingSimpleSearch(c, numTrials))
	}

	return newTournamentSearch(methods...)
}

func newAsyncHalvingSimpleSearch(config model.AsyncHalvingConfig, trials int) SearchMethod {
	rungs := make([]*rung, 0, config.NumRungs)
	expectedSteps := 0
	expectedWorkloads := 0
	for id := 0; id < config.NumRungs; id++ {
		stepsNeeded := max(int(float64(config.TargetTrialSteps)/
			math.Pow(config.Divisor, float64(config.NumRungs-id-1))), 1)
		startTrials := max(int(float64(trials)/math.Pow(config.Divisor, float64(id))), 1)
		if id != 0 {
			prev := rungs[id-1]
			stepsNeeded = max(stepsNeeded, prev.stepsNeeded+1)
			startTrials = max(startTrials, prev.promoteTrials)
			prev.promoteTrials = startTrials
			expectedSteps += (stepsNeeded - rungs[id-1].stepsNeeded) * startTrials
			expectedWorkloads += (stepsNeeded - rungs[id-1].stepsNeeded + 1) * startTrials
		} else {
			expectedSteps += stepsNeeded * startTrials
			expectedWorkloads += (stepsNeeded + 1) * startTrials
		}
		rungs = append(rungs, &rung{stepsNeeded: stepsNeeded, startTrials: startTrials})
	}
	config.StepBudget = expectedSteps
	return &asyncHalvingSearch{
		AsyncHalvingConfig: config,
		rungs:              rungs,
		trialRungs:         make(map[RequestID]int),
		expectedWorkloads:  expectedWorkloads,
	}
}
