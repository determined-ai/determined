package searcher

import (
	"strconv"
	"testing"

	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/pkg/model"
)

func intP(x int) *int {
	return &x
}

func generateHyperparameters(counts []int) model.Hyperparameters {
	params := make(model.Hyperparameters, len(counts))
	for i, count := range counts {
		c := count
		params[strconv.Itoa(i)] = model.Hyperparameter{
			DoubleHyperparameter: &model.DoubleHyperparameter{Minval: -1.0, Maxval: 1.0, Count: &c},
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
	dParam := model.DoubleHyperparameter{Maxval: 2.0, Count: intP(5)}
	assert.Equal(t, len(grid(model.Hyperparameter{DoubleHyperparameter: &dParam})), 5)
	iParam := model.IntHyperparameter{Maxval: 20, Count: intP(7)}
	assert.Equal(t, len(grid(model.Hyperparameter{IntHyperparameter: &iParam})), 7)
	lParam := model.LogHyperparameter{Minval: -3.0, Maxval: -2.0, Base: 10, Count: intP(2)}
	assert.Equal(t, len(grid(model.Hyperparameter{LogHyperparameter: &lParam})), 2)
	catParam := model.CategoricalHyperparameter{Vals: []interface{}{1, 2, 3}}
	assert.Equal(t, len(grid(model.Hyperparameter{CategoricalHyperparameter: &catParam})), 3)
	constParam := model.ConstHyperparameter{Val: 3}
	assert.Equal(t, len(grid(model.Hyperparameter{ConstHyperparameter: &constParam})), 1)
}

func TestGrid(t *testing.T) {
	iParam1 := &model.IntHyperparameter{Maxval: 20, Count: intP(3)}
	iParam2 := &model.IntHyperparameter{Maxval: 10, Count: intP(3)}
	hparams := model.Hyperparameters{
		"1": model.Hyperparameter{IntHyperparameter: iParam1},
		"2": model.Hyperparameter{IntHyperparameter: iParam2},
	}
	actual := newHyperparameterGrid(hparams)
	expected := []hparamSample{
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

func TestGridIntCount(t *testing.T) {
	hparams := model.Hyperparameters{
		"1": model.Hyperparameter{
			IntHyperparameter: &model.IntHyperparameter{Minval: 0, Maxval: 4, Count: intP(5)}},
	}
	actual := newHyperparameterGrid(hparams)
	expected := []hparamSample{
		{"1": 0},
		{"1": 1},
		{"1": 2},
		{"1": 3},
		{"1": 4},
	}
	assert.DeepEqual(t, actual, expected)
}

func TestGridIntCountNegative(t *testing.T) {
	hparams := model.Hyperparameters{
		"1": model.Hyperparameter{
			IntHyperparameter: &model.IntHyperparameter{Minval: -4, Maxval: -2, Count: intP(3)}},
	}
	actual := newHyperparameterGrid(hparams)
	expected := []hparamSample{
		{"1": -4},
		{"1": -3},
		{"1": -2},
	}
	assert.DeepEqual(t, actual, expected)
}

func TestGridSearcherRecords(t *testing.T) {
	actual := model.GridConfig{MaxLength: model.NewLengthInRecords(19200)}
	params := generateHyperparameters([]int{2, 1, 3})
	expected := [][]ValidateAfter{
		toOps("19200R"), toOps("19200R"), toOps("19200R"),
		toOps("19200R"), toOps("19200R"), toOps("19200R"),
	}
	searchMethod := newGridSearch(actual)
	checkSimulation(t, searchMethod, params, ConstantValidation, expected)
}

func TestGridSearcherBatches(t *testing.T) {
	actual := model.GridConfig{MaxLength: model.NewLengthInBatches(300)}
	params := generateHyperparameters([]int{2, 1, 3})
	expected := [][]ValidateAfter{
		toOps("300B"), toOps("300B"), toOps("300B"),
		toOps("300B"), toOps("300B"), toOps("300B"),
	}
	searchMethod := newGridSearch(actual)
	checkSimulation(t, searchMethod, params, ConstantValidation, expected)
}

func TestGridSearcherEpochs(t *testing.T) {
	actual := model.GridConfig{MaxLength: model.NewLengthInEpochs(3)}
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
			config: model.SearcherConfig{
				GridConfig: &model.GridConfig{
					MaxLength:           model.NewLengthInBatches(300),
					MaxConcurrentTrials: 2,
				},
			},
			hparams: generateHyperparameters([]int{2, 1, 3}),
		},
	}

	runValueSimulationTestCases(t, testCases)
}
