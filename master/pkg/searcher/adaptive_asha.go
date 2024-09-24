package searcher

import (
	"fmt"
	"math"
	"sort"

	"github.com/determined-ai/determined/master/pkg/mathx"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
)

type bracket struct {
	numRungs            int
	maxTrials           int
	maxConcurrentTrials int
}

func (b *bracket) String() string {
	return fmt.Sprintf("Bracket(%d, %d, %d)", b.numRungs, b.maxTrials, b.maxConcurrentTrials)
}

func getBracketMaxTrials(
	maxTrials int, divisor float64, brackets []int,
) []int {
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
			bracketTrials, mathx.Max(int(bracketWeight[i]/totalWeight*float64(maxTrials)), 1))

		allocated += bracketTrials[i]
	}
	remainder := mathx.Max(maxTrials-allocated, 0)
	bracketTrials[0] += remainder
	return bracketTrials
}

func getBracketMaxConcurrentTrials(
	maxConcurrentTrials int, divisor float64, maxTrials []int,
) []int {
	// If maxConcurrentTrials is provided, we will split that evenly across brackets
	// and fill remainder from most aggressive early stopping bracket to least.
	// Otherwise, we will default to minimum of the maxTrials across brackets
	// to guarantee roughly equal work between brackets.
	var minTrials int
	remainder := 0
	numBrackets := len(maxTrials)
	bracketMaxConcurrentTrials := make([]int, 0, numBrackets)
	if maxConcurrentTrials == 0 {
		minTrials = mathx.Max(maxTrials[numBrackets-1], int(divisor))
	} else {
		// Without this, the remainder will be less than numBrackets and later brackets will
		// not receive a constraint on bracketMaxConcurrentTrials.
		maxConcurrentTrials = mathx.Max(maxConcurrentTrials, numBrackets)
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

func makeBrackets(config expconf.AdaptiveASHAConfig) []bracket {
	modeFunc := parseAdaptiveMode(config.Mode())

	bracketRungs := config.BracketRungs()
	if len(bracketRungs) == 0 {
		maxRungs := config.MaxRungs()
		maxRungs = mathx.Min(
			maxRungs,
			int(math.Log(float64(config.Length().Units))/math.Log(config.Divisor()))+1,
			int(math.Log(float64(config.MaxTrials()))/math.Log(config.Divisor()))+1)
		bracketRungs = modeFunc(maxRungs)
	}
	// We prioritize bracketRungs that perform more early stopping to try to max speedups early on.
	sort.Sort(sort.Reverse(sort.IntSlice(bracketRungs)))
	bracketMaxTrials := getBracketMaxTrials(
		config.MaxTrials(), config.Divisor(), bracketRungs)
	bracketMaxConcurrentTrials := getBracketMaxConcurrentTrials(
		config.MaxConcurrentTrials(), config.Divisor(), bracketMaxTrials)

	brackets := make([]bracket, len(bracketRungs))
	for i, bracketRung := range bracketRungs {
		brackets[i] = bracket{
			numRungs:            bracketRung,
			maxTrials:           bracketMaxTrials[i],
			maxConcurrentTrials: bracketMaxConcurrentTrials[i],
		}
	}
	return brackets
}

func newAdaptiveASHASearch(config expconf.AdaptiveASHAConfig, smallerIsBetter bool, metric string) SearchMethod {
	brackets := makeBrackets(config)
	fmt.Printf("Brackets: %v\n", brackets)
	methods := make([]SearchMethod, 0, len(brackets))
	for _, bracket := range brackets {
		c := expconf.AsyncHalvingConfig{
			RawNumRungs:            ptrs.Ptr(bracket.numRungs),
			RawMaxLength:           config.RawMaxLength,
			RawMaxTrials:           &bracket.maxTrials,
			RawDivisor:             ptrs.Ptr(config.Divisor()),
			RawMaxConcurrentTrials: ptrs.Ptr(bracket.maxConcurrentTrials),
			RawStopOnce:            ptrs.Ptr(config.StopOnce()),
			RawTimeMetric:          config.RawTimeMetric,
			RawMaxTime:             config.RawMaxTime,
		}
		methods = append(methods, newAsyncHalvingStoppingSearch(c, smallerIsBetter, metric))
	}

	return newTournamentSearch(AdaptiveASHASearch, methods...)
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

func parseAdaptiveMode(rawMode expconf.AdaptiveMode) adaptiveMode {
	switch rawMode {
	case expconf.ConservativeMode:
		return conservativeMode
	case expconf.StandardMode:
		return standardMode
	case expconf.AggressiveMode:
		return aggressiveMode
	default:
		panic(fmt.Sprintf("unexpected adaptive mode: %s", rawMode))
	}
}
