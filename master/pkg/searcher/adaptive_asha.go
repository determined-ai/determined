package searcher

import (
	"math"
	"sort"

	"github.com/determined-ai/determined/master/pkg/model"
)

func getBracketMaxTrials(
	maxTrials int, divisor float64, brackets []int) []int {
	// This allocation will result in roughly equal total step budget
	// allocated to each rung.
	// Each bracket roughly requires numRungs * targetTrialSteps budget to evaluate
	// divisor ^ (numRungs - 1) configurations.  Hence, we can compute the average
	// budget per configuration for each bracket and back into the number of
	// trials per bracket if we want roughly equal total step budget.
	bracketWeight := make([]float64, 0., len(brackets))
	var totalWeight float64
	for i, numRungs := range brackets {
		bracketWeight = append(bracketWeight, math.Pow(divisor, float64(numRungs-1))/float64(numRungs))
		totalWeight += bracketWeight[i]
	}
	bracketTrials := make([]int, 0, len(brackets))
	allocated := 0
	for i := 0; i < len(brackets); i++ {
		bracketTrials = append(
			bracketTrials, max(int(bracketWeight[i]/totalWeight*float64(maxTrials)), 1))

		allocated += bracketTrials[i]
	}
	remainder := max(maxTrials-allocated, 0)
	bracketTrials[0] += remainder
	return bracketTrials
}

func getBracketMaxConcurrentTrials(
	maxConcurrentTrials int, divisor float64, maxTrials []int) []int {
	// If maxConcurrentTrials is provided, we will split that evenly across brackets
	// and fill remainder from most aggressive early stopping bracket to least.
	// Otherwise, we will default to minimum of the maxTrials across brackets
	// to guarantee roughly equal work between brackets.
	var minTrials int
	remainder := 0
	numBrackets := len(maxTrials)
	bracketMaxConcurrentTrials := make([]int, 0, numBrackets)
	// Without this, the remainder will be less than numBrackets and later brackets will,
	// not receive a constraint on bracketMaxConcurrentTrials.
	maxConcurrentTrials = min(maxConcurrentTrials, numBrackets)
	if maxConcurrentTrials == 0 {
		minTrials = max(maxTrials[numBrackets-1], int(divisor))
	} else {
		minTrials = maxConcurrentTrials / numBrackets
		remainder = maxConcurrentTrials % numBrackets
	}
	for i := 0; i < numBrackets; i++ {
		bracketMaxConcurrentTrials = append(bracketMaxConcurrentTrials, minTrials)
	}

	for i := 0; i < remainder; i++ {
		bracketMaxConcurrentTrials[i]++
	}
	return bracketMaxConcurrentTrials
}

func newAdaptiveASHASearch(config model.AdaptiveASHAConfig) SearchMethod {
	modeFunc := parseAdaptiveMode(config.Mode)

	brackets := config.BracketRungs
	if len(brackets) == 0 {
		config.MaxRungs = min(
			config.MaxRungs,
			int(math.Log(float64(config.MaxLength.Units))/math.Log(config.Divisor))+1)
		config.MaxRungs = min(
			config.MaxRungs,
			int(math.Log(float64(config.MaxTrials))/math.Log(config.Divisor))+1)
		brackets = modeFunc(config.MaxRungs)
	}
	// We prioritize brackets that perform more early stopping to try to max speedups early on.
	sort.Sort(sort.Reverse(sort.IntSlice(brackets)))
	bracketMaxTrials := getBracketMaxTrials(
		config.MaxTrials, config.Divisor, brackets)
	bracketMaxConcurrentTrials := getBracketMaxConcurrentTrials(
		config.MaxConcurrentTrials, config.Divisor, bracketMaxTrials)

	methods := make([]SearchMethod, 0, len(brackets))
	for i, numRungs := range brackets {
		c := model.AsyncHalvingConfig{
			Metric:              config.Metric,
			SmallerIsBetter:     config.SmallerIsBetter,
			NumRungs:            numRungs,
			MaxLength:           config.MaxLength,
			MaxTrials:           bracketMaxTrials[i],
			Divisor:             config.Divisor,
			MaxConcurrentTrials: bracketMaxConcurrentTrials[i],
		}
		methods = append(methods, newAsyncHalvingSearch(c))
	}

	return newTournamentSearch(methods...)
}
