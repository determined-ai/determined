package searcher

import (
	"encoding/json"
	"fmt"
	"math"

	"github.com/determined-ai/determined/master/pkg/mathx"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
)

type (
	// gridSearchState stores the state for grid. The state will track the remaining hp settings
	// that have yet to be created for evaluation.  PendingTrials tracks how many trials have
	// active workloads and is used to check max_concurrent_trials for the searcher is respected.
	// Tracking searcher type on restart gives us the ability to differentiate grid searches
	// in a shim if needed.
	gridSearchState struct {
		PendingTrials    int              `json:"pending_trials"`
		RemainingTrials  []HParamSample   `json:"remaining_trials"`
		SearchMethodType SearchMethodType `json:"search_method_type"`
	}
	// gridSearch corresponds to a grid search method. A grid of hyperparameter configs is built. Then,
	// one trial is generated per point on the grid and trained for the specified number of steps.
	gridSearch struct {
		defaultSearchMethod
		expconf.GridConfig
		gridSearchState
		trials int
	}
)

func newGridSearch(config expconf.GridConfig) SearchMethod {
	return &gridSearch{
		GridConfig: config,
		gridSearchState: gridSearchState{
			SearchMethodType: GridSearch,
			RemainingTrials:  make([]HParamSample, 0),
		},
	}
}

func (s *gridSearch) initialOperations(ctx context) ([]Operation, error) {
	grid := newHyperparameterGrid(ctx.hparams)
	s.trials = len(grid)
	s.RemainingTrials = append(s.RemainingTrials, grid...)
	initialTrials := s.trials
	if s.MaxConcurrentTrials() > 0 {
		initialTrials = mathx.Min(s.trials, s.MaxConcurrentTrials())
	}
	var ops []Operation
	for trial := 0; trial < initialTrials; trial++ {
		params := s.RemainingTrials[len(s.RemainingTrials)-1]
		s.RemainingTrials = s.RemainingTrials[:len(s.RemainingTrials)-1]
		create := NewCreate(ctx.rand, params, model.TrialWorkloadSequencerType)
		ops = append(ops, create)
		ops = append(ops, NewValidateAfter(create.RequestID, s.MaxLength().Units))
		ops = append(ops, NewClose(create.RequestID))
		s.PendingTrials++
	}
	return ops, nil
}

func (s *gridSearch) progress(
	trialProgress map[model.RequestID]PartialUnits,
	trialsClosed map[model.RequestID]bool,
) float64 {
	if s.MaxConcurrentTrials() > 0 && s.PendingTrials > s.MaxConcurrentTrials() {
		panic("pending trials is greater than max_concurrent_trials")
	}
	// Progress is calculated as follows:
	//   - InvalidHP trials contribute max_length units since they represent one config within the grid
	//     and are not replaced with a new config as with random search
	//   - Other early-exit trials contribute max_length units
	//   - In progress trials contribute units trained
	unitsCompleted := 0.
	// trialsClosed includes InvalidHP trials and other exited trials
	for range trialsClosed {
		unitsCompleted += float64(s.MaxLength().Units)
	}
	// trialProgress records units trained for all trials except for InvalidHP trials.
	// This can overlap with trialsClosed so we need to be sure to not double count.
	for k, v := range trialProgress {
		if !trialsClosed[k] {
			unitsCompleted += float64(v)
		}
	}
	unitsExpected := s.MaxLength().Units * uint64(s.trials)
	return unitsCompleted / float64(unitsExpected)
}

// trialExitedEarly does nothing since grid does not take actions based on
// search status or progress.
func (s *gridSearch) trialExitedEarly(
	ctx context, requestID model.RequestID, exitedReason model.ExitedReason,
) ([]Operation, error) {
	return nil, nil
}

func (s *gridSearch) trialClosed(ctx context, _ model.RequestID) ([]Operation, error) {
	s.PendingTrials--
	var ops []Operation
	if len(s.RemainingTrials) > 0 {
		params := s.RemainingTrials[len(s.RemainingTrials)-1]
		s.RemainingTrials = s.RemainingTrials[:len(s.RemainingTrials)-1]
		create := NewCreate(ctx.rand, params, model.TrialWorkloadSequencerType)
		ops = append(ops, create)
		ops = append(ops, NewValidateAfter(create.RequestID, s.MaxLength().Units))
		ops = append(ops, NewClose(create.RequestID))
		s.PendingTrials++
	}
	return ops, nil
}

func newHyperparameterGrid(params expconf.Hyperparameters) []HParamSample {
	var axes []gridAxis
	// Use params.Each for consistent ordering.
	params.Each(func(name string, param expconf.HyperparameterV0) {
		route := []string{name}
		axes = append(axes, getGridAxes(route, param)...)
	})
	points := cartesianProduct(axes)
	var samples []HParamSample
	for _, axisValues := range points {
		sample := HParamSample{}
		for _, av := range axisValues {
			applyToSample(av.Route, av.Value, sample)
		}
		samples = append(samples, sample)
	}
	return samples
}

// axisValue is a single value a parameter can take, plus the route to set it if it is nested.
type axisValue struct {
	Route []string
	Value interface{}
}

