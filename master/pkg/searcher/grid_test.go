package searcher

import (
	"strconv"
	"testing"

	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
)

func generateHyperparameters(counts []int) expconf.Hyperparameters {
	params := make(expconf.Hyperparameters, len(counts))
	for i, count := range counts {
		c := count
		params[strconv.Itoa(i)] = expconf.Hyperparameter{
			RawDoubleHyperparameter: &expconf.DoubleHyperparameter{
				RawMinval: -1.0, RawMaxval: 1.0, RawCount: &c,
			},
		}
	}
	return params
}

func checkGrid(t *testing.T, counts []int) {
	numTrials := 1
	for _, count := range counts {
		numTrials *= count
	}
	grid := newHyperparameterGrid(generateHyperparameters(counts))
	assert.Equal(t, len(grid), numTrials)
}

func TestGridFunctionality(t *testing.T) {
	checkGrid(t, []int{1})
	checkGrid(t, []int{4})
	checkGrid(t, []int{1, 4})
	checkGrid(t, []int{3, 4})
	checkGrid(t, []int{2, 3, 4})
	checkGrid(t, []int{2, 2, 3, 3, 4, 5})
}

func TestHyperparameterGridMethod(t *testing.T) {
	dParam := expconf.DoubleHyperparameter{RawMaxval: 2.0, RawCount: ptrs.IntPtr(5)}
	assert.Equal(t, len(grid(expconf.Hyperparameter{RawDoubleHyperparameter: &dParam})), 5)
	iParam := expconf.IntHyperparameter{RawMaxval: 20, RawCount: ptrs.IntPtr(7)}
	assert.Equal(t, len(grid(expconf.Hyperparameter{RawIntHyperparameter: &iParam})), 7)
	lParam := expconf.LogHyperparameter{
		RawMinval: -3.0, RawMaxval: -2.0, RawBase: 10, RawCount: ptrs.IntPtr(2),
	}
	assert.Equal(t, len(grid(expconf.Hyperparameter{RawLogHyperparameter: &lParam})), 2)
	catParam := expconf.CategoricalHyperparameter{RawVals: []interface{}{1, 2, 3}}
	assert.Equal(t, len(grid(expconf.Hyperparameter{RawCategoricalHyperparameter: &catParam})), 3)
	constParam := expconf.ConstHyperparameter{RawVal: 3}
	assert.Equal(t, len(grid(expconf.Hyperparameter{RawConstHyperparameter: &constParam})), 1)
}

func TestGrid(t *testing.T) {
	iParam1 := &expconf.IntHyperparameter{RawMaxval: 20, RawCount: ptrs.IntPtr(3)}
	iParam2 := &expconf.IntHyperparameter{RawMaxval: 10, RawCount: ptrs.IntPtr(3)}
	hparams := expconf.Hyperparameters{
		"1": expconf.Hyperparameter{RawIntHyperparameter: iParam1},
		"2": expconf.Hyperparameter{RawIntHyperparameter: iParam2},
	}
	actual := newHyperparameterGrid(hparams)
	expected := []HParamSample{
		{"1": 0, "2": 0},
		{"1": 0, "2": 5},
		{"1": 0, "2": 10},
		{"1": 10, "2": 0},
		{"1": 10, "2": 5},
		{"1": 10, "2": 10},
		{"1": 20, "2": 0},
		{"1": 20, "2": 5},
		{"1": 20, "2": 10},
	}
	assert.DeepEqual(t, actual, expected)
}

func TestNestedGrid(t *testing.T) {
	iParam1 := &expconf.IntHyperparameter{RawMaxval: 20, RawCount: ptrs.IntPtr(3)}
	iParam2 := &expconf.IntHyperparameter{RawMaxval: 10, RawCount: ptrs.IntPtr(3)}
	hparams := expconf.Hyperparameters{
		"1": expconf.Hyperparameter{RawIntHyperparameter: iParam1},
		"2": expconf.Hyperparameter{
			RawNestedHyperparameter: &map[string]expconf.Hyperparameter{
				"3": {RawIntHyperparameter: iParam2},
			},
		},
	}
	actual := newHyperparameterGrid(hparams)
	expected := []HParamSample{
		{"1": 0, "2": map[string]interface{}{"3": 0}},
		{"1": 0, "2": map[string]interface{}{"3": 5}},
		{"1": 0, "2": map[string]interface{}{"3": 10}},
		{"1": 10, "2": map[string]interface{}{"3": 0}},
		{"1": 10, "2": map[string]interface{}{"3": 5}},
		{"1": 10, "2": map[string]interface{}{"3": 10}},
		{"1": 20, "2": map[string]interface{}{"3": 0}},
		{"1": 20, "2": map[string]interface{}{"3": 5}},
		{"1": 20, "2": map[string]interface{}{"3": 10}},
	}
	assert.DeepEqual(t, actual, expected)
}

