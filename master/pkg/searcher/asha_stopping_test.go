package searcher

import (
	"testing"

	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
)

func TestASHAStoppingSearcherRecords(t *testing.T) {
	actual := expconf.AsyncHalvingConfig{
		RawNumRungs:            ptrs.IntPtr(3),
		RawMaxLength:           lengthPtr(expconf.NewLengthInRecords(576000)),
		RawDivisor:             ptrs.Float64Ptr(3),
		RawMaxTrials:           ptrs.IntPtr(12),
		RawStopOnce:            ptrs.BoolPtr(true),
		RawMaxConcurrentTrials: ptrs.IntPtr(2),
	}
	actual = schemas.WithDefaults(actual).(expconf.AsyncHalvingConfig)
	// Stopping-based ASHA will only promote if a trial is in top 1/3 of trials in the rung or if
	// there have been no promotions so far.  Since trials cannot be restarted and metrics increase
	// for later trials, only the first trial will be promoted and all others will be stopped on
	// the first rung.  See continueTraining method in asha_stopping.go for the logic.
	expected := [][]ValidateAfter{
		toOps("64000R 192000R 576000R"),
		toOps("64000R"), toOps("64000R"), toOps("64000R"),
		toOps("64000R"), toOps("64000R"), toOps("64000R"),
		toOps("64000R"), toOps("64000R"), toOps("64000R"),
		toOps("64000R"), toOps("64000R"),
	}
	checkSimulation(t, newAsyncHalvingStoppingSearch(actual, true), nil, TrialIDMetric, expected)
}

func TestASHAStoppingSearcherBatches(t *testing.T) {
	actual := expconf.AsyncHalvingConfig{
		RawNumRungs:            ptrs.IntPtr(3),
		RawMaxLength:           lengthPtr(expconf.NewLengthInBatches(9000)),
		RawDivisor:             ptrs.Float64Ptr(3),
		RawMaxTrials:           ptrs.IntPtr(12),
		RawStopOnce:            ptrs.BoolPtr(true),
		RawMaxConcurrentTrials: ptrs.IntPtr(2),
	}
	actual = schemas.WithDefaults(actual).(expconf.AsyncHalvingConfig)
	expected := [][]ValidateAfter{
		toOps("1000B 3000B 9000B"),
		toOps("1000B"), toOps("1000B"), toOps("1000B"),
		toOps("1000B"), toOps("1000B"), toOps("1000B"),
		toOps("1000B"), toOps("1000B"), toOps("1000B"),
		toOps("1000B"), toOps("1000B"),
	}
	checkSimulation(t, newAsyncHalvingStoppingSearch(actual, true), nil, TrialIDMetric, expected)
}

func TestASHAStoppingSearcherEpochs(t *testing.T) {
	actual := expconf.AsyncHalvingConfig{
		RawNumRungs:            ptrs.IntPtr(3),
		RawMaxLength:           lengthPtr(expconf.NewLengthInEpochs(12)),
		RawDivisor:             ptrs.Float64Ptr(3),
		RawMaxTrials:           ptrs.IntPtr(12),
		RawStopOnce:            ptrs.BoolPtr(true),
		RawMaxConcurrentTrials: ptrs.IntPtr(2),
	}
	actual = schemas.WithDefaults(actual).(expconf.AsyncHalvingConfig)
	expected := [][]ValidateAfter{
		toOps("1E 4E 12E"),
		toOps("1E"), toOps("1E"), toOps("1E"),
		toOps("1E"), toOps("1E"), toOps("1E"),
		toOps("1E"), toOps("1E"), toOps("1E"),
		toOps("1E"), toOps("1E"),
	}
	checkSimulation(t, newAsyncHalvingStoppingSearch(actual, true), nil, TrialIDMetric, expected)
}

