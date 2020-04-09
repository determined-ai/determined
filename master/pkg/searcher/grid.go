package searcher

import (
	"fmt"
	"math"

	"github.com/determined-ai/determined/master/pkg/model"
)

// gridSearch corresponds to a grid search method. A grid of hyperparameter configs is built. Then,
// one trial is generated per point on the grid and trained for the specified number of steps.
type gridSearch struct {
	model.GridConfig
	trials int
}

func newGridSearch(config model.GridConfig) SearchMethod {
	return &gridSearch{GridConfig: config}
}

func (s *gridSearch) initialOperations(ctx Context) {
	grid := newHyperparameterGrid(ctx.Hyperparameters())
	s.trials = len(grid)
	for _, params := range grid {
		trial := ctx.NewTrial(PreSampled(params))
		ctx.TrainAndValidate(trial, s.MaxSteps)
		ctx.CloseTrial(trial)
	}
}

func (s *gridSearch) progress(workloadsCompleted int) float64 {
	return float64(workloadsCompleted) / float64((s.MaxSteps+1)*s.trials)
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

func (s *gridSearch) trainCompleted(Context, RequestID, Workload) {}
func (s *gridSearch) validationCompleted(Context, RequestID, Workload, ValidationMetrics) error {
	return nil
}
