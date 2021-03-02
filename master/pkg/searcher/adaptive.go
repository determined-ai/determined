package searcher

import (
	"fmt"
	"sort"

	"github.com/determined-ai/determined/master/pkg/model"
)

func newAdaptiveSearch(config model.AdaptiveConfig) SearchMethod {
	modeFunc := parseAdaptiveMode(config.Mode)

	brackets := config.BracketRungs
	if len(brackets) == 0 {
		brackets = modeFunc(config.MaxRungs)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(brackets)))

	methods := make([]SearchMethod, 0, len(brackets))
	for _, numRungs := range brackets {
		c := model.SyncHalvingConfig{
			Metric:          config.Metric,
			SmallerIsBetter: config.SmallerIsBetter,
			NumRungs:        numRungs,
			MaxLength:       config.MaxLength,
			Budget:          config.Budget.DivInt(len(brackets)),
			Divisor:         config.Divisor,
			TrainStragglers: config.TrainStragglers,
		}
		methods = append(methods, newSyncHalvingSearch(c))
	}

	return newTournamentSearch(AdaptiveSearch, methods...)
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
