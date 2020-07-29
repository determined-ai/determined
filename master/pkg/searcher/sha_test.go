package searcher

import (
	"testing"

	"github.com/determined-ai/determined/master/pkg/model"
)

func TestSHASearcherWithRecords(t *testing.T) {
	actual := model.SyncHalvingConfig{
		Metric:          defaultMetric,
		NumRungs:        4,
		MaxLength:       model.NewLengthInRecords(5120050),
		Budget:          model.NewLengthInRecords(3072050),
		Divisor:         4,
		TrainStragglers: true,
	}
	expected := [][]Runnable{
		toOps("80000R V"), toOps("80000R V"), toOps("80000R V"),
		toOps("80000R V"), toOps("80000R V"), toOps("80000R V"),
		toOps("80000R V"), toOps("80000R V"), toOps("80000R V"),
		toOps("80000R V 240003R V"),
		toOps("80000R V 240003R V 960009R V 3840038R V"),
	}
	searchMethod := newSyncHalvingSearch(actual)
	checkSimulation(t, searchMethod, nil, ConstantValidation, expected)
}

func TestSHASearcherWithBatches(t *testing.T) {
	actual := model.SyncHalvingConfig{
		Metric:          defaultMetric,
		NumRungs:        4,
		MaxLength:       model.NewLengthInBatches(80000),
		Budget:          model.NewLengthInBatches(48000),
		Divisor:         4,
		TrainStragglers: true,
	}
	expected := [][]Runnable{
		toOps("1250B V"), toOps("1250B V"), toOps("1250B V"),
		toOps("1250B V"), toOps("1250B V"), toOps("1250B V"),
		toOps("1250B V"), toOps("1250B V"), toOps("1250B V"),
		toOps("1250B V 3750B V"),
		toOps("1250B V 3750B V 15000B V 60000B V"),
	}
	searchMethod := newSyncHalvingSearch(actual)
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
			config: model.SearcherConfig{
				SyncHalvingConfig: &model.SyncHalvingConfig{
					Metric:          "error",
					NumRungs:        4,
					SmallerIsBetter: true,
					MaxLength:       model.NewLengthInBatches(80000),
					Budget:          model.NewLengthInBatches(48000),
					Divisor:         4,
					TrainStragglers: true,
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
			config: model.SearcherConfig{
				SyncHalvingConfig: &model.SyncHalvingConfig{
					Metric:          "error",
					NumRungs:        4,
					SmallerIsBetter: true,
					MaxLength:       model.NewLengthInBatches(80000),
					Budget:          model.NewLengthInBatches(48000),
					Divisor:         4,
					TrainStragglers: true,
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
			config: model.SearcherConfig{
				SyncHalvingConfig: &model.SyncHalvingConfig{
					Metric:          "error",
					NumRungs:        4,
					SmallerIsBetter: false,
					MaxLength:       model.NewLengthInBatches(80000),
					Budget:          model.NewLengthInBatches(48000),
					Divisor:         4,
					TrainStragglers: true,
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
			config: model.SearcherConfig{
				SyncHalvingConfig: &model.SyncHalvingConfig{
					Metric:          "error",
					NumRungs:        4,
					SmallerIsBetter: false,
					MaxLength:       model.NewLengthInBatches(80000),
					Budget:          model.NewLengthInBatches(48000),
					Divisor:         4,
					TrainStragglers: true,
				},
			},
		},
	}

	runValueSimulationTestCases(t, testCases)
}
