package searcher

import (
	"testing"

	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
)

func TestRandomSearcherRecords(t *testing.T) {
	actual := expconf.RandomConfig{
		RawMaxTrials: ptrs.IntPtr(4), RawMaxLength: lengthPtr(expconf.NewLengthInRecords(19200)),
	}
	actual = schemas.WithDefaults(actual).(expconf.RandomConfig)
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
		RawMaxTrials: ptrs.IntPtr(4), RawMaxLength: lengthPtr(expconf.NewLengthInBatches(300)),
	}
	actual = schemas.WithDefaults(actual).(expconf.RandomConfig)
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
		RawMaxTrials: ptrs.IntPtr(4), RawMaxLength: lengthPtr(expconf.NewLengthInBatches(300)),
	}
	conf = schemas.WithDefaults(conf).(expconf.RandomConfig)
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
					RawMaxLength:           lengthPtr(expconf.NewLengthInBatches(500)),
					RawMaxTrials:           ptrs.IntPtr(4),
					RawMaxConcurrentTrials: ptrs.IntPtr(2),
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
			config: expconf.SearcherConfig{
				RawRandomConfig: &expconf.RandomConfig{
					RawMaxLength: lengthPtr(expconf.NewLengthInRecords(32017)),
					RawMaxTrials: ptrs.IntPtr(4),
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
			config: expconf.SearcherConfig{
				RawSingleConfig: &expconf.SingleConfig{
					RawMaxLength: lengthPtr(expconf.NewLengthInBatches(500)),
				},
			},
		},
	}

	runValueSimulationTestCases(t, testCases)
}
