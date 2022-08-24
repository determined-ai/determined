package searcher

import (
	"encoding/json"

	"github.com/determined-ai/determined/master/pkg/mathx"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
)

type (
	// randomSearchState stores the state for random.  Since not all trials are always created at
	// initialization, we need to track CreatedTrials so we know whether we need to create more
	// trials when workloads complete so that we reach MaxTrials.  PendingTrials tracks active
	// workloads and is used to check max_concurrent_trials for the searcher is respected.
	// Tracking searcher type on restart gives us the ability to differentiate random searches
	// in a shim if needed.
	randomSearchState struct {
		CreatedTrials    int              `json:"created_trials"`
		PendingTrials    int              `json:"pending_trials"`
		SearchMethodType SearchMethodType `json:"search_method_type"`
	}
	// randomSearch corresponds to the standard random search method. Each random trial configuration
	// is trained for the specified number of steps, and then validation metrics are computed.
	randomSearch struct {
		defaultSearchMethod
		expconf.RandomConfig
		randomSearchState
	}
)

func newRandomSearch(config expconf.RandomConfig) SearchMethod {
	return &randomSearch{
		RandomConfig: config,
		randomSearchState: randomSearchState{
			SearchMethodType: RandomSearch,
		},
	}
}

func newSingleSearch(config expconf.SingleConfig) SearchMethod {
	return &randomSearch{
		RandomConfig: schemas.WithDefaults(expconf.RandomConfig{
			RawMaxTrials:           ptrs.Ptr(1),
			RawMaxLength:           ptrs.Ptr(config.MaxLength()),
			RawMaxConcurrentTrials: ptrs.Ptr(1),
		}).(expconf.RandomConfig),
		randomSearchState: randomSearchState{
			SearchMethodType: SingleSearch,
		},
	}
}

func (s *randomSearch) initialOperations(ctx context) ([]Operation, error) {
	var ops []Operation
	initialTrials := s.MaxTrials()
	if s.MaxConcurrentTrials() > 0 {
		initialTrials = mathx.Min(s.MaxTrials(), s.MaxConcurrentTrials())
	}
	for trial := 0; trial < initialTrials; trial++ {
		create := NewCreate(ctx.rand, sampleAll(ctx.hparams, ctx.rand), model.TrialWorkloadSequencerType)
		ops = append(ops, create)
		ops = append(ops, NewValidateAfter(create.RequestID, s.MaxLength().Units))
		ops = append(ops, NewClose(create.RequestID))
		s.CreatedTrials++
		s.PendingTrials++
	}
	return ops, nil
}

func (s *randomSearch) progress(
	trialProgress map[model.RequestID]PartialUnits,
	trialsClosed map[model.RequestID]bool,
) float64 {
	if s.MaxConcurrentTrials() > 0 && s.PendingTrials > s.MaxConcurrentTrials() {
		panic("pending trials is greater than max_concurrent_trials")
	}
	// Progress is calculated as follows:
	//   - InvalidHP trials contribute 0 since we do not count them against max_trials budget and are
	//     replaced with another randomly sampled config
	//   - Other early-exit trials contribute max_length units
	//   - In progress trials contribute units trained
	unitsCompleted := 0.
	// trialProgress records units trained for all trials except for InvalidHP trials.
	for k, v := range trialProgress {
		if trialsClosed[k] {
			unitsCompleted += float64(s.MaxLength().Units)
		} else {
			unitsCompleted += float64(v)
		}
	}
	unitsExpected := s.MaxLength().Units * uint64(s.MaxTrials())
	return unitsCompleted / float64(unitsExpected)
}

// trialExitedEarly creates a new trial upon receiving an InvalidHP workload.
// Otherwise, it does nothing since actions are not taken based on search status.
func (s *randomSearch) trialExitedEarly(
	ctx context, requestID model.RequestID, exitedReason model.ExitedReason,
) ([]Operation, error) {
	s.PendingTrials--
	if s.SearchMethodType == RandomSearch {
		if exitedReason == model.InvalidHP || exitedReason == model.InitInvalidHP {
			// We decrement CreatedTrials here because this trial is replacing the invalid trial.
			// It will be created by trialClosed when the close is received for this trial.
			s.CreatedTrials--
			return nil, nil
		}
	}
	return nil, nil
}

func (s *randomSearch) trialClosed(ctx context, requestID model.RequestID) ([]Operation, error) {
	s.PendingTrials--
	var ops []Operation
	if s.CreatedTrials < s.MaxTrials() {
		create := NewCreate(ctx.rand, sampleAll(ctx.hparams, ctx.rand), model.TrialWorkloadSequencerType)
		ops = append(ops, create)
		ops = append(ops, NewValidateAfter(create.RequestID, s.MaxLength().Units))
		ops = append(ops, NewClose(create.RequestID))
		s.CreatedTrials++
		s.PendingTrials++
	}
	return ops, nil
}

func (s *randomSearch) Snapshot() (json.RawMessage, error) {
	return json.Marshal(s.randomSearchState)
}

func (s *randomSearch) Restore(state json.RawMessage) error {
	if state == nil {
		return nil
	}
	return json.Unmarshal(state, &s.randomSearchState)
}
