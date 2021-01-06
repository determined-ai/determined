package searcher

import (
	"testing"

	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
)

func TestASHASearcherRecords(t *testing.T) {
	config := expconf.AsyncHalvingConfig{
		Metric: defaultMetric, NumRungs: 3,
		MaxLength: expconf.NewLengthInRecords(576000),
		Divisor:   ptrs.Float64Ptr(3),
		MaxTrials: 12,
	}
	schemas.FillDefaults(&config)
	expected := [][]Runnable{
		toOps("64000R V"), toOps("64000R V"), toOps("64000R V"),
		toOps("64000R V"), toOps("64000R V"), toOps("64000R V"),
		toOps("64000R V"), toOps("64000R V"),
		toOps("64000R V 128000R V"),
		toOps("64000R V 128000R V"),
		toOps("64000R V 128000R V"),
		toOps("64000R V 128000R V 384000R V"),
	}
	checkSimulation(t, newAsyncHalvingSearch(config), nil, ConstantValidation, expected)
}

func TestASHASearcherBatches(t *testing.T) {
	config := expconf.AsyncHalvingConfig{
		Metric: defaultMetric, NumRungs: 3,
		MaxLength: expconf.NewLengthInBatches(9000),
		Divisor:   ptrs.Float64Ptr(3),
		MaxTrials: 12,
	}
	schemas.FillDefaults(&config)
	expected := [][]Runnable{
		toOps("1000B V"), toOps("1000B V"), toOps("1000B V"),
		toOps("1000B V"), toOps("1000B V"), toOps("1000B V"),
		toOps("1000B V"), toOps("1000B V"),
		toOps("1000B V 2000B V"),
		toOps("1000B V 2000B V"),
		toOps("1000B V 2000B V"),
		toOps("1000B V 2000B V 6000B V"),
	}
	checkSimulation(t, newAsyncHalvingSearch(config), nil, ConstantValidation, expected)
}

func TestASHASearcherEpochs(t *testing.T) {
	config := expconf.AsyncHalvingConfig{
		Metric: defaultMetric, NumRungs: 3,
		MaxLength: expconf.NewLengthInEpochs(12),
		Divisor:   ptrs.Float64Ptr(3),
		MaxTrials: 12,
	}
	schemas.FillDefaults(&config)
	expected := [][]Runnable{
		toOps("1E V"), toOps("1E V"), toOps("1E V"),
		toOps("1E V"), toOps("1E V"), toOps("1E V"),
		toOps("1E V"), toOps("1E V"),
		toOps("1E V 3E V"),
		toOps("1E V 3E V"),
		toOps("1E V 3E V"),
		toOps("1E V 3E V 8E V"),
	}
	checkSimulation(t, newAsyncHalvingSearch(config), nil, ConstantValidation, expected)
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
			config: expconf.SearcherConfig{
				AsyncHalvingConfig: &expconf.AsyncHalvingConfig{
					Metric:              "error",
					NumRungs:            3,
					SmallerIsBetter:     ptrs.BoolPtr(true),
					MaxLength:           expconf.NewLengthInBatches(9000),
					MaxTrials:           12,
					Divisor:             ptrs.Float64Ptr(3),
					MaxConcurrentTrials: ptrs.IntPtr(maxConcurrentTrials),
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
			config: expconf.SearcherConfig{
				AsyncHalvingConfig: &expconf.AsyncHalvingConfig{
					Metric:              "error",
					NumRungs:            3,
					SmallerIsBetter:     ptrs.BoolPtr(true),
					MaxLength:           expconf.NewLengthInBatches(9000),
					MaxTrials:           12,
					Divisor:             ptrs.Float64Ptr(3),
					MaxConcurrentTrials: ptrs.IntPtr(maxConcurrentTrials),
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
			config: expconf.SearcherConfig{
				AsyncHalvingConfig: &expconf.AsyncHalvingConfig{
					Metric:              "error",
					NumRungs:            3,
					SmallerIsBetter:     ptrs.BoolPtr(false),
					MaxLength:           expconf.NewLengthInBatches(9000),
					MaxTrials:           12,
					Divisor:             ptrs.Float64Ptr(3),
					MaxConcurrentTrials: ptrs.IntPtr(maxConcurrentTrials),
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
			config: expconf.SearcherConfig{
				AsyncHalvingConfig: &expconf.AsyncHalvingConfig{
					Metric:              "error",
					NumRungs:            3,
					SmallerIsBetter:     ptrs.BoolPtr(false),
					MaxLength:           expconf.NewLengthInBatches(9000),
					MaxTrials:           12,
					Divisor:             ptrs.Float64Ptr(3),
					MaxConcurrentTrials: ptrs.IntPtr(maxConcurrentTrials),
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
			config: expconf.SearcherConfig{
				AsyncHalvingConfig: &expconf.AsyncHalvingConfig{
					Metric:              "error",
					NumRungs:            3,
					SmallerIsBetter:     ptrs.BoolPtr(true),
					MaxLength:           expconf.NewLengthInBatches(9000),
					MaxTrials:           12,
					Divisor:             ptrs.Float64Ptr(3),
					MaxConcurrentTrials: ptrs.IntPtr(maxConcurrentTrials),
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
			config: expconf.SearcherConfig{
				AsyncHalvingConfig: &expconf.AsyncHalvingConfig{
					Metric:              "error",
					NumRungs:            1,
					SmallerIsBetter:     ptrs.BoolPtr(true),
					MaxLength:           expconf.NewLengthInBatches(9000),
					MaxTrials:           4,
					Divisor:             ptrs.Float64Ptr(3),
					MaxConcurrentTrials: ptrs.IntPtr(maxConcurrentTrials),
				},
			},
		},
	}

	runValueSimulationTestCases(t, testCases)
}
