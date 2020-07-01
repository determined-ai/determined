package searcher

import (
	"testing"

	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/pkg/model"
)

func TestRandomTournamentSearcher(t *testing.T) {
	actual := newTournamentSearch(
		newRandomSearch(model.RandomConfig{
			MaxTrials: 2,
			MaxLength: model.NewLengthInBatches(300),
		}, defaultBatchesPerStep, 0),
		newRandomSearch(model.RandomConfig{
			MaxTrials: 3,
			MaxLength: model.NewLengthInBatches(200),
		}, defaultBatchesPerStep, 0),
	)
	expected := [][]Kind{
		{RunStep, RunStep, RunStep, ComputeValidationMetrics},
		{RunStep, RunStep, RunStep, ComputeValidationMetrics},
		{RunStep, RunStep, ComputeValidationMetrics},
		{RunStep, RunStep, ComputeValidationMetrics},
		{RunStep, RunStep, ComputeValidationMetrics},
	}
	checkSimulation(t, actual, defaultHyperparameters(), ConstantValidation, expected)
}

func TestRandomTournamentSearcherReproducibility(t *testing.T) {
	conf := model.RandomConfig{MaxTrials: 5, MaxLength: model.NewLengthInBatches(8)}
	gen := func() SearchMethod {
		return newTournamentSearch(
			newRandomSearch(conf, defaultBatchesPerStep, 0),
			newRandomSearch(conf, defaultBatchesPerStep, 0),
		)
	}
	checkReproducibility(t, gen, defaultHyperparameters(), defaultMetric)
}

func TestTournamentSearchMethod(t *testing.T) {
	// Run both of the tests from adaptive_test.go side by side.
	expectedTrials := []predefinedTrial{
		newConstantPredefinedTrial(0.1, 32, []int{8, 32}, nil),
		newConstantPredefinedTrial(0.2, 8, []int{8}, nil),
		newConstantPredefinedTrial(0.3, 32, []int{32}, nil),

		newConstantPredefinedTrial(0.3, 32, []int{8, 32}, nil),
		newConstantPredefinedTrial(0.2, 8, []int{8}, nil),
		newConstantPredefinedTrial(0.1, 32, []int{32}, nil),
	}

	adaptiveConfig1 := model.SearcherConfig{
		AdaptiveConfig: &model.AdaptiveConfig{
			Metric:          "error",
			SmallerIsBetter: true,
			MaxLength:       model.NewLengthInBatches(3200),
			Budget:          model.NewLengthInBatches(6400),
			Mode:            model.StandardMode,
			MaxRungs:        2,
			Divisor:         4,
		},
	}
	adaptiveMethod1 := NewSearchMethod(adaptiveConfig1, defaultBatchesPerStep, 0)

	adaptiveConfig2 := model.SearcherConfig{
		AdaptiveConfig: &model.AdaptiveConfig{
			Metric:          "error",
			SmallerIsBetter: false,
			MaxLength:       model.NewLengthInBatches(3200),
			Budget:          model.NewLengthInBatches(6400),
			Mode:            model.StandardMode,
			MaxRungs:        2,
			Divisor:         4,
		},
	}
	adaptiveMethod2 := NewSearchMethod(adaptiveConfig2, defaultBatchesPerStep, 0)

	method := newTournamentSearch(adaptiveMethod1, adaptiveMethod2)

	err := checkValueSimulation(t, method, defaultHyperparameters(), expectedTrials)
	assert.NilError(t, err)
}
