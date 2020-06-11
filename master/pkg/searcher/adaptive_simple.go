package searcher

import (
	"math"
	"sort"

	"github.com/determined-ai/determined/master/pkg/model"
)

func newAdaptiveSimpleSearch(config model.AdaptiveSimpleConfig) SearchMethod {
	config.MaxRungs = min(
		config.MaxRungs,
		int(math.Log(float64(config.MaxSteps))/math.Log(config.Divisor))+1)
	brackets := parseAdaptiveMode(config.Mode)(config.MaxRungs)
	sort.Sort(sort.Reverse(sort.IntSlice(brackets)))
	bracketMaxTrials, bracketWeights, totalWeight := getBracketMaxTrials(
		config.MaxTrials, config.Divisor, brackets)
	maxConcurrentTrials := getBracketMaxConcurrentTrials(
		config.MaxConcurrentTrials, bracketWeights, totalWeight)

	methods := make([]SearchMethod, 0, len(brackets))
	for i, numRungs := range brackets {
		bracketConcurrentTrials := int(bracketWeights[i]/totalWeight*float64(maxConcurrentTrials)) + 1
		c := model.AsyncHalvingConfig{
			Metric:              config.Metric,
			SmallerIsBetter:     config.SmallerIsBetter,
			TargetTrialSteps:    config.MaxSteps,
			MaxTrials:           bracketMaxTrials[i],
			Divisor:             config.Divisor,
			NumRungs:            numRungs,
			MaxConcurrentTrials: &bracketConcurrentTrials,
		}
		methods = append(methods, newAsyncHalvingSearch(c))
	}

	return newTournamentSearch(methods...)
}
