package searcher

import (
	"testing"

	"github.com/determined-ai/determined/master/pkg/model"
)

func TestRandomSearcher(t *testing.T) {
	actual := model.RandomConfig{MaxTrials: 4, MaxSteps: 3}
	expected := [][]Kind{
		{RunStep, RunStep, RunStep, ComputeValidationMetrics},
		{RunStep, RunStep, RunStep, ComputeValidationMetrics},
		{RunStep, RunStep, RunStep, ComputeValidationMetrics},
		{RunStep, RunStep, RunStep, ComputeValidationMetrics},
	}
	checkSimulation(t, newRandomSearch(actual), nil, ConstantValidation, expected)
}

func TestRandomSearcherReproducibility(t *testing.T) {
	conf := model.RandomConfig{MaxTrials: 4, MaxSteps: 3}
	gen := func() SearchMethod { return newRandomSearch(conf) }
	checkReproducibility(t, gen, nil, defaultMetric)
}

func TestRandomSearchMethod(t *testing.T) {
	testCases := []valueSimulationTestCase{
		{
			name: "test random search method",
			expectedTrials: []predefinedTrial{
				newConstantPredefinedTrial(.1, 5, []int{5}, nil),
				newConstantPredefinedTrial(.1, 5, []int{5}, nil),
				newConstantPredefinedTrial(.1, 5, []int{5}, nil),
				newEarlyExitPredefinedTrial(.1, 5, nil, nil),
			},
			config: model.SearcherConfig{
				RandomConfig: &model.RandomConfig{
					MaxSteps:  5,
					MaxTrials: 4,
				},
			},
		},
	}

	runValueSimulationTestCases(t, testCases)
}

func TestSingleSearchMethod(t *testing.T) {
	testCases := []valueSimulationTestCase{
		{
			name: "test single search method",
			expectedTrials: []predefinedTrial{
				newConstantPredefinedTrial(.1, 5, []int{5}, nil),
			},
			config: model.SearcherConfig{
				SingleConfig: &model.SingleConfig{
					MaxSteps: 5,
				},
			},
		},
	}

	runValueSimulationTestCases(t, testCases)
}
