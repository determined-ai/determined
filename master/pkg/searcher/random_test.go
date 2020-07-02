package searcher

import (
	"testing"

	"github.com/determined-ai/determined/master/pkg/model"
)

func TestRandomSearcherRecords(t *testing.T) {
	actual := model.RandomConfig{MaxTrials: 4, MaxLength: model.NewLengthInRecords(19200)}
	expected := [][]Kind{
		{RunStep, RunStep, RunStep, ComputeValidationMetrics},
		{RunStep, RunStep, RunStep, ComputeValidationMetrics},
		{RunStep, RunStep, RunStep, ComputeValidationMetrics},
		{RunStep, RunStep, RunStep, ComputeValidationMetrics},
	}
	search := newRandomSearch(actual)
	checkSimulation(t, search, defaultHyperparameters(), ConstantValidation, expected, 0)
}

func TestRandomSearcherBatches(t *testing.T) {
	actual := model.RandomConfig{MaxTrials: 4, MaxLength: model.NewLengthInBatches(300)}
	expected := [][]Kind{
		{RunStep, RunStep, RunStep, ComputeValidationMetrics},
		{RunStep, RunStep, RunStep, ComputeValidationMetrics},
		{RunStep, RunStep, RunStep, ComputeValidationMetrics},
		{RunStep, RunStep, RunStep, ComputeValidationMetrics},
	}
	search := newRandomSearch(actual)
	checkSimulation(t, search, defaultHyperparameters(), ConstantValidation, expected, 0)
}

func TestRandomSearcherReproducibility(t *testing.T) {
	conf := model.RandomConfig{MaxTrials: 4, MaxLength: model.NewLengthInBatches(300)}
	gen := func() SearchMethod { return newRandomSearch(conf) }
	checkReproducibility(t, gen, defaultHyperparameters(), defaultMetric)
}

func TestRandomSearchMethod(t *testing.T) {
	testCases := []valueSimulationTestCase{
		{
			name: "test random search method",
			kind: model.Batches,
			expectedTrials: []predefinedTrial{
				newConstantPredefinedTrial(.1, 5, []int{5}, nil),
				newConstantPredefinedTrial(.1, 5, []int{5}, nil),
				newConstantPredefinedTrial(.1, 5, []int{5}, nil),
				newEarlyExitPredefinedTrial(.1, 5, nil, nil),
			},
			config: model.SearcherConfig{
				RandomConfig: &model.RandomConfig{
					MaxLength: model.NewLengthInBatches(500),
					MaxTrials: 4,
				},
			},
			hparams:         defaultHyperparameters(),
			batchesPerStep:  defaultBatchesPerStep,
			recordsPerEpoch: 0,
		},
		{
			name: "test random search method with records",
			kind: model.Records,
			expectedTrials: []predefinedTrial{
				newConstantPredefinedTrial(.1, 5, []int{5}, nil),
				newConstantPredefinedTrial(.1, 5, []int{5}, nil),
				newConstantPredefinedTrial(.1, 5, []int{5}, nil),
				newConstantPredefinedTrial(.1, 5, []int{5}, nil),
			},
			config: model.SearcherConfig{
				RandomConfig: &model.RandomConfig{
					MaxLength: model.NewLengthInRecords(32017),
					MaxTrials: 4,
				},
			},
			hparams:         defaultHyperparameters(),
			batchesPerStep:  defaultBatchesPerStep,
			recordsPerEpoch: 0,
		},
	}

	runValueSimulationTestCases(t, testCases)
}

func TestSingleSearchMethod(t *testing.T) {
	testCases := []valueSimulationTestCase{
		{
			name: "test single search method",
			kind: model.Batches,
			expectedTrials: []predefinedTrial{
				newConstantPredefinedTrial(.1, 5, []int{5}, nil),
			},
			config: model.SearcherConfig{
				SingleConfig: &model.SingleConfig{
					MaxLength: model.NewLengthInBatches(500),
				},
			},
			hparams:         defaultHyperparameters(),
			batchesPerStep:  defaultBatchesPerStep,
			recordsPerEpoch: 0,
		},
	}

	runValueSimulationTestCases(t, testCases)
}
