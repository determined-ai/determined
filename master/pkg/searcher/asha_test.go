package searcher

import (
	"testing"

	"github.com/determined-ai/determined/master/pkg/model"
)

func TestASHASearcherRecords(t *testing.T) {
	actual := model.AsyncHalvingConfig{
		Metric: defaultMetric, NumRungs: 3,
		MaxLength: model.NewLengthInRecords(576000),
		Divisor:   3,
		MaxTrials: 12,
	}
	expected := [][]ValidateAfter{
		toOps("64000R"), toOps("64000R"), toOps("64000R"),
		toOps("64000R"), toOps("64000R"), toOps("64000R"),
		toOps("64000R"), toOps("64000R"),
		toOps("64000R 192000R"),
		toOps("64000R 192000R"),
		toOps("64000R 192000R"),
		toOps("64000R 192000R 576000R"),
	}
	checkSimulation(t, newAsyncHalvingSearch(actual), nil, ConstantValidation, expected)
}

func TestASHASearcherBatches(t *testing.T) {
	actual := model.AsyncHalvingConfig{
		Metric: defaultMetric, NumRungs: 3,
		MaxLength: model.NewLengthInBatches(9000),
		Divisor:   3,
		MaxTrials: 12,
	}
	expected := [][]ValidateAfter{
		toOps("1000B"), toOps("1000B"), toOps("1000B"),
		toOps("1000B"), toOps("1000B"), toOps("1000B"),
		toOps("1000B"), toOps("1000B"),
		toOps("1000B 3000B"),
		toOps("1000B 3000B"),
		toOps("1000B 3000B"),
		toOps("1000B 3000B 9000B"),
	}
	checkSimulation(t, newAsyncHalvingSearch(actual), nil, ConstantValidation, expected)
}

func TestASHASearcherEpochs(t *testing.T) {
	actual := model.AsyncHalvingConfig{
		Metric: defaultMetric, NumRungs: 3,
		MaxLength: model.NewLengthInEpochs(12),
		Divisor:   3,
		MaxTrials: 12,
	}
	expected := [][]ValidateAfter{
		toOps("1E"), toOps("1E"), toOps("1E"),
		toOps("1E"), toOps("1E"), toOps("1E"),
		toOps("1E"), toOps("1E"),
		toOps("1E 4E"),
		toOps("1E 4E"),
		toOps("1E 4E"),
		toOps("1E 4E 12E"),
	}
	checkSimulation(t, newAsyncHalvingSearch(actual), nil, ConstantValidation, expected)
}

