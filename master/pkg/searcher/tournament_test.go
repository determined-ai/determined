package searcher

import (
	"testing"

	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/pkg/model"
)

func TestRandomTournamentSearcher(t *testing.T) {
	actual := newTournamentSearch(
		newRandomSearch(model.RandomConfig{MaxTrials: 2, MaxSteps: 3}),
		newRandomSearch(model.RandomConfig{MaxTrials: 3, MaxSteps: 2}),
	)
	expected := [][]Kind{
		{RunStep, RunStep, RunStep, ComputeValidationMetrics},
		{RunStep, RunStep, RunStep, ComputeValidationMetrics},
		{RunStep, RunStep, ComputeValidationMetrics},
		{RunStep, RunStep, ComputeValidationMetrics},
		{RunStep, RunStep, ComputeValidationMetrics},
	}
	checkSimulation(t, actual, nil, ConstantValidation, expected)
}

func TestRandomTournamentSearcherReproducibility(t *testing.T) {
	conf := model.RandomConfig{MaxTrials: 5, MaxSteps: 8}
	gen := func() SearchMethod {
		return newTournamentSearch(newRandomSearch(conf), newRandomSearch(conf))
	}
	checkReproducibility(t, gen, nil, defaultMetric)
}

func TestTournamentSearchMethod(t *testing.T) {
	// Run both of the tests from adaptive_test.go side by side.
	expectedTrials := []predefinedTrial{
		// Adaptive 1 trials
		newConstantPredefinedTrial(0.1, 32, []int{8, 32}, nil),
		newConstantPredefinedTrial(0.2, 8, []int{8}, nil),
		newConstantPredefinedTrial(0.3, 8, []int{8}, nil),
		newConstantPredefinedTrial(0.4, 8, []int{8}, nil),
		newConstantPredefinedTrial(0.5, 32, []int{32}, nil),

		// Adaptive 2 trials
		newConstantPredefinedTrial(0.6, 32, []int{8, 32}, nil),
		newConstantPredefinedTrial(0.5, 8, []int{8}, nil),
		newConstantPredefinedTrial(0.4, 8, []int{8}, nil),
		newConstantPredefinedTrial(0.3, 8, []int{8}, nil),
		newConstantPredefinedTrial(0.2, 32, []int{32}, nil),

		// Top off adaptive 1 trials
		newConstantPredefinedTrial(0.6, 32, []int{32}, nil),

		// Top off adaptive 2 trials
		newConstantPredefinedTrial(0.1, 32, []int{32}, nil),
	}

	adaptiveConfig1 := model.SearcherConfig{
		AdaptiveConfig: &model.AdaptiveConfig{
			Metric:           "error",
			SmallerIsBetter:  true,
			TargetTrialSteps: 32,
			MaxTrials:        6,
			Mode:             model.StandardMode,
			MaxRungs:         2,
			Divisor:          4,
		},
	}
	adaptiveMethod1 := NewSearchMethod(adaptiveConfig1)

	adaptiveConfig2 := model.SearcherConfig{
		AdaptiveConfig: &model.AdaptiveConfig{
			Metric:           "error",
			SmallerIsBetter:  false,
			TargetTrialSteps: 32,
			MaxTrials:        6,
			Mode:             model.StandardMode,
			MaxRungs:         2,
			Divisor:          4,
		},
	}
	adaptiveMethod2 := NewSearchMethod(adaptiveConfig2)

	params := model.Hyperparameters{}

	method := newTournamentSearch(adaptiveMethod1, adaptiveMethod2)

	err := checkValueSimulation(t, method, params, expectedTrials)
	assert.NilError(t, err)
}
