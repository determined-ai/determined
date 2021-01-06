package searcher

import (
	"testing"

	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
)

func TestSHASearcherWithRecords(t *testing.T) {
	config := expconf.SyncHalvingConfig{
		Metric:          defaultMetric,
		NumRungs:        4,
		MaxLength:       expconf.NewLengthInRecords(5120050),
		Budget:          expconf.NewLengthInRecords(3072050),
		Divisor:         ptrs.Float64Ptr(4),
		TrainStragglers: ptrs.BoolPtr(true),
	}
	schemas.FillDefaults(&config)
	expected := [][]Runnable{
		toOps("80000R V"), toOps("80000R V"), toOps("80000R V"),
		toOps("80000R V"), toOps("80000R V"), toOps("80000R V"),
		toOps("80000R V"), toOps("80000R V"), toOps("80000R V"),
		toOps("80000R V 240003R V"),
		toOps("80000R V 240003R V 960009R V 3840038R V"),
	}
	searchMethod := newSyncHalvingSearch(config)
	checkSimulation(t, searchMethod, nil, ConstantValidation, expected)
}

func TestSHASearcherWithBatches(t *testing.T) {
	config := expconf.SyncHalvingConfig{
		Metric:          defaultMetric,
		NumRungs:        4,
		MaxLength:       expconf.NewLengthInBatches(80000),
		Budget:          expconf.NewLengthInBatches(48000),
		Divisor:         ptrs.Float64Ptr(4),
		TrainStragglers: ptrs.BoolPtr(true),
	}
	schemas.FillDefaults(&config)
	expected := [][]Runnable{
		toOps("1250B V"), toOps("1250B V"), toOps("1250B V"),
		toOps("1250B V"), toOps("1250B V"), toOps("1250B V"),
		toOps("1250B V"), toOps("1250B V"), toOps("1250B V"),
		toOps("1250B V 3750B V"),
		toOps("1250B V 3750B V 15000B V 60000B V"),
	}
	searchMethod := newSyncHalvingSearch(config)
	checkSimulation(t, searchMethod, nil, ConstantValidation, expected)
}

