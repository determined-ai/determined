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
	expected := [][]Runnable{
		toOps("64000R V"), toOps("64000R V"), toOps("64000R V"),
		toOps("64000R V"), toOps("64000R V"), toOps("64000R V"),
		toOps("64000R V"), toOps("64000R V"),
		toOps("64000R V 128000R V"),
		toOps("64000R V 128000R V"),
		toOps("64000R V 128000R V"),
		toOps("64000R V 128000R V 384000R V"),
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
	expected := [][]Runnable{
		toOps("1000B V"), toOps("1000B V"), toOps("1000B V"),
		toOps("1000B V"), toOps("1000B V"), toOps("1000B V"),
		toOps("1000B V"), toOps("1000B V"),
		toOps("1000B V 2000B V"),
		toOps("1000B V 2000B V"),
		toOps("1000B V 2000B V"),
		toOps("1000B V 2000B V 6000B V"),
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
	expected := [][]Runnable{
		toOps("1E V"), toOps("1E V"), toOps("1E V"),
		toOps("1E V"), toOps("1E V"), toOps("1E V"),
		toOps("1E V"), toOps("1E V"),
		toOps("1E V 3E V"),
		toOps("1E V 3E V"),
		toOps("1E V 3E V"),
		toOps("1E V 3E V 8E V"),
	}
	checkSimulation(t, newAsyncHalvingSearch(actual), nil, ConstantValidation, expected)
}

func TestASHASearchMethod(t *testing.T) {
	maxConcurrentTrials := 3
	testCases := []valueSimulationTestCase{
		{
			name: "smaller is better",
			expectedTrials: []predefinedTrial{
				newConstantPredefinedTrial(toOps("1000B V 2000B V 6000B V"), 0.01),
				newConstantPredefinedTrial(toOps("1000B V 2000B V"), 0.02),
				newConstantPredefinedTrial(toOps("1000B V 2000B V"), 0.03),
				newConstantPredefinedTrial(toOps("1000B V 2000B V"), 0.04),
				newConstantPredefinedTrial(toOps("1000B V"), 0.05),
				newConstantPredefinedTrial(toOps("1000B V"), 0.06),
				newConstantPredefinedTrial(toOps("1000B V"), 0.07),
				newConstantPredefinedTrial(toOps("1000B V"), 0.08),
				newConstantPredefinedTrial(toOps("1000B V"), 0.09),
				newConstantPredefinedTrial(toOps("1000B V"), 0.10),
				newConstantPredefinedTrial(toOps("1000B V"), 0.11),
				newConstantPredefinedTrial(toOps("1000B V"), 0.12),
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
				newConstantPredefinedTrial(toOps("1000B V 2000B V 6000B V"), 0.01),
				newConstantPredefinedTrial(toOps("1000B V 2000B V"), 0.02),
				newEarlyExitPredefinedTrial(toOps("1000B V 2000B"), 0.03),
				newConstantPredefinedTrial(toOps("1000B V 2000B V"), 0.04),
				newConstantPredefinedTrial(toOps("1000B V"), 0.05),
				newConstantPredefinedTrial(toOps("1000B V"), 0.06),
				newConstantPredefinedTrial(toOps("1000B V"), 0.07),
				newConstantPredefinedTrial(toOps("1000B V"), 0.08),
				newConstantPredefinedTrial(toOps("1000B V"), 0.09),
				newConstantPredefinedTrial(toOps("1000B V"), 0.10),
				newConstantPredefinedTrial(toOps("1000B V"), 0.11),
				newConstantPredefinedTrial(toOps("1000B V"), 0.12),
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
				newConstantPredefinedTrial(toOps("1000B V 2000B V 6000B V"), 0.12),
				newConstantPredefinedTrial(toOps("1000B V 2000B V"), 0.11),
				newConstantPredefinedTrial(toOps("1000B V 2000B V"), 0.10),
				newConstantPredefinedTrial(toOps("1000B V 2000B V"), 0.09),
				newConstantPredefinedTrial(toOps("1000B V"), 0.08),
				newConstantPredefinedTrial(toOps("1000B V"), 0.07),
				newConstantPredefinedTrial(toOps("1000B V"), 0.06),
				newConstantPredefinedTrial(toOps("1000B V"), 0.05),
				newConstantPredefinedTrial(toOps("1000B V"), 0.04),
				newConstantPredefinedTrial(toOps("1000B V"), 0.03),
				newConstantPredefinedTrial(toOps("1000B V"), 0.02),
				newConstantPredefinedTrial(toOps("1000B V"), 0.01),
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
				newConstantPredefinedTrial(toOps("1000B V 2000B V 6000B V"), 0.12),
				newConstantPredefinedTrial(toOps("1000B V 2000B V"), 0.11),
				newEarlyExitPredefinedTrial(toOps("1000B V 2000B"), 0.10),
				newConstantPredefinedTrial(toOps("1000B V 2000B V"), 0.09),
				newConstantPredefinedTrial(toOps("1000B V"), 0.08),
				newConstantPredefinedTrial(toOps("1000B V"), 0.07),
				newConstantPredefinedTrial(toOps("1000B V"), 0.06),
				newConstantPredefinedTrial(toOps("1000B V"), 0.05),
				newConstantPredefinedTrial(toOps("1000B V"), 0.04),
				newConstantPredefinedTrial(toOps("1000B V"), 0.03),
				newConstantPredefinedTrial(toOps("1000B V"), 0.02),
				newConstantPredefinedTrial(toOps("1000B V"), 0.01),
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
				newConstantPredefinedTrial(toOps("1000B V 2000B V"), 0.10),
				newConstantPredefinedTrial(toOps("1000B V"), 0.11),
				newEarlyExitPredefinedTrial(toOps("1000B V"), 0.12),
				newConstantPredefinedTrial(toOps("1000B V 2000B V 6000B V"), 0.01),
				newConstantPredefinedTrial(toOps("1000B V 2000B V"), 0.02),
				newConstantPredefinedTrial(toOps("1000B V 2000B V"), 0.03),
				newConstantPredefinedTrial(toOps("1000B V 2000B V"), 0.04),
				newConstantPredefinedTrial(toOps("1000B V"), 0.05),
				newConstantPredefinedTrial(toOps("1000B V"), 0.06),
				newConstantPredefinedTrial(toOps("1000B V"), 0.07),
				newConstantPredefinedTrial(toOps("1000B V"), 0.08),
				newConstantPredefinedTrial(toOps("1000B V"), 0.09),
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
	}

	runValueSimulationTestCases(t, testCases)
}