func TestASHASearchMethod(t *testing.T) {
	maxConcurrentTrials := 3
	testCases := []valueSimulationTestCase{
		{
			name: "smaller is better",
			expectedTrials: []predefinedTrial{
				newConstantPredefinedTrial(toOps("1000B 3000B 9000B"), 0.01),
				newConstantPredefinedTrial(toOps("1000B 3000B"), 0.02),
				newConstantPredefinedTrial(toOps("1000B 3000B"), 0.03),
				newConstantPredefinedTrial(toOps("1000B 3000B"), 0.04),
				newConstantPredefinedTrial(toOps("1000B"), 0.05),
				newConstantPredefinedTrial(toOps("1000B"), 0.06),
				newConstantPredefinedTrial(toOps("1000B"), 0.07),
				newConstantPredefinedTrial(toOps("1000B"), 0.08),
				newConstantPredefinedTrial(toOps("1000B"), 0.09),
				newConstantPredefinedTrial(toOps("1000B"), 0.10),
				newConstantPredefinedTrial(toOps("1000B"), 0.11),
				newConstantPredefinedTrial(toOps("1000B"), 0.12),
			},
			config: model.SearcherConfig{
				AsyncHalvingConfig: &model.AsyncHalvingConfig{
					Metric:              "error",
					NumRungs:            3,
					SmallerIsBetter:     true,
					MaxLength:           model.NewLengthInBatches(9000),
					MaxTrials:           12,
					Divisor:             3,
					MaxConcurrentTrials: maxConcurrentTrials,
				},
			},
		},
		{
			name: "early exit -- smaller is better",
			expectedTrials: []predefinedTrial{
				newConstantPredefinedTrial(toOps("1000B 3000B 9000B"), 0.01),
				newConstantPredefinedTrial(toOps("1000B 3000B"), 0.02),
				newEarlyExitPredefinedTrial(toOps("1000B 3000B"), 0.03),
				newConstantPredefinedTrial(toOps("1000B 3000B"), 0.04),
				newConstantPredefinedTrial(toOps("1000B"), 0.05),
				newConstantPredefinedTrial(toOps("1000B"), 0.06),
				newConstantPredefinedTrial(toOps("1000B"), 0.07),
				newConstantPredefinedTrial(toOps("1000B"), 0.08),
				newConstantPredefinedTrial(toOps("1000B"), 0.09),
				newConstantPredefinedTrial(toOps("1000B"), 0.10),
				newConstantPredefinedTrial(toOps("1000B"), 0.11),
				newConstantPredefinedTrial(toOps("1000B"), 0.12),
			},
			config: model.SearcherConfig{
				AsyncHalvingConfig: &model.AsyncHalvingConfig{
					Metric:              "error",
					NumRungs:            3,
					SmallerIsBetter:     true,
					MaxLength:           model.NewLengthInBatches(9000),
					MaxTrials:           12,
					Divisor:             3,
					MaxConcurrentTrials: maxConcurrentTrials,
				},
			},
		},
		{
			name: "smaller is not better",
			expectedTrials: []predefinedTrial{
				newConstantPredefinedTrial(toOps("1000B 3000B 9000B"), 0.12),
				newConstantPredefinedTrial(toOps("1000B 3000B"), 0.11),
				newConstantPredefinedTrial(toOps("1000B 3000B"), 0.10),
				newConstantPredefinedTrial(toOps("1000B 3000B"), 0.09),
				newConstantPredefinedTrial(toOps("1000B"), 0.08),
				newConstantPredefinedTrial(toOps("1000B"), 0.07),
				newConstantPredefinedTrial(toOps("1000B"), 0.06),
				newConstantPredefinedTrial(toOps("1000B"), 0.05),
				newConstantPredefinedTrial(toOps("1000B"), 0.04),
				newConstantPredefinedTrial(toOps("1000B"), 0.03),
				newConstantPredefinedTrial(toOps("1000B"), 0.02),
				newConstantPredefinedTrial(toOps("1000B"), 0.01),
			},
			config: model.SearcherConfig{
				AsyncHalvingConfig: &model.AsyncHalvingConfig{
					Metric:              "error",
					NumRungs:            3,
					SmallerIsBetter:     false,
					MaxLength:           model.NewLengthInBatches(9000),
					MaxTrials:           12,
					Divisor:             3,
					MaxConcurrentTrials: maxConcurrentTrials,
				},
			},
		},
		{
			name: "early exit -- smaller is not better",
			expectedTrials: []predefinedTrial{
				newConstantPredefinedTrial(toOps("1000B 3000B 9000B"), 0.12),
				newConstantPredefinedTrial(toOps("1000B 3000B"), 0.11),
				newEarlyExitPredefinedTrial(toOps("1000B 3000B"), 0.10),
				newConstantPredefinedTrial(toOps("1000B 3000B"), 0.09),
				newConstantPredefinedTrial(toOps("1000B"), 0.08),
				newConstantPredefinedTrial(toOps("1000B"), 0.07),
				newConstantPredefinedTrial(toOps("1000B"), 0.06),
				newConstantPredefinedTrial(toOps("1000B"), 0.05),
				newConstantPredefinedTrial(toOps("1000B"), 0.04),
				newConstantPredefinedTrial(toOps("1000B"), 0.03),
				newConstantPredefinedTrial(toOps("1000B"), 0.02),
				newConstantPredefinedTrial(toOps("1000B"), 0.01),
			},
			config: model.SearcherConfig{
				AsyncHalvingConfig: &model.AsyncHalvingConfig{
					Metric:              "error",
					NumRungs:            3,
					SmallerIsBetter:     false,
					MaxLength:           model.NewLengthInBatches(9000),
					MaxTrials:           12,
					Divisor:             3,
					MaxConcurrentTrials: maxConcurrentTrials,
				},
			},
		},
		{
			name: "async promotions",
			expectedTrials: []predefinedTrial{
				// The first trial is promoted due to asynchronous
				// promotions despite being below top 1/3 of trials in
				// base rung.
				newConstantPredefinedTrial(toOps("1000B 3000B"), 0.10),
				newConstantPredefinedTrial(toOps("1000B"), 0.11),
				newEarlyExitPredefinedTrial(toOps("1000B"), 0.12),
				newConstantPredefinedTrial(toOps("1000B 3000B 9000B"), 0.01),
				newConstantPredefinedTrial(toOps("1000B 3000B"), 0.02),
				newConstantPredefinedTrial(toOps("1000B 3000B"), 0.03),
				newConstantPredefinedTrial(toOps("1000B 3000B"), 0.04),
				newConstantPredefinedTrial(toOps("1000B"), 0.05),
				newConstantPredefinedTrial(toOps("1000B"), 0.06),
				newConstantPredefinedTrial(toOps("1000B"), 0.07),
				newConstantPredefinedTrial(toOps("1000B"), 0.08),
				newConstantPredefinedTrial(toOps("1000B"), 0.09),
			},
			config: model.SearcherConfig{
				AsyncHalvingConfig: &model.AsyncHalvingConfig{
					Metric:              "error",
					NumRungs:            3,
					SmallerIsBetter:     true,
					MaxLength:           model.NewLengthInBatches(9000),
					MaxTrials:           12,
					Divisor:             3,
					MaxConcurrentTrials: maxConcurrentTrials,
				},
			},
		},
		{
			name: "single rung bracket",
			expectedTrials: []predefinedTrial{
				// The first trial is promoted due to asynchronous
				// promotions despite being below top 1/3 of trials in
				// base rung.
				newConstantPredefinedTrial(toOps("9000B"), 0.05),
				newConstantPredefinedTrial(toOps("9000B"), 0.06),
				newConstantPredefinedTrial(toOps("9000B"), 0.07),
				newConstantPredefinedTrial(toOps("9000B"), 0.08),
			},
			config: model.SearcherConfig{
				AsyncHalvingConfig: &model.AsyncHalvingConfig{
					Metric:              "error",
					NumRungs:            1,
					SmallerIsBetter:     true,
					MaxLength:           model.NewLengthInBatches(9000),
					MaxTrials:           4,
					Divisor:             3,
					MaxConcurrentTrials: maxConcurrentTrials,
				},
			},
		},
	}

	runValueSimulationTestCases(t, testCases)
}
