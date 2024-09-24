//nolint:exhaustruct
package searcher

import (
	"encoding/json"
	"strconv"
	"testing"

	"github.com/pkg/errors"
	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
)

func generateHyperparameters(counts []int) expconf.Hyperparameters {
	params := make(expconf.Hyperparameters, len(counts))
	for i, count := range counts {
		params[strconv.Itoa(i)] = expconf.Hyperparameter{
			RawDoubleHyperparameter: &expconf.DoubleHyperparameter{
				RawMinval: -1.0, RawMaxval: 1.0, RawCount: &count,
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
	grid := NewHyperparameterGrid(generateHyperparameters(counts))
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
	dParam := expconf.DoubleHyperparameter{RawMaxval: 2.0, RawCount: ptrs.Ptr(5)}
	assert.Equal(t,
		len(getGridAxes([]string{"x"}, expconf.Hyperparameter{RawDoubleHyperparameter: &dParam})[0]),
		5,
	)
	iParam := expconf.IntHyperparameter{RawMaxval: 20, RawCount: ptrs.Ptr(7)}
	assert.Equal(t,
		len(getGridAxes([]string{"x"}, expconf.Hyperparameter{RawIntHyperparameter: &iParam})[0]),
		7,
	)
	lParam := expconf.LogHyperparameter{
		RawMinval: -3.0, RawMaxval: -2.0, RawBase: 10, RawCount: ptrs.Ptr(2),
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
	// Regression test: make sure empty nested hyperparameters don't disappear during sampling.
	nestedParam := map[string]expconf.Hyperparameter{
		"empty": {RawNestedHyperparameter: &map[string]expconf.Hyperparameter{}},
		"full":  {RawCategoricalHyperparameter: &catParam},
	}
	result := getGridAxes([]string{"x"}, expconf.Hyperparameter{RawNestedHyperparameter: &nestedParam})
	assert.DeepEqual(t, result, []gridAxis{
		[]axisValue{{Route: []string{"x", "empty"}, Value: map[string]interface{}{}}},
		[]axisValue{
			{Route: []string{"x", "full"}, Value: 1},
			{Route: []string{"x", "full"}, Value: 2},
			{Route: []string{"x", "full"}, Value: 3},
		},
	})
}

func TestGrid(t *testing.T) {
	iParam1 := &expconf.IntHyperparameter{RawMaxval: 20, RawCount: ptrs.Ptr(3)}
	iParam2 := &expconf.IntHyperparameter{RawMaxval: 10, RawCount: ptrs.Ptr(3)}
	hparams := expconf.Hyperparameters{
		"1": expconf.Hyperparameter{RawIntHyperparameter: iParam1},
		"2": expconf.Hyperparameter{RawIntHyperparameter: iParam2},
	}
	actual := NewHyperparameterGrid(hparams)
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
	iParam1 := &expconf.IntHyperparameter{RawMaxval: 20, RawCount: ptrs.Ptr(3)}
	iParam2 := &expconf.IntHyperparameter{RawMaxval: 10, RawCount: ptrs.Ptr(3)}
	hparams := expconf.Hyperparameters{
		"1": expconf.Hyperparameter{RawIntHyperparameter: iParam1},
		"2": expconf.Hyperparameter{
			RawNestedHyperparameter: &map[string]expconf.Hyperparameter{
				"3": {RawIntHyperparameter: iParam2},
			},
		},
	}
	actual := NewHyperparameterGrid(hparams)
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
								RawCount:  ptrs.Ptr(2),
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
				RawCount:  ptrs.Ptr(2),
			},
		},
		"l": {
			RawLogHyperparameter: &expconf.LogHyperparameter{
				RawMinval: 1,
				RawMaxval: 2,
				RawBase:   10,
				RawCount:  ptrs.Ptr(2),
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

	for _, sample := range NewHyperparameterGrid(hps) {
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
				RawMinval: 0, RawMaxval: 4, RawCount: ptrs.Ptr(5),
			},
		},
	}
	actual := NewHyperparameterGrid(hparams)
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
				RawMinval: -4, RawMaxval: -2, RawCount: ptrs.Ptr(3),
			},
		},
	}
	actual := NewHyperparameterGrid(hparams)
	expected := []HParamSample{
		{"1": -4},
		{"1": -3},
		{"1": -2},
	}
	assert.DeepEqual(t, actual, expected)
}

func TestGridSearchMethod(t *testing.T) {
	// write this
}
