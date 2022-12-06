//nolint:exhaustivestruct
package searcher

import (
	"testing"

	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
)

func TestASHASearcherRecords(t *testing.T) {
	actual := expconf.AsyncHalvingConfig{
		RawNumRungs:  ptrs.Ptr(3),
		RawMaxLength: ptrs.Ptr(expconf.NewLengthInRecords(576000)),
		RawDivisor:   ptrs.Ptr[float64](3),
		RawMaxTrials: ptrs.Ptr(12),
	}
	actual = schemas.WithDefaults(actual)
	expected := [][]ValidateAfter{
		toOps("64000R"), toOps("64000R"), toOps("64000R"),
		toOps("64000R"), toOps("64000R"), toOps("64000R"),
		toOps("64000R"), toOps("64000R"),
		toOps("64000R 192000R"),
		toOps("64000R 192000R"),
		toOps("64000R 192000R"),
		toOps("64000R 192000R 576000R"),
	}
	checkSimulation(t, newAsyncHalvingSearch(actual, true), nil, ConstantValidation, expected)
}

func TestASHASearcherBatches(t *testing.T) {
	actual := expconf.AsyncHalvingConfig{
		RawNumRungs:  ptrs.Ptr(3),
		RawMaxLength: ptrs.Ptr(expconf.NewLengthInBatches(9000)),
		RawDivisor:   ptrs.Ptr[float64](3),
		RawMaxTrials: ptrs.Ptr(12),
	}
	actual = schemas.WithDefaults(actual)
	expected := [][]ValidateAfter{
		toOps("1000B"), toOps("1000B"), toOps("1000B"),
		toOps("1000B"), toOps("1000B"), toOps("1000B"),
		toOps("1000B"), toOps("1000B"),
		toOps("1000B 3000B"),
		toOps("1000B 3000B"),
		toOps("1000B 3000B"),
		toOps("1000B 3000B 9000B"),
	}
	checkSimulation(t, newAsyncHalvingSearch(actual, true), nil, ConstantValidation, expected)
}

func TestASHASearcherEpochs(t *testing.T) {
	actual := expconf.AsyncHalvingConfig{
		RawNumRungs:  ptrs.Ptr(3),
		RawMaxLength: ptrs.Ptr(expconf.NewLengthInEpochs(12)),
		RawDivisor:   ptrs.Ptr[float64](3),
		RawMaxTrials: ptrs.Ptr(12),
	}
	actual = schemas.WithDefaults(actual)
	expected := [][]ValidateAfter{
		toOps("1E"), toOps("1E"), toOps("1E"),
		toOps("1E"), toOps("1E"), toOps("1E"),
		toOps("1E"), toOps("1E"),
		toOps("1E 4E"),
		toOps("1E 4E"),
		toOps("1E 4E"),
		toOps("1E 4E 12E"),
	}
	checkSimulation(t, newAsyncHalvingSearch(actual, true), nil, ConstantValidation, expected)
}

func TestASHASearchMethod(t *testing.T) {
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
			config: expconf.SearcherConfig{
				RawSmallerIsBetter: ptrs.Ptr(true),
				RawAsyncHalvingConfig: &expconf.AsyncHalvingConfig{
					RawNumRungs:  ptrs.Ptr(3),
					RawMaxLength: ptrs.Ptr(expconf.NewLengthInBatches(9000)),
					RawMaxTrials: ptrs.Ptr(12),
					RawDivisor:   ptrs.Ptr[float64](3),
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
			config: expconf.SearcherConfig{
				RawSmallerIsBetter: ptrs.Ptr(true),
				RawAsyncHalvingConfig: &expconf.AsyncHalvingConfig{
					RawNumRungs:  ptrs.Ptr(3),
					RawMaxLength: ptrs.Ptr(expconf.NewLengthInBatches(9000)),
					RawMaxTrials: ptrs.Ptr(12),
					RawDivisor:   ptrs.Ptr[float64](3),
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
			config: expconf.SearcherConfig{
				RawSmallerIsBetter: ptrs.Ptr(false),
				RawAsyncHalvingConfig: &expconf.AsyncHalvingConfig{
					RawNumRungs:  ptrs.Ptr(3),
					RawMaxLength: ptrs.Ptr(expconf.NewLengthInBatches(9000)),
					RawMaxTrials: ptrs.Ptr(12),
					RawDivisor:   ptrs.Ptr[float64](3),
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
			config: expconf.SearcherConfig{
				RawSmallerIsBetter: ptrs.Ptr(false),
				RawAsyncHalvingConfig: &expconf.AsyncHalvingConfig{
					RawNumRungs:  ptrs.Ptr(3),
					RawMaxLength: ptrs.Ptr(expconf.NewLengthInBatches(9000)),
					RawMaxTrials: ptrs.Ptr(12),
					RawDivisor:   ptrs.Ptr[float64](3),
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
			config: expconf.SearcherConfig{
				RawSmallerIsBetter: ptrs.Ptr(true),
				RawAsyncHalvingConfig: &expconf.AsyncHalvingConfig{
					RawNumRungs:  ptrs.Ptr(3),
					RawMaxLength: ptrs.Ptr(expconf.NewLengthInBatches(9000)),
					RawMaxTrials: ptrs.Ptr(12),
					RawDivisor:   ptrs.Ptr[float64](3),
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
				RawSmallerIsBetter: ptrs.Ptr(true),
				RawAsyncHalvingConfig: &expconf.AsyncHalvingConfig{
					RawNumRungs:  ptrs.Ptr(1),
					RawMaxLength: ptrs.Ptr(expconf.NewLengthInBatches(9000)),
					RawMaxTrials: ptrs.Ptr(4),
					RawDivisor:   ptrs.Ptr[float64](3),
				},
			},
		},
	}

	runValueSimulationTestCases(t, testCases)
}