func TestASHAStoppingSearchMethod(t *testing.T) {
	testCases := []valueSimulationTestCase{
		{
			name: "smaller is better",
			expectedTrials: []predefinedTrial{
				newConstantPredefinedTrial(toOps("1000B 3000B 9000B"), 0.01),
				newConstantPredefinedTrial(toOps("1000B"), 0.02),
				newConstantPredefinedTrial(toOps("1000B"), 0.03),
				newConstantPredefinedTrial(toOps("1000B"), 0.04),
				newConstantPredefinedTrial(toOps("1000B"), 0.05),
				newConstantPredefinedTrial(toOps("1000B"), 0.06),
				newConstantPredefinedTrial(toOps("1000B"), 0.07),
				newConstantPredefinedTrial(toOps("1000B"), 0.08),
				newConstantPredefinedTrial(toOps("1000B"), 0.09),
				newConstantPredefinedTrial(toOps("1000B"), 0.10),
				newConstantPredefinedTrial(toOps("1000B"), 0.11),
				newConstantPredefinedTrial(toOps("1000B"), 0.12),
			},
			config: expconf.SearcherConfig{
				RawSmallerIsBetter: ptrs.BoolPtr(true),
				RawAsyncHalvingConfig: &expconf.AsyncHalvingConfig{
					RawNumRungs:  ptrs.IntPtr(3),
					RawMaxLength: lengthPtr(expconf.NewLengthInBatches(9000)),
					RawMaxTrials: ptrs.IntPtr(12),
					RawDivisor:   ptrs.Float64Ptr(3),
					RawStopOnce:  ptrs.BoolPtr(true),
				},
			},
		},
		{
			name: "smaller is better (round robin)",
			expectedTrials: []predefinedTrial{
				newConstantPredefinedTrial(toOps("1000B 3000B 9000B"), 0.01),
				newConstantPredefinedTrial(toOps("1000B"), 0.02),
				newConstantPredefinedTrial(toOps("1000B"), 0.03),
				newConstantPredefinedTrial(toOps("1000B"), 0.04),
				newConstantPredefinedTrial(toOps("1000B 3000B 9000B"), 0.01),
				newConstantPredefinedTrial(toOps("1000B"), 0.02),
				newConstantPredefinedTrial(toOps("1000B"), 0.03),
				newConstantPredefinedTrial(toOps("1000B"), 0.04),
				newConstantPredefinedTrial(toOps("1000B 3000B 9000B"), 0.01),
				newConstantPredefinedTrial(toOps("1000B"), 0.02),
				newConstantPredefinedTrial(toOps("1000B"), 0.03),
				newConstantPredefinedTrial(toOps("1000B"), 0.04),
			},
			config: expconf.SearcherConfig{
				RawSmallerIsBetter: ptrs.BoolPtr(true),
				RawAsyncHalvingConfig: &expconf.AsyncHalvingConfig{
					RawNumRungs:  ptrs.IntPtr(3),
					RawMaxLength: lengthPtr(expconf.NewLengthInBatches(9000)),
					RawMaxTrials: ptrs.IntPtr(12),
					RawDivisor:   ptrs.Float64Ptr(3),
					RawStopOnce:  ptrs.BoolPtr(true),
				},
			},
		},
		{
			name: "smaller is not better",
			expectedTrials: []predefinedTrial{
				newConstantPredefinedTrial(toOps("1000B 3000B 9000B"), 0.01),
				newConstantPredefinedTrial(toOps("1000B 3000B 9000B"), 0.02),
				newConstantPredefinedTrial(toOps("1000B 3000B 9000B"), 0.03),
				newConstantPredefinedTrial(toOps("1000B 3000B 9000B"), 0.04),
				newConstantPredefinedTrial(toOps("1000B 3000B 9000B"), 0.05),
				newConstantPredefinedTrial(toOps("1000B 3000B 9000B"), 0.06),
				newConstantPredefinedTrial(toOps("1000B 3000B 9000B"), 0.07),
				newConstantPredefinedTrial(toOps("1000B 3000B 9000B"), 0.08),
				newConstantPredefinedTrial(toOps("1000B 3000B 9000B"), 0.09),
				newConstantPredefinedTrial(toOps("1000B 3000B 9000B"), 0.10),
				newConstantPredefinedTrial(toOps("1000B 3000B 9000B"), 0.11),
				newConstantPredefinedTrial(toOps("1000B 3000B 9000B"), 0.12),
			},
			config: expconf.SearcherConfig{
				RawSmallerIsBetter: ptrs.BoolPtr(false),
				RawAsyncHalvingConfig: &expconf.AsyncHalvingConfig{
					RawNumRungs:  ptrs.IntPtr(3),
					RawMaxLength: lengthPtr(expconf.NewLengthInBatches(9000)),
					RawMaxTrials: ptrs.IntPtr(12),
					RawDivisor:   ptrs.Float64Ptr(3),
					RawStopOnce:  ptrs.BoolPtr(true),
				},
			},
		},
		{
			name: "smaller is not better (round robin)",
			expectedTrials: []predefinedTrial{
				newConstantPredefinedTrial(toOps("1000B 3000B 9000B"), 0.01),
				newConstantPredefinedTrial(toOps("1000B 3000B 9000B"), 0.02),
				newConstantPredefinedTrial(toOps("1000B 3000B 9000B"), 0.03),
				newConstantPredefinedTrial(toOps("1000B 3000B 9000B"), 0.04),
				newConstantPredefinedTrial(toOps("1000B"), 0.01),
				newConstantPredefinedTrial(toOps("1000B"), 0.02),
				newConstantPredefinedTrial(toOps("1000B 3000B"), 0.03),
				newConstantPredefinedTrial(toOps("1000B 3000B 9000B"), 0.04),
				newConstantPredefinedTrial(toOps("1000B"), 0.01),
				newConstantPredefinedTrial(toOps("1000B"), 0.02),
				newConstantPredefinedTrial(toOps("1000B 3000B"), 0.03),
				newConstantPredefinedTrial(toOps("1000B 3000B 9000B"), 0.04),
			},
			config: expconf.SearcherConfig{
				RawSmallerIsBetter: ptrs.BoolPtr(false),
				RawAsyncHalvingConfig: &expconf.AsyncHalvingConfig{
					RawNumRungs:  ptrs.IntPtr(3),
					RawMaxLength: lengthPtr(expconf.NewLengthInBatches(9000)),
					RawMaxTrials: ptrs.IntPtr(12),
					RawDivisor:   ptrs.Float64Ptr(3),
					RawStopOnce:  ptrs.BoolPtr(true),
				},
			},
		},
		{
			name: "early exit -- smaller is better (round robin)",
			expectedTrials: []predefinedTrial{
				newConstantPredefinedTrial(toOps("1000B 3000B 9000B"), 0.01),
				newConstantPredefinedTrial(toOps("1000B"), 0.02),
				newConstantPredefinedTrial(toOps("1000B"), 0.03),
				newConstantPredefinedTrial(toOps("1000B"), 0.04),
				newEarlyExitPredefinedTrial(toOps("1000B 3000B"), 0.01),
				newConstantPredefinedTrial(toOps("1000B"), 0.02),
				newConstantPredefinedTrial(toOps("1000B"), 0.03),
				newConstantPredefinedTrial(toOps("1000B"), 0.04),
				newConstantPredefinedTrial(toOps("1000B 3000B 9000B"), 0.01),
				newConstantPredefinedTrial(toOps("1000B"), 0.02),
				newConstantPredefinedTrial(toOps("1000B"), 0.03),
				newConstantPredefinedTrial(toOps("1000B"), 0.04),
			},
			config: expconf.SearcherConfig{
				RawSmallerIsBetter: ptrs.BoolPtr(true),
				RawAsyncHalvingConfig: &expconf.AsyncHalvingConfig{
					RawNumRungs:  ptrs.IntPtr(3),
					RawMaxLength: lengthPtr(expconf.NewLengthInBatches(9000)),
					RawMaxTrials: ptrs.IntPtr(12),
					RawDivisor:   ptrs.Float64Ptr(3),
					RawStopOnce:  ptrs.BoolPtr(true),
				},
			},
		},
		{
			name: "early exit -- smaller is not better (round robin)",
			expectedTrials: []predefinedTrial{
				newConstantPredefinedTrial(toOps("1000B 3000B 9000B"), 0.01),
				newConstantPredefinedTrial(toOps("1000B 3000B 9000B"), 0.02),
				newConstantPredefinedTrial(toOps("1000B 3000B 9000B"), 0.03),
				newEarlyExitPredefinedTrial(toOps("1000B 3000B"), 0.04),
				newConstantPredefinedTrial(toOps("1000B"), 0.01),
				newConstantPredefinedTrial(toOps("1000B"), 0.02),
				newConstantPredefinedTrial(toOps("1000B 3000B 9000B"), 0.03),
				newConstantPredefinedTrial(toOps("1000B 3000B 9000B"), 0.04),
				newConstantPredefinedTrial(toOps("1000B"), 0.01),
				newConstantPredefinedTrial(toOps("1000B"), 0.02),
				newConstantPredefinedTrial(toOps("1000B 3000B 9000B"), 0.03),
				newConstantPredefinedTrial(toOps("1000B 3000B 9000B"), 0.04),
			},
			config: expconf.SearcherConfig{
				RawSmallerIsBetter: ptrs.BoolPtr(false),
				RawAsyncHalvingConfig: &expconf.AsyncHalvingConfig{
					RawNumRungs:  ptrs.IntPtr(3),
					RawMaxLength: lengthPtr(expconf.NewLengthInBatches(9000)),
					RawMaxTrials: ptrs.IntPtr(12),
					RawDivisor:   ptrs.Float64Ptr(3),
					RawStopOnce:  ptrs.BoolPtr(true),
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
			config: expconf.SearcherConfig{
				RawSmallerIsBetter: ptrs.BoolPtr(true),
				RawAsyncHalvingConfig: &expconf.AsyncHalvingConfig{
					RawNumRungs:  ptrs.IntPtr(1),
					RawMaxLength: lengthPtr(expconf.NewLengthInBatches(9000)),
					RawMaxTrials: ptrs.IntPtr(4),
					RawDivisor:   ptrs.Float64Ptr(3),
					RawStopOnce:  ptrs.BoolPtr(true),
				},
			},
		},
	}

	runValueSimulationTestCases(t, testCases)
}
