package searcher

import (
	"testing"

	"github.com/determined-ai/determined/master/pkg/model"
)

func TestASHAStoppingSearcherRecords(t *testing.T) {
	actual := model.AsyncHalvingConfig{
		Metric: defaultMetric, NumRungs: 3,
		MaxLength:           model.NewLengthInRecords(576000),
		SmallerIsBetter:     true,
		Divisor:             3,
		MaxTrials:           12,
		StopOnce:            true,
		MaxConcurrentTrials: 2,
	}
	// Stopping-based ASHA will only promote if a trial is in top 1/3 of trials in the rung or if
	// there have been no promotions so far.  Since trials cannot be restarted and metrics increase
	// for later trials, only the first trial will be promoted and all others will be stopped on
	// the first rung.  See continueTraining method in asha_stopping.go for the logic.
	expected := [][]Runnable{
		toOps("64000R V 128000R V 384000R V"),
		toOps("64000R V"), toOps("64000R V"), toOps("64000R V"),
		toOps("64000R V"), toOps("64000R V"), toOps("64000R V"),
		toOps("64000R V"), toOps("64000R V"), toOps("64000R V"),
		toOps("64000R V"), toOps("64000R V"),
	}
	checkSimulation(t, newAsyncHalvingStoppingSearch(actual), nil, TrialIDMetric, expected)
}

func TestASHAStoppingSearcherBatches(t *testing.T) {
	actual := model.AsyncHalvingConfig{
		Metric: defaultMetric, NumRungs: 3,
		MaxLength:           model.NewLengthInBatches(9000),
		SmallerIsBetter:     true,
		Divisor:             3,
		MaxTrials:           12,
		StopOnce:            true,
		MaxConcurrentTrials: 2,
	}
	expected := [][]Runnable{
		toOps("1000B V 2000B V 6000B V"),
		toOps("1000B V"), toOps("1000B V"), toOps("1000B V"),
		toOps("1000B V"), toOps("1000B V"), toOps("1000B V"),
		toOps("1000B V"), toOps("1000B V"), toOps("1000B V"),
		toOps("1000B V"), toOps("1000B V"),
	}
	checkSimulation(t, newAsyncHalvingStoppingSearch(actual), nil, TrialIDMetric, expected)
}

func TestASHAStoppingSearcherEpochs(t *testing.T) {
	actual := model.AsyncHalvingConfig{
		Metric: defaultMetric, NumRungs: 3,
		MaxLength:           model.NewLengthInEpochs(12),
		SmallerIsBetter:     true,
		Divisor:             3,
		MaxTrials:           12,
		StopOnce:            true,
		MaxConcurrentTrials: 2,
	}
	expected := [][]Runnable{
		toOps("1E V 3E V 8E V"),
		toOps("1E V"), toOps("1E V"), toOps("1E V"),
		toOps("1E V"), toOps("1E V"), toOps("1E V"),
		toOps("1E V"), toOps("1E V"), toOps("1E V"),
		toOps("1E V"), toOps("1E V"),
	}
	checkSimulation(t, newAsyncHalvingStoppingSearch(actual), nil, TrialIDMetric, expected)
}

