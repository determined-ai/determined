package searcher

import (
	"testing"

	"github.com/determined-ai/determined/master/pkg/model"
)

func TestAdaptiveSimpleConservativeCornerCase(t *testing.T) {
	actual := model.AdaptiveSimpleConfig{
		Metric: defaultMetric, SmallerIsBetter: true, MaxSteps: 1, MaxTrials: 1,
		Divisor: 4, Mode: model.ConservativeMode, MaxRungs: 3,
	}
	expected := [][]Kind{
		toKinds("1S 1V"),
		toKinds("1S 1V 1S 1V"),
		toKinds("1S 1V 1S 1V 1S 1V"),
	}
	checkSimulation(t, newAdaptiveSimpleSearch(actual), nil, ConstantValidation, expected)
}

func TestAdaptiveSimpleAggressiveCornerCase(t *testing.T) {
	actual := model.AdaptiveSimpleConfig{
		Metric: defaultMetric, SmallerIsBetter: true, MaxSteps: 1, MaxTrials: 1,
		Divisor: 4, Mode: model.AggressiveMode, MaxRungs: 3,
	}
	expected := [][]Kind{
		toKinds("1S 1V 1S 1V 1S 1V"),
	}
	checkSimulation(t, newAdaptiveSimpleSearch(actual), nil, ConstantValidation, expected)
}

func TestAdaptiveSimpleSearcherReproducibility(t *testing.T) {
	conf := model.AdaptiveSimpleConfig{
		Metric: defaultMetric, SmallerIsBetter: true, MaxSteps: 64, MaxTrials: 50,
		Divisor: 4, Mode: model.ConservativeMode, MaxRungs: 3,
	}
	gen := func() SearchMethod { return newAdaptiveSimpleSearch(conf) }
	checkReproducibility(t, gen, nil, defaultMetric)
}
