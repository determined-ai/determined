package searcher

import (
	"encoding/json"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/workload"
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
		model.RandomConfig
		randomSearchState
	}
)

func newRandomSearch(config model.RandomConfig) SearchMethod {
	return &randomSearch{
		RandomConfig: config,
		randomSearchState: randomSearchState{
			SearchMethodType: RandomSearch,
		},
	}
}

func newSingleSearch(config model.SingleConfig) SearchMethod {
	return &randomSearch{
		RandomConfig: model.RandomConfig{MaxTrials: 1, MaxLength: config.MaxLength},
		randomSearchState: randomSearchState{
			SearchMethodType: SingleSearch,
		},
	}
}

func (s *randomSearch) initialOperations(ctx context) ([]Operation, error) {
	var ops []Operation
	initialTrials := s.MaxTrials
	if s.MaxConcurrentTrials > 0 {
		initialTrials = min(s.MaxTrials, s.MaxConcurrentTrials)
	}
	for trial := 0; trial < initialTrials; trial++ {
		create := NewCreate(ctx.rand, sampleAll(ctx.hparams, ctx.rand), model.TrialWorkloadSequencerType)
		ops = append(ops, create)
		ops = append(ops, NewValidateAfter(create.RequestID, s.MaxLength))
		ops = append(ops, NewClose(create.RequestID))
		s.CreatedTrials++
		s.PendingTrials++
	}
	return ops, nil
}

func (s *randomSearch) progress(trialProgress map[model.RequestID]model.PartialUnits) float64 {
	if s.MaxConcurrentTrials > 0 && s.PendingTrials > s.MaxConcurrentTrials {
		panic("pending trials is greater than max_concurrent_trials")
	}
	unitsCompleted := sumTrialLengths(trialProgress)
	unitsExpected := s.MaxLength.Units * s.MaxTrials
	return float64(unitsCompleted) / float64(unitsExpected)
}

// trialExitedEarly creates a new trial upon receiving an InvalidHP workload.
// Otherwise, it does nothing since actions are not taken based on search status.
func (s *randomSearch) trialExitedEarly(
	ctx context, requestID model.RequestID, exitedReason workload.ExitedReason,
) ([]Operation, error) {
	s.PendingTrials--
	if exitedReason == workload.InvalidHP {
		var ops []Operation
		create := NewCreate(ctx.rand, sampleAll(ctx.hparams, ctx.rand), model.TrialWorkloadSequencerType)
		ops = append(ops, create)
		ops = append(ops, NewValidateAfter(create.RequestID, s.MaxLength))
		ops = append(ops, NewClose(create.RequestID))
		// We don't increment CreatedTrials here because this trial is replacing the invalid trial.
		s.PendingTrials++
		return ops, nil
	}
	return nil, nil
}

func (s *randomSearch) trialClosed(ctx context, requestID model.RequestID) ([]Operation, error) {
	s.PendingTrials--
	var ops []Operation
	if s.CreatedTrials < s.MaxTrials {
		create := NewCreate(ctx.rand, sampleAll(ctx.hparams, ctx.rand), model.TrialWorkloadSequencerType)
		ops = append(ops, create)
		ops = append(ops, NewValidateAfter(create.RequestID, s.MaxLength))
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
