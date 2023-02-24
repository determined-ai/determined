//nolint:exhaustivestruct
package searcher

import (
	"testing"

	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
)

func TestRandomSearcherRecords(t *testing.T) {
	actual := expconf.RandomConfig{
		RawMaxTrials: ptrs.Ptr(4), RawMaxLength: ptrs.Ptr(expconf.NewLengthInRecords(19200)),
	}
	actual = schemas.WithDefaults(actual)
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
	actual := expconf.RandomConfig{
		RawMaxTrials: ptrs.Ptr(4), RawMaxLength: ptrs.Ptr(expconf.NewLengthInBatches(300)),
	}
	actual = schemas.WithDefaults(actual)
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
	conf := expconf.RandomConfig{
		RawMaxTrials: ptrs.Ptr(4), RawMaxLength: ptrs.Ptr(expconf.NewLengthInBatches(300)),
	}
	conf = schemas.WithDefaults(conf)
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
			config: expconf.SearcherConfig{
				RawRandomConfig: &expconf.RandomConfig{
					RawMaxLength:           ptrs.Ptr(expconf.NewLengthInBatches(500)),
					RawMaxTrials:           ptrs.Ptr(4),
					RawMaxConcurrentTrials: ptrs.Ptr(2),
				},
				RawMetric: ptrs.Ptr("loss"),
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
			config: expconf.SearcherConfig{
				RawRandomConfig: &expconf.RandomConfig{
					RawMaxLength: ptrs.Ptr(expconf.NewLengthInRecords(32017)),
					RawMaxTrials: ptrs.Ptr(4),
				},
				RawMetric: ptrs.Ptr("loss"),
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
			config: expconf.SearcherConfig{
				RawSingleConfig: &expconf.SingleConfig{
					RawMaxLength: ptrs.Ptr(expconf.NewLengthInBatches(500)),
				},
				RawMetric: ptrs.Ptr("loss"),
			},
		},
	}

	runValueSimulationTestCases(t, testCases)
}

func TestRandomSearcherSingleConcurrent(t *testing.T) {
	actual := expconf.RandomConfig{
		RawMaxTrials:           ptrs.Ptr(2),
		RawMaxLength:           ptrs.Ptr(expconf.NewLengthInRecords(100)),
		RawMaxConcurrentTrials: ptrs.Ptr(1),
	}
	actual = schemas.WithDefaults(actual)
	expected := [][]ValidateAfter{
		toOps("100R"),
		toOps("100R"),
	}
	search := newRandomSearch(actual)
	checkSimulation(t, search, nil, ConstantValidation, expected)
}
