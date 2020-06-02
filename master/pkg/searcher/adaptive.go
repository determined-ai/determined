package searcher

import (
	"fmt"
	"math"
	"sort"

	"github.com/determined-ai/determined/master/pkg/model"
)

func getBracketMaxTrials(maxTrials int, divisor float64, brackets []int) []int {
	// This allocation will result in roughly equal total step budget
	// allocated to each rung.
	// Each bracket roughly requires numRungs * targetTrialSteps budget to evaluate
	// divisor ^ (numRungs - 1) configurations.  Hence, we can compute the average
	// budget per configuration for each bracket and back into the number of
	// trials per bracket if we want roughly equal total step budget.
	var bracketWeight []float64
	denom := 0.
	for i, numRungs := range brackets {
		bracketWeight = append(bracketWeight, math.Pow(divisor, float64(numRungs-1))/float64(numRungs))
		denom += bracketWeight[i]
	}
	var bracketTrials []int
	allocated := 0
	for i := 0; i < len(brackets); i++ {
		bracketTrials = append(bracketTrials, max(int(bracketWeight[i]/denom*float64(maxTrials)), 1))

		allocated += bracketTrials[i]
	}
	remainder := max(maxTrials-allocated, 0)
	bracketTrials[0] += remainder
	return bracketTrials
}

func newAdaptiveSearch(config model.AdaptiveConfig) SearchMethod {
	modeFunc := parseAdaptiveMode(config.Mode)

	brackets := config.BracketRungs
	if len(brackets) == 0 {
		brackets = modeFunc(config.MaxRungs)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(brackets)))
	bracketMaxTrials := getBracketMaxTrials(config.MaxTrials, config.Divisor, brackets)
	//fmt.Printf("bracketRungs: %v\n", brackets)
	//fmt.Printf("bracketTrials: %v\n", bracketMaxTrials)

	methods := make([]SearchMethod, 0, len(brackets))
	for i, numRungs := range brackets {
		c := model.AsyncHalvingConfig{
			Metric:           config.Metric,
			SmallerIsBetter:  config.SmallerIsBetter,
			NumRungs:         numRungs,
			TargetTrialSteps: config.TargetTrialSteps,
			MaxTrials:        bracketMaxTrials[i],
			Divisor:          config.Divisor,
		}
		methods = append(methods, newAsyncHalvingSearch(c))
		fmt.Printf("Bracket created with %d rungs and %d max trials\n", numRungs, bracketMaxTrials[i])
	}

	return newTournamentSearch(methods...)
}

type adaptiveMode func(maxRungs int) []int

func conservativeMode(maxRungs int) []int {
	bracketRungs := make([]int, 0, maxRungs)
	for i := 1; i <= maxRungs; i++ {
		bracketRungs = append(bracketRungs, i)
	}
	return bracketRungs
}

func standardMode(maxRungs int) []int {
	var bracketRungs []int
	for i := (maxRungs-1)/2 + 1; i <= maxRungs; i++ {
		bracketRungs = append(bracketRungs, i)
	}
	return bracketRungs
}

func aggressiveMode(maxRungs int) []int {
	return []int{maxRungs}
}

func parseAdaptiveMode(rawMode model.AdaptiveMode) adaptiveMode {
	switch rawMode {
	case model.ConservativeMode:
		return conservativeMode
	case model.StandardMode:
		return standardMode
	case model.AggressiveMode:
		return aggressiveMode
	default:
		panic(fmt.Sprintf("unexpected adaptive mode: %s", rawMode))
	}
}