// gridAxis is a set of possible axisValues for a single hyperparameter.
type gridAxis = []axisValue

func applyToSample(route []string, val interface{}, sample HParamSample) HParamSample {
	key := route[0]
	if len(route) == 1 {
		// end of the route
		sample[key] = val
		return sample
	}
	// make sure subsample is present
	if _, ok := sample[key]; !ok {
		sample[key] = HParamSample{}
	}
	// descend one layer and recurse
	subsample := sample[key].(HParamSample)
	subsample = applyToSample(route[1:], val, subsample)
	sample[key] = subsample
	return sample
}

// Turns lists of all-values-per-axis into all combinations of one-value-per-axis.
// Technically, both input and output are [][]axisValue, but semantically they are different.
func cartesianProduct(axes []gridAxis) [][]axisValue {
	switch {
	case len(axes) == 0:
		return nil
	case len(axes) == 1:
		axis := axes[0]
		cross := make([][]axisValue, 0, len(axis))
		for _, value := range axis {
			cross = append(cross, []axisValue{value})
		}
		return cross
	default:
		right := cartesianProduct(axes[1:])
		left := axes[0]
		cross := make([][]axisValue, 0, len(left)*len(right))
		for _, lValue := range left {
			for _, rValue := range right {
				var duplicate []axisValue
				duplicate = append(duplicate, lValue)
				duplicate = append(duplicate, rValue...)
				cross = append(cross, duplicate)
			}
		}
		return cross
	}
}

// Return a list of all axes represented by a hyperparameter.  Non-nested hyperparameters will
// return a single axis, which is simply all values that parameter can take in the search.  Nested
// hyperparameters will return one axis for every subordinate parameter.
func getGridAxes(route []string, h expconf.Hyperparameter) []gridAxis {
	switch {
	case h.RawConstHyperparameter != nil:
		p := *h.RawConstHyperparameter
		axis := []axisValue{{route, p.Val()}}
		return []gridAxis{axis}
	case h.RawIntHyperparameter != nil:
		p := *h.RawIntHyperparameter
		// Dereferencing is okay because initialization of GridSearch has checked p.Count is non-nil.
		count := *p.Count()

		// Clamp to the maximum number of integers in the range.
		count = mathx.Min(count, p.Maxval()-p.Minval()+1)

		axis := make([]axisValue, count)
		// Includes temporary validation, for invalid count
		if count == 1 {
			axis[0] = axisValue{route, int(math.Round(float64(p.Minval()+p.Maxval()) / 2.0))}
		} else {
			for i := 0; i < count; i++ {
				axis[i] = axisValue{route, int(
					math.Round(
						float64(p.Minval()) + float64(i*(p.Maxval()-p.Minval()))/float64(count-1),
					),
				)}
			}
		}
		return []gridAxis{axis}
	case h.RawDoubleHyperparameter != nil:
		p := *h.RawDoubleHyperparameter
		// Dereferencing is okay because initialization of GridSearch has checked p.Count is non-nil.
		count := *p.Count()
		axis := make([]axisValue, count)

		if count == 1 {
			axis[0] = axisValue{route, (p.Minval() + p.Maxval()) / 2.0}
		} else {
			for i := 0; i < count; i++ {
				axis[i] = axisValue{
					route, p.Minval() + float64(i)*(p.Maxval()-p.Minval())/float64(count-1),
				}
			}
		}
		return []gridAxis{axis}
	case h.RawLogHyperparameter != nil:
		p := *h.RawLogHyperparameter
		count := *p.Count()
		axis := make([]axisValue, count)

		// Includes temporary validation, for invalid count.
		if count == 1 {
			axis[0] = axisValue{route, math.Pow(p.Base(), (p.Minval()+p.Maxval())/2.0)}
		} else {
			for i := 0; i < count; i++ {
				axis[i] = axisValue{route, math.Pow(
					p.Base(), p.Minval()+float64(i)*(p.Maxval()-p.Minval())/float64(count-1),
				)}
			}
		}
		return []gridAxis{axis}
	case h.RawCategoricalHyperparameter != nil:
		p := *h.RawCategoricalHyperparameter
		axis := make([]axisValue, len(p.Vals()))
		for i, val := range p.Vals() {
			axis[i] = axisValue{route, val}
		}
		return []gridAxis{axis}
	case h.RawNestedHyperparameter != nil:
		axes := []gridAxis{}
		// Use h.Each for deterministic ordering.
		nested := expconf.Hyperparameters(*h.RawNestedHyperparameter)
		nested.Each(func(name string, subparam expconf.HyperparameterV0) {
			// make a completely clean copy of route
			var subroute []string
			subroute = append(subroute, route...)
			// extend subroute with this key
			subroute = append(subroute, name)
			axes = append(axes, getGridAxes(subroute, subparam)...)
		})
		return axes
	default:
		panic(fmt.Sprintf("unexpected hyperparameter type %+v", h))
	}
}

func (s *gridSearch) Snapshot() (json.RawMessage, error) {
	return json.Marshal(s.gridSearchState)
}

func (s *gridSearch) Restore(state json.RawMessage) error {
	if state == nil {
		return nil
	}
	return json.Unmarshal(state, &s.gridSearchState)
}
