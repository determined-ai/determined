package searcher

import (
	"fmt"
	"math"

	"github.com/determined-ai/determined/master/pkg/model"
)

// gridSearch corresponds to a grid search method. A grid of hyperparameter configs is built. Then,
// one trial is generated per point on the grid and trained for the specified number of steps.
type gridSearch struct {
	defaultSearchMethod
	model.GridConfig
	trials int
}

func newGridSearch(config model.GridConfig) SearchMethod {
	return &gridSearch{GridConfig: config}
}

func (s *gridSearch) initialOperations(ctx context) ([]Operation, error) {
	var operations []Operation
	grid := newHyperparameterGrid(ctx.hparams)
	s.trials = len(grid)
	for _, params := range grid {
		create := NewCreate(ctx.rand, params, model.TrialWorkloadSequencerType)
		operations = append(operations, create)
		operations = append(operations, NewTrain(create.RequestID, s.MaxLength))
		operations = append(operations, NewValidate(create.RequestID))
		operations = append(operations, NewClose(create.RequestID))
	}
	return operations, nil
}

func (s *gridSearch) progress(unitsCompleted model.Length) float64 {
	return float64(unitsCompleted.Units) / float64(s.GridConfig.MaxLength.MultInt(s.trials).Units)
}

// trialExitedEarly does nothing since grid does not take actions based on
// search status or progress.
func (s *gridSearch) trialExitedEarly(context, RequestID) ([]Operation, error) {
	return nil, nil
}

func newHyperparameterGrid(params model.Hyperparameters) []hparamSample {
	var names []string
	var values [][]interface{}
	params.Each(func(name string, param model.Hyperparameter) {
		names = append(names, name)
		values = append(values, grid(param))
	})
	return cartesianProduct(names, values)
}

func cartesianProduct(names []string, valueSets [][]interface{}) []hparamSample {
	switch {
	case len(names) == 0:
		return nil
	case len(names) == 1:
		cross := make([]hparamSample, 0, len(valueSets[0]))
		for _, value := range valueSets[0] {
			cross = append(cross, hparamSample{names[0]: value})
		}
		return cross
	default:
		right := cartesianProduct(names[1:], valueSets[1:])
		name, left := names[0], valueSets[0]
		cross := make([]hparamSample, 0, len(left)*len(right))
		for _, lValue := range left {
			for _, rValue := range right {
				duplicate := make(hparamSample)
				for oKey, oValue := range rValue {
					duplicate[oKey] = oValue
				}
				duplicate[name] = lValue
				cross = append(cross, duplicate)
			}
		}
		return cross
	}
}

func grid(h model.Hyperparameter) []interface{} {
	switch {
	case h.ConstHyperparameter != nil:
		p := *h.ConstHyperparameter
		return []interface{}{p.Val}
	case h.IntHyperparameter != nil:
		p := *h.IntHyperparameter
		// Dereferencing is okay because initialization of GridSearch has checked p.Count is non-nil.
		count := *p.Count

		// Clamp to the maximum number of integers in the range.
		count = min(count, p.Maxval-p.Minval+1)

		vals := make([]interface{}, count)
		// Includes temporary validation, for invalid count
		if count == 1 {
			vals[0] = int(math.Round(float64(p.Minval+p.Maxval) / 2.0))
		} else {
			for i := 0; i < count; i++ {
				vals[i] = int(math.Round(float64(p.Minval) + float64(i*(p.Maxval-p.Minval))/float64(count-1)))
			}
		}
		return vals
	case h.DoubleHyperparameter != nil:
		p := *h.DoubleHyperparameter
		// Dereferencing is okay because initialization of GridSearch has checked p.Count is non-nil.
		count := *p.Count
		vals := make([]interface{}, count)

		if count == 1 {
			vals[0] = (p.Minval + p.Maxval) / 2.0
		} else {
			for i := 0; i < count; i++ {
				vals[i] = p.Minval + float64(i)*(p.Maxval-p.Minval)/float64(count-1)
			}
		}
		return vals
	case h.LogHyperparameter != nil:
		p := *h.LogHyperparameter
		count := *p.Count
		vals := make([]interface{}, count)

		// Includes temporary validation, for invalid count.
		if count == 1 {
			vals[0] = math.Pow(p.Base, (p.Minval+p.Maxval)/2.0)
		} else {
			for i := 0; i < count; i++ {
				vals[i] = math.Pow(p.Base, p.Minval+float64(i)*(p.Maxval-p.Minval)/float64(count-1))
			}
		}
		return vals
	case h.CategoricalHyperparameter != nil:
		p := *h.CategoricalHyperparameter
		return p.Vals
	default:
		panic(fmt.Sprintf("unexpected hyperparameter type %+v", h))
	}
}
