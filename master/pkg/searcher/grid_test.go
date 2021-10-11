package searcher

import (
	"encoding/json"
	"strconv"
	"testing"

	"github.com/pkg/errors"
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
	assert.Equal(t,
		len(getGridAxes([]string{"x"}, expconf.Hyperparameter{RawDoubleHyperparameter: &dParam})[0]),
		5,
	)
	iParam := expconf.IntHyperparameter{RawMaxval: 20, RawCount: ptrs.IntPtr(7)}
	assert.Equal(t,
		len(getGridAxes([]string{"x"}, expconf.Hyperparameter{RawIntHyperparameter: &iParam})[0]),
		7,
	)
	lParam := expconf.LogHyperparameter{
		RawMinval: -3.0, RawMaxval: -2.0, RawBase: 10, RawCount: ptrs.IntPtr(2),
	}
	assert.Equal(t,
		len(getGridAxes([]string{"x"}, expconf.Hyperparameter{RawLogHyperparameter: &lParam})[0]),
		2,
	)
	catParam := expconf.CategoricalHyperparameter{RawVals: []interface{}{1, 2, 3}}
	assert.Equal(t,
		len(getGridAxes(
			[]string{"x"}, expconf.Hyperparameter{RawCategoricalHyperparameter: &catParam})[0],
		),
		3,
	)
	constParam := expconf.ConstHyperparameter{RawVal: 3}
	assert.Equal(t,
		len(getGridAxes([]string{"x"}, expconf.Hyperparameter{RawConstHyperparameter: &constParam})[0]),
		1,
	)
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
		{"1": 0, "2": HParamSample{"3": 0}},
		{"1": 0, "2": HParamSample{"3": 5}},
		{"1": 0, "2": HParamSample{"3": 10}},
		{"1": 10, "2": HParamSample{"3": 0}},
		{"1": 10, "2": HParamSample{"3": 5}},
		{"1": 10, "2": HParamSample{"3": 10}},
		{"1": 20, "2": HParamSample{"3": 0}},
		{"1": 20, "2": HParamSample{"3": 5}},
		{"1": 20, "2": HParamSample{"3": 10}},
	}
	assert.DeepEqual(t, actual, expected)
}

func TestNestedGridFurther(t *testing.T) {
	hps := map[string]expconf.Hyperparameter{
		"constant": {
			RawConstHyperparameter: &expconf.ConstHyperparameter{RawVal: 2},
		},
		"a": {
			RawNestedHyperparameter: &map[string]expconf.Hyperparameter{
				"b": {
					RawNestedHyperparameter: &map[string]expconf.Hyperparameter{
						"c1": {
							RawCategoricalHyperparameter: &expconf.CategoricalHyperparameter{
								RawVals: []interface{}{3, 5},
							},
						},
						"c2": {
							RawIntHyperparameter: &expconf.IntHyperparameter{
								RawMinval: 7,
								RawMaxval: 11,
								RawCount:  ptrs.IntPtr(2),
							},
						},
					},
				},
			},
		},
		"f": {
			RawDoubleHyperparameter: &expconf.DoubleHyperparameter{
				RawMinval: 13.0,
				RawMaxval: 17.0,
				RawCount:  ptrs.IntPtr(2),
			},
		},
		"l": {
			RawLogHyperparameter: &expconf.LogHyperparameter{
				RawMinval: 1,
				RawMaxval: 2,
				RawBase:   10,
				RawCount:  ptrs.IntPtr(2),
			},
		},
	}

	expect := map[string]bool{
		`{"a":{"b":{"c1":3,"c2":7}},"constant":2,"f":13,"l":10}`:   true,
		`{"a":{"b":{"c1":3,"c2":11}},"constant":2,"f":13,"l":10}`:  true,
		`{"a":{"b":{"c1":5,"c2":7}},"constant":2,"f":13,"l":10}`:   true,
		`{"a":{"b":{"c1":5,"c2":11}},"constant":2,"f":13,"l":10}`:  true,
		`{"a":{"b":{"c1":3,"c2":7}},"constant":2,"f":17,"l":10}`:   true,
		`{"a":{"b":{"c1":3,"c2":11}},"constant":2,"f":17,"l":10}`:  true,
		`{"a":{"b":{"c1":5,"c2":7}},"constant":2,"f":17,"l":10}`:   true,
		`{"a":{"b":{"c1":5,"c2":11}},"constant":2,"f":17,"l":10}`:  true,
		`{"a":{"b":{"c1":3,"c2":7}},"constant":2,"f":13,"l":100}`:  true,
		`{"a":{"b":{"c1":3,"c2":11}},"constant":2,"f":13,"l":100}`: true,
		`{"a":{"b":{"c1":5,"c2":7}},"constant":2,"f":13,"l":100}`:  true,
		`{"a":{"b":{"c1":5,"c2":11}},"constant":2,"f":13,"l":100}`: true,
		`{"a":{"b":{"c1":3,"c2":7}},"constant":2,"f":17,"l":100}`:  true,
		`{"a":{"b":{"c1":3,"c2":11}},"constant":2,"f":17,"l":100}`: true,
		`{"a":{"b":{"c1":5,"c2":7}},"constant":2,"f":17,"l":100}`:  true,
		`{"a":{"b":{"c1":5,"c2":11}},"constant":2,"f":17,"l":100}`: true,
	}

	for _, sample := range newHyperparameterGrid(hps) {
		byts, err := json.Marshal(sample)
		assert.NilError(t, err)
		result := string(byts)
		val, ok := expect[result]
		if !ok {
			assert.NilError(t, errors.Errorf("got unexpected value: %v", result))
		}
		if !val {
			assert.NilError(t, errors.Errorf("got value twice: %v", result))
		}
		expect[result] = false
	}

	for exp, val := range expect {
		if val {
			assert.NilError(t, errors.Errorf("did not see %v", exp))
		}
	}
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