func TestSHASearchMethod(t *testing.T) {
	testCases := []valueSimulationTestCase{
		{
			name: "smaller is better",
			expectedTrials: []predefinedTrial{
				newConstantPredefinedTrial(toOps("1250B V 3750B V 15000B V 60000B V"), 0.01),
				newConstantPredefinedTrial(toOps("1250B V 3750B V"), 0.02),
				newConstantPredefinedTrial(toOps("1250B V"), 0.03),
				newConstantPredefinedTrial(toOps("1250B V"), 0.04),
				newConstantPredefinedTrial(toOps("1250B V"), 0.05),
				newConstantPredefinedTrial(toOps("1250B V"), 0.06),
				newConstantPredefinedTrial(toOps("1250B V"), 0.07),
				newConstantPredefinedTrial(toOps("1250B V"), 0.08),
				newConstantPredefinedTrial(toOps("1250B V"), 0.09),
				newConstantPredefinedTrial(toOps("1250B V"), 0.10),
				newConstantPredefinedTrial(toOps("1250B V"), 0.11),
			},
			config: expconf.SearcherConfig{
				SyncHalvingConfig: &expconf.SyncHalvingConfig{
					Metric:          "error",
					NumRungs:        4,
					SmallerIsBetter: ptrs.BoolPtr(true),
					MaxLength:       expconf.NewLengthInBatches(80000),
					Budget:          expconf.NewLengthInBatches(48000),
					Divisor:         ptrs.Float64Ptr(4),
					TrainStragglers: ptrs.BoolPtr(true),
				},
			},
		},
		{
			name: "early exit -- smaller is better",
			expectedTrials: []predefinedTrial{
				newConstantPredefinedTrial(toOps("1250B V 3750B V 15000B V 60000B V"), 0.01),
				newConstantPredefinedTrial(toOps("1250B V 3750B V"), 0.02),
				newConstantPredefinedTrial(toOps("1250B V"), 0.03),
				newConstantPredefinedTrial(toOps("1250B V"), 0.04),
				newConstantPredefinedTrial(toOps("1250B V"), 0.05),
				newConstantPredefinedTrial(toOps("1250B V"), 0.06),
				newConstantPredefinedTrial(toOps("1250B V"), 0.07),
				newConstantPredefinedTrial(toOps("1250B V"), 0.08),
				newConstantPredefinedTrial(toOps("1250B V"), 0.09),
				newConstantPredefinedTrial(toOps("1250B V"), 0.10),
				newEarlyExitPredefinedTrial(toOps("1250B"), 0.11),
			},
			config: expconf.SearcherConfig{
				SyncHalvingConfig: &expconf.SyncHalvingConfig{
					Metric:          "error",
					NumRungs:        4,
					SmallerIsBetter: ptrs.BoolPtr(true),
					MaxLength:       expconf.NewLengthInBatches(80000),
					Budget:          expconf.NewLengthInBatches(48000),
					Divisor:         ptrs.Float64Ptr(4),
					TrainStragglers: ptrs.BoolPtr(true),
				},
			},
		},
		{
			name: "smaller is not better",
			expectedTrials: []predefinedTrial{
				newConstantPredefinedTrial(toOps("1250B V 3750B V 15000B V 60000B V"), 0.11),
				newConstantPredefinedTrial(toOps("1250B V 3750B V"), 0.10),
				newConstantPredefinedTrial(toOps("1250B V"), 0.09),
				newConstantPredefinedTrial(toOps("1250B V"), 0.08),
				newConstantPredefinedTrial(toOps("1250B V"), 0.07),
				newConstantPredefinedTrial(toOps("1250B V"), 0.06),
				newConstantPredefinedTrial(toOps("1250B V"), 0.05),
				newConstantPredefinedTrial(toOps("1250B V"), 0.04),
				newConstantPredefinedTrial(toOps("1250B V"), 0.03),
				newConstantPredefinedTrial(toOps("1250B V"), 0.02),
				newEarlyExitPredefinedTrial(toOps("1250B"), 0.01),
			},
			config: expconf.SearcherConfig{
				SyncHalvingConfig: &expconf.SyncHalvingConfig{
					Metric:          "error",
					NumRungs:        4,
					SmallerIsBetter: ptrs.BoolPtr(false),
					MaxLength:       expconf.NewLengthInBatches(80000),
					Budget:          expconf.NewLengthInBatches(48000),
					Divisor:         ptrs.Float64Ptr(4),
					TrainStragglers: ptrs.BoolPtr(true),
				},
			},
		},
		{
			name: "early exit -- smaller is not better",
			expectedTrials: []predefinedTrial{
				newConstantPredefinedTrial(toOps("1250B V 3750B V 15000B V 60000B V"), 0.11),
				newEarlyExitPredefinedTrial(toOps("1250B V 3750B"), 0.10),
				newConstantPredefinedTrial(toOps("1250B V"), 0.09),
				newConstantPredefinedTrial(toOps("1250B V"), 0.08),
				newConstantPredefinedTrial(toOps("1250B V"), 0.07),
				newConstantPredefinedTrial(toOps("1250B V"), 0.06),
				newConstantPredefinedTrial(toOps("1250B V"), 0.05),
				newConstantPredefinedTrial(toOps("1250B V"), 0.04),
				newConstantPredefinedTrial(toOps("1250B V"), 0.03),
				newConstantPredefinedTrial(toOps("1250B V"), 0.02),
				newEarlyExitPredefinedTrial(toOps("1250B"), 0.01),
			},
			config: expconf.SearcherConfig{
				SyncHalvingConfig: &expconf.SyncHalvingConfig{
					Metric:          "error",
					NumRungs:        4,
					SmallerIsBetter: ptrs.BoolPtr(false),
					MaxLength:       expconf.NewLengthInBatches(80000),
					Budget:          expconf.NewLengthInBatches(48000),
					Divisor:         ptrs.Float64Ptr(4),
					TrainStragglers: ptrs.BoolPtr(true),
				},
			},
		},
	}

	runValueSimulationTestCases(t, testCases)
}
