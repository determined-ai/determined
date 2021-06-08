package searcher

import (
	"encoding/json"
	"fmt"
	"math"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/master/pkg/workload"
)

type (
	// gridSearchState stores the state for grid. The state will track the remaining hp settings
	// that have yet to be created for evaluation.  PendingTrials tracks how many trials have
	// active workloads and is used to check max_concurrent_trials for the searcher is respected.
	// Tracking searcher type on restart gives us the ability to differentiate grid searches
	// in a shim if needed.
	gridSearchState struct {
		PendingTrials    int              `json:"pending_trials"`
		RemainingTrials  []hparamSample   `json:"remaining_trials"`
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
			RemainingTrials:  make([]hparamSample, 0),
		},
	}
}

func (s *gridSearch) initialOperations(ctx context) ([]Operation, error) {
	grid := newHyperparameterGrid(ctx.hparams)
	s.trials = len(grid)
	s.RemainingTrials = append(s.RemainingTrials, grid...)
	initialTrials := s.trials
	if s.MaxConcurrentTrials() > 0 {
		initialTrials = min(s.trials, s.MaxConcurrentTrials())
	}
	var ops []Operation
	for trial := 0; trial < initialTrials; trial++ {
		params := s.RemainingTrials[len(s.RemainingTrials)-1]
		s.RemainingTrials = s.RemainingTrials[:len(s.RemainingTrials)-1]
		create := NewCreate(ctx.rand, params, model.TrialWorkloadSequencerType)
		ops = append(ops, create)
		ops = append(ops, NewValidateAfter(create.RequestID, s.MaxLength()))
		ops = append(ops, NewClose(create.RequestID))
		s.PendingTrials++
	}
	return ops, nil
}

func (s *gridSearch) progress(trialProgress map[model.RequestID]model.PartialUnits) float64 {
	if s.MaxConcurrentTrials() > 0 && s.PendingTrials > s.MaxConcurrentTrials() {
		panic("pending trials is greater than max_concurrent_trials")
	}
	unitsCompleted := sumTrialLengths(trialProgress)
	unitsExpected := s.MaxLength().Units * s.trials
	return float64(unitsCompleted) / float64(unitsExpected)
}

// trialExitedEarly does nothing since grid does not take actions based on
// search status or progress.
func (s *gridSearch) trialExitedEarly(
	ctx context, requestID model.RequestID, exitedReason workload.ExitedReason,
) ([]Operation, error) {
	s.PendingTrials--
	var ops []Operation
	if len(s.RemainingTrials) > 0 {
		params := s.RemainingTrials[len(s.RemainingTrials)-1]
		s.RemainingTrials = s.RemainingTrials[:len(s.RemainingTrials)-1]
		create := NewCreate(ctx.rand, params, model.TrialWorkloadSequencerType)
		ops = append(ops, create)
		ops = append(ops, NewValidateAfter(create.RequestID, s.MaxLength()))
		ops = append(ops, NewClose(create.RequestID))
		s.PendingTrials++
	}
	return ops, nil
}

func (s *gridSearch) trialClosed(ctx context, _ model.RequestID) ([]Operation, error) {
	s.PendingTrials--
	var ops []Operation
	if len(s.RemainingTrials) > 0 {
		params := s.RemainingTrials[len(s.RemainingTrials)-1]
		s.RemainingTrials = s.RemainingTrials[:len(s.RemainingTrials)-1]
		create := NewCreate(ctx.rand, params, model.TrialWorkloadSequencerType)
		ops = append(ops, create)
		ops = append(ops, NewValidateAfter(create.RequestID, s.MaxLength()))
		ops = append(ops, NewClose(create.RequestID))
		s.PendingTrials++
	}
	return ops, nil
}

func newHyperparameterGrid(params expconf.Hyperparameters) []hparamSample {
	var names []string
	var values [][]interface{}
	params.Each(func(name string, param expconf.Hyperparameter) {
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

func grid(h expconf.Hyperparameter) []interface{} {
	switch {
	case h.RawConstHyperparameter != nil:
		p := *h.RawConstHyperparameter
		return []interface{}{p.Val()}
	case h.RawIntHyperparameter != nil:
		p := *h.RawIntHyperparameter
		// Dereferencing is okay because initialization of GridSearch has checked p.Count is non-nil.
		count := *p.Count()

		// Clamp to the maximum number of integers in the range.
		count = min(count, p.Maxval()-p.Minval()+1)

		vals := make([]interface{}, count)
		// Includes temporary validation, for invalid count
		if count == 1 {
			vals[0] = int(math.Round(float64(p.Minval()+p.Maxval()) / 2.0))
		} else {
			for i := 0; i < count; i++ {
				vals[i] = int(
					math.Round(
						float64(p.Minval()) + float64(i*(p.Maxval()-p.Minval()))/float64(count-1),
					),
				)
			}
		}
		return vals
	case h.RawDoubleHyperparameter != nil:
		p := *h.RawDoubleHyperparameter
		// Dereferencing is okay because initialization of GridSearch has checked p.Count is non-nil.
		count := *p.Count()
		vals := make([]interface{}, count)

		if count == 1 {
			vals[0] = (p.Minval() + p.Maxval()) / 2.0
		} else {
			for i := 0; i < count; i++ {
				vals[i] = p.Minval() + float64(i)*(p.Maxval()-p.Minval())/float64(count-1)
			}
		}
		return vals
	case h.RawLogHyperparameter != nil:
		p := *h.RawLogHyperparameter
		count := *p.Count()
		vals := make([]interface{}, count)

		// Includes temporary validation, for invalid count.
		if count == 1 {
			vals[0] = math.Pow(p.Base(), (p.Minval()+p.Maxval())/2.0)
		} else {
			for i := 0; i < count; i++ {
				vals[i] = math.Pow(
					p.Base(), p.Minval()+float64(i)*(p.Maxval()-p.Minval())/float64(count-1),
				)
			}
		}
		return vals
	case h.RawCategoricalHyperparameter != nil:
		p := *h.RawCategoricalHyperparameter
		return p.Vals()
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
