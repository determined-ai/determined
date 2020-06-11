package searcher

import (
	"fmt"
	"math"
	"sort"

	"github.com/determined-ai/determined/master/pkg/model"
)

func getBracketMaxTrials(
	maxTrials int, divisor float64, brackets []int) ([]int, []float64, float64) {
	// This allocation will result in roughly equal total step budget
	// allocated to each rung.
	// Each bracket roughly requires numRungs * targetTrialSteps budget to evaluate
	// divisor ^ (numRungs - 1) configurations.  Hence, we can compute the average
	// budget per configuration for each bracket and back into the number of
	// trials per bracket if we want roughly equal total step budget.
	var bracketWeight []float64
	totalWeight := 0.
	for i, numRungs := range brackets {
		bracketWeight = append(bracketWeight, math.Pow(divisor, float64(numRungs-1))/float64(numRungs))
		totalWeight += bracketWeight[i]
	}
	var bracketTrials []int
	allocated := 0
	for i := 0; i < len(brackets); i++ {
		bracketTrials = append(
			bracketTrials, max(int(bracketWeight[i]/totalWeight*float64(maxTrials)), 1))

		allocated += bracketTrials[i]
	}
	remainder := max(maxTrials-allocated, 0)
	bracketTrials[0] += remainder
	return bracketTrials, bracketWeight, totalWeight
}

func getBracketMaxConcurrentTrials(
	maxConcurrentTrials *int, bracketWeight []float64, totalWeight float64) int {
	// This is the minimum number of trials needed to keep the training budget
	// per bracket roughly the same.  This is because we neeed to scale the
	// number of jobs per bracket has to account for the average training
	// budget per trial for the bracket.
	minConcurrentTrials := int(totalWeight / bracketWeight[len(bracketWeight)-1])
	if maxConcurrentTrials != nil {
		return max(*maxConcurrentTrials, minConcurrentTrials)
	}
	return minConcurrentTrials
}

func newAdaptiveSearch(config model.AdaptiveConfig) SearchMethod {
	modeFunc := parseAdaptiveMode(config.Mode)
	config.MaxRungs = min(
		config.MaxRungs,
		int(math.Log(float64(config.TargetTrialSteps))/math.Log(config.Divisor))+1)

	brackets := config.BracketRungs
	if len(brackets) == 0 {
		brackets = modeFunc(config.MaxRungs)
	}
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
			NumRungs:            numRungs,
			TargetTrialSteps:    config.TargetTrialSteps,
			MaxTrials:           bracketMaxTrials[i],
			Divisor:             config.Divisor,
			MaxConcurrentTrials: &bracketConcurrentTrials,
		}
		methods = append(methods, newAsyncHalvingSearch(c))
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