func TestGridIntCount(t *testing.T) {
	hparams := expconf.Hyperparameters{
		"1": expconf.Hyperparameter{
			RawIntHyperparameter: &expconf.IntHyperparameter{
				RawMinval: 0, RawMaxval: 4, RawCount: ptrs.IntPtr(5),
			},
		},
	}
	actual := newHyperparameterGrid(hparams)
	expected := []HParamSample{
		{"1": 0},
		{"1": 1},
		{"1": 2},
		{"1": 3},
		{"1": 4},
	}
	assert.DeepEqual(t, actual, expected)
}

func TestGridIntCountNegative(t *testing.T) {
	hparams := expconf.Hyperparameters{
		"1": expconf.Hyperparameter{
			RawIntHyperparameter: &expconf.IntHyperparameter{
				RawMinval: -4, RawMaxval: -2, RawCount: ptrs.IntPtr(3),
			},
		},
	}
	actual := newHyperparameterGrid(hparams)
	expected := []HParamSample{
		{"1": -4},
		{"1": -3},
		{"1": -2},
	}
	assert.DeepEqual(t, actual, expected)
}

func TestGridSearcherRecords(t *testing.T) {
	actual := expconf.GridConfig{RawMaxLength: lengthPtr(expconf.NewLengthInRecords(19200))}
	actual = schemas.WithDefaults(actual).(expconf.GridConfig)
	params := generateHyperparameters([]int{2, 1, 3})
	expected := [][]ValidateAfter{
		toOps("19200R"), toOps("19200R"), toOps("19200R"),
		toOps("19200R"), toOps("19200R"), toOps("19200R"),
	}
	searchMethod := newGridSearch(actual)
	checkSimulation(t, searchMethod, params, ConstantValidation, expected)
}

func TestGridSearcherBatches(t *testing.T) {
	actual := expconf.GridConfig{RawMaxLength: lengthPtr(expconf.NewLengthInBatches(300))}
	actual = schemas.WithDefaults(actual).(expconf.GridConfig)
	params := generateHyperparameters([]int{2, 1, 3})
	expected := [][]ValidateAfter{
		toOps("300B"), toOps("300B"), toOps("300B"),
		toOps("300B"), toOps("300B"), toOps("300B"),
	}
	searchMethod := newGridSearch(actual)
	checkSimulation(t, searchMethod, params, ConstantValidation, expected)
}

func TestGridSearcherEpochs(t *testing.T) {
	actual := expconf.GridConfig{RawMaxLength: lengthPtr(expconf.NewLengthInEpochs(3))}
	actual = schemas.WithDefaults(actual).(expconf.GridConfig)
	params := generateHyperparameters([]int{2, 1, 3})
	expected := [][]ValidateAfter{
		toOps("3E"), toOps("3E"), toOps("3E"),
		toOps("3E"), toOps("3E"), toOps("3E"),
	}
	searchMethod := newGridSearch(actual)
	checkSimulation(t, searchMethod, params, ConstantValidation, expected)
}

func TestGridSearchMethod(t *testing.T) {
	testCases := []valueSimulationTestCase{
		{
			name: "test grid search method",
			expectedTrials: []predefinedTrial{
				newConstantPredefinedTrial(toOps("300B"), 0.1),
				newConstantPredefinedTrial(toOps("300B"), 0.1),
				newConstantPredefinedTrial(toOps("300B"), 0.1),
				newConstantPredefinedTrial(toOps("300B"), 0.1),
				newConstantPredefinedTrial(toOps("300B"), 0.1),
				newEarlyExitPredefinedTrial(toOps("300B"), .1),
			},
			config: expconf.SearcherConfig{
				RawGridConfig: &expconf.GridConfig{
					RawMaxLength:           lengthPtr(expconf.NewLengthInBatches(300)),
					RawMaxConcurrentTrials: ptrs.IntPtr(2),
				},
			},
			hparams: generateHyperparameters([]int{2, 1, 3}),
		},
	}

	runValueSimulationTestCases(t, testCases)
}
