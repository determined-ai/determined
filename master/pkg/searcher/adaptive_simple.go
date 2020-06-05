package searcher

import (
	"fmt"
	"sort"

	"github.com/determined-ai/determined/master/pkg/model"
)

func newAdaptiveSimpleSearch(config model.AdaptiveSimpleConfig) SearchMethod {
	brackets := parseAdaptiveMode(config.Mode)(config.MaxRungs)
	sort.Sort(sort.Reverse(sort.IntSlice(brackets)))
	bracketMaxTrials := getBracketMaxTrials(config.MaxTrials, config.Divisor, brackets)
	//fmt.Printf("bracketRungs: %v\n", brackets)
	//fmt.Printf("bracketTrials: %v\n", bracketMaxTrials)

	methods := make([]SearchMethod, 0, len(brackets))
	for i, numRungs := range brackets {
		c := model.AsyncHalvingConfig{
			Metric:           config.Metric,
			SmallerIsBetter:  config.SmallerIsBetter,
			TargetTrialSteps: config.MaxSteps,
			MaxTrials:        bracketMaxTrials[i],
			Divisor:          config.Divisor,
			NumRungs:         numRungs,
		}
		methods = append(methods, newAsyncHalvingSearch(c))
		fmt.Printf("Bracket created with %d rungs and %d max trials\n", numRungs, bracketMaxTrials[i])
	}

	return newTournamentSearch(methods...)
}
