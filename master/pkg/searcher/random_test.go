package searcher

import (
	"testing"

	"github.com/determined-ai/determined/master/pkg/model"
)

func TestRandomSearcherRecords(t *testing.T) {
	actual := model.RandomConfig{MaxTrials: 4, MaxLength: model.NewLengthInRecords(19200)}
	expected := [][]ValidateAfter{
		toOps("19200R"),
		toOps("19200R"),
		toOps("19200R"),
		toOps("19200R"),
	}
	search := newRandomSearch(actual)
	checkSimulation(t, search, nil, ConstantValidation, expected)
}

func TestRandomSearcherBatches(t *testing.T) {
	actual := model.RandomConfig{MaxTrials: 4, MaxLength: model.NewLengthInBatches(300)}
	expected := [][]ValidateAfter{
		toOps("300B"),
		toOps("300B"),
		toOps("300B"),
		toOps("300B"),
	}
	search := newRandomSearch(actual)
	checkSimulation(t, search, nil, ConstantValidation, expected)
}

func TestRandomSearcherReproducibility(t *testing.T) {
	conf := model.RandomConfig{MaxTrials: 4, MaxLength: model.NewLengthInBatches(300)}
	gen := func() SearchMethod { return newRandomSearch(conf) }
	checkReproducibility(t, gen, nil, defaultMetric)
}

func TestRandomSearchMethod(t *testing.T) {
	testCases := []valueSimulationTestCase{
		{
			name: "test random search method",
			expectedTrials: []predefinedTrial{
				newConstantPredefinedTrial(toOps("500B"), .1),
				newConstantPredefinedTrial(toOps("500B"), .1),
				newConstantPredefinedTrial(toOps("500B"), .1),
				newEarlyExitPredefinedTrial(toOps("500B"), .1),
			},
			config: model.SearcherConfig{
				RandomConfig: &model.RandomConfig{
					MaxLength:           model.NewLengthInBatches(500),
					MaxTrials:           4,
					MaxConcurrentTrials: 2,
				},
			},
		},
		{
			name: "test random search method with records",
			expectedTrials: []predefinedTrial{
				newConstantPredefinedTrial(toOps("32017R"), .1),
				newConstantPredefinedTrial(toOps("32017R"), .1),
				newConstantPredefinedTrial(toOps("32017R"), .1),
				newConstantPredefinedTrial(toOps("32017R"), .1),
			},
			config: model.SearcherConfig{
				RandomConfig: &model.RandomConfig{
					MaxLength: model.NewLengthInRecords(32017),
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
				newConstantPredefinedTrial(toOps("500B"), .1),
			},
			config: model.SearcherConfig{
				SingleConfig: &model.SingleConfig{
					MaxLength: model.NewLengthInBatches(500),
				},
			},
		},
	}

	runValueSimulationTestCases(t, testCases)
}