func TestASHAStoppingSearchMethod(t *testing.T) {
	maxConcurrentTrials := 3
	testCases := []valueSimulationTestCase{
		{
			name: "smaller is better",
			expectedTrials: []predefinedTrial{
				newConstantPredefinedTrial(toOps("1000B V 2000B V 6000B V"), 0.01),
				newConstantPredefinedTrial(toOps("1000B V"), 0.02),
				newConstantPredefinedTrial(toOps("1000B V"), 0.03),
				newConstantPredefinedTrial(toOps("1000B V"), 0.04),
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
					StopOnce:            true,
				},
			},
		},
		{
			name: "smaller is better (round robin)",
			expectedTrials: []predefinedTrial{
				newConstantPredefinedTrial(toOps("1000B V 2000B V 6000B V"), 0.01),
				newConstantPredefinedTrial(toOps("1000B V"), 0.02),
				newConstantPredefinedTrial(toOps("1000B V"), 0.03),
				newConstantPredefinedTrial(toOps("1000B V"), 0.04),
				newConstantPredefinedTrial(toOps("1000B V 2000B V 6000B V"), 0.01),
				newConstantPredefinedTrial(toOps("1000B V"), 0.02),
				newConstantPredefinedTrial(toOps("1000B V"), 0.03),
				newConstantPredefinedTrial(toOps("1000B V"), 0.04),
				newConstantPredefinedTrial(toOps("1000B V 2000B V 6000B V"), 0.01),
				newConstantPredefinedTrial(toOps("1000B V"), 0.02),
				newConstantPredefinedTrial(toOps("1000B V"), 0.03),
				newConstantPredefinedTrial(toOps("1000B V"), 0.04),
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
					StopOnce:            true,
				},
			},
		},
		{
			name: "smaller is not better",
			expectedTrials: []predefinedTrial{
				newConstantPredefinedTrial(toOps("1000B V 2000B V 6000B V"), 0.01),
				newConstantPredefinedTrial(toOps("1000B V 2000B V 6000B V"), 0.02),
				newConstantPredefinedTrial(toOps("1000B V 2000B V 6000B V"), 0.03),
				newConstantPredefinedTrial(toOps("1000B V 2000B V 6000B V"), 0.04),
				newConstantPredefinedTrial(toOps("1000B V 2000B V 6000B V"), 0.05),
				newConstantPredefinedTrial(toOps("1000B V 2000B V 6000B V"), 0.06),
				newConstantPredefinedTrial(toOps("1000B V 2000B V 6000B V"), 0.07),
				newConstantPredefinedTrial(toOps("1000B V 2000B V 6000B V"), 0.08),
				newConstantPredefinedTrial(toOps("1000B V 2000B V 6000B V"), 0.09),
				newConstantPredefinedTrial(toOps("1000B V 2000B V 6000B V"), 0.10),
				newConstantPredefinedTrial(toOps("1000B V 2000B V 6000B V"), 0.11),
				newConstantPredefinedTrial(toOps("1000B V 2000B V 6000B V"), 0.12),
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
					StopOnce:            true,
				},
			},
		},
		{
			name: "smaller is not better (round robin)",
			expectedTrials: []predefinedTrial{
				newConstantPredefinedTrial(toOps("1000B V 2000B V 6000B V"), 0.01),
				newConstantPredefinedTrial(toOps("1000B V 2000B V 6000B V"), 0.02),
				newConstantPredefinedTrial(toOps("1000B V 2000B V 6000B V"), 0.03),
				newConstantPredefinedTrial(toOps("1000B V 2000B V 6000B V"), 0.04),
				newConstantPredefinedTrial(toOps("1000B V"), 0.01),
				newConstantPredefinedTrial(toOps("1000B V"), 0.02),
				newConstantPredefinedTrial(toOps("1000B V 2000B V"), 0.03),
				newConstantPredefinedTrial(toOps("1000B V 2000B V 6000B V"), 0.04),
				newConstantPredefinedTrial(toOps("1000B V"), 0.01),
				newConstantPredefinedTrial(toOps("1000B V"), 0.02),
				newConstantPredefinedTrial(toOps("1000B V 2000B V"), 0.03),
				newConstantPredefinedTrial(toOps("1000B V 2000B V 6000B V"), 0.04),
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
					StopOnce:            true,
				},
			},
		},
		{
			name: "early exit -- smaller is better (round robin)",
			expectedTrials: []predefinedTrial{
				newConstantPredefinedTrial(toOps("1000B V 2000B V 6000B V"), 0.01),
				newConstantPredefinedTrial(toOps("1000B V"), 0.02),
				newConstantPredefinedTrial(toOps("1000B V"), 0.03),
				newConstantPredefinedTrial(toOps("1000B V"), 0.04),
				newEarlyExitPredefinedTrial(toOps("1000B V 2000B"), 0.01),
				newConstantPredefinedTrial(toOps("1000B V"), 0.02),
				newConstantPredefinedTrial(toOps("1000B V"), 0.03),
				newConstantPredefinedTrial(toOps("1000B V"), 0.04),
				newConstantPredefinedTrial(toOps("1000B V 2000B V 6000B V"), 0.01),
				newConstantPredefinedTrial(toOps("1000B V"), 0.02),
				newConstantPredefinedTrial(toOps("1000B V"), 0.03),
				newConstantPredefinedTrial(toOps("1000B V"), 0.04),
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
					StopOnce:            true,
				},
			},
		},
		{
			name: "early exit -- smaller is not better (round robin)",
			expectedTrials: []predefinedTrial{
				newConstantPredefinedTrial(toOps("1000B V 2000B V 6000B V"), 0.01),
				newConstantPredefinedTrial(toOps("1000B V 2000B V 6000B V"), 0.02),
				newConstantPredefinedTrial(toOps("1000B V 2000B V 6000B V"), 0.03),
				newEarlyExitPredefinedTrial(toOps("1000B V 2000B"), 0.04),
				newConstantPredefinedTrial(toOps("1000B V"), 0.01),
				newConstantPredefinedTrial(toOps("1000B V"), 0.02),
				newConstantPredefinedTrial(toOps("1000B V 2000B V 6000B V"), 0.03),
				newConstantPredefinedTrial(toOps("1000B V 2000B V 6000B V"), 0.04),
				newConstantPredefinedTrial(toOps("1000B V"), 0.01),
				newConstantPredefinedTrial(toOps("1000B V"), 0.02),
				newConstantPredefinedTrial(toOps("1000B V 2000B V 6000B V"), 0.03),
				newConstantPredefinedTrial(toOps("1000B V 2000B V 6000B V"), 0.04),
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
					StopOnce:            true,
				},
			},
		},
		{
			name: "single rung bracket",
			expectedTrials: []predefinedTrial{
				// The first trial is promoted due to asynchronous
				// promotions despite being below top 1/3 of trials in
				// base rung.
				newConstantPredefinedTrial(toOps("9000B V"), 0.05),
				newConstantPredefinedTrial(toOps("9000B V"), 0.06),
				newConstantPredefinedTrial(toOps("9000B V"), 0.07),
				newConstantPredefinedTrial(toOps("9000B V"), 0.08),
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
					StopOnce:            true,
				},
			},
		},
	}

	runValueSimulationTestCases(t, testCases)
}
