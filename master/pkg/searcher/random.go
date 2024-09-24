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
	// trials and is used to check max_concurrent_trials for the searcher is respected.
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
			RawMaxConcurrentTrials: ptrs.Ptr(1),
		}),
		randomSearchState: randomSearchState{
			SearchMethodType: SingleSearch,
		},
	}
}

func (s *randomSearch) initialTrials(ctx context) ([]Action, error) {
	var actions []Action
	initialTrials := s.MaxTrials()
	if s.MaxConcurrentTrials() > 0 {
		initialTrials = mathx.Min(s.MaxTrials(), s.MaxConcurrentTrials())
	}
	for trial := 0; trial < initialTrials; trial++ {
		create := NewCreate(ctx.rand, sampleAll(ctx.hparams, ctx.rand))
		actions = append(actions, create)
		s.CreatedTrials++
		s.PendingTrials++
	}
	return actions, nil
}

func (s *randomSearch) progress(
	trialProgress map[int32]float64,
	trialsClosed map[int32]bool,
) float64 {
	if s.MaxConcurrentTrials() > 0 && s.PendingTrials > s.MaxConcurrentTrials() {
		panic("pending trials is greater than max_concurrent_trials")
	}
	// Progress is calculated as follows:
	//   - InvalidHP trials contribute 0 since we do not count them against max_trials budget and are
	//     replaced with another randomly sampled config
	//   - Other early-exit trials contribute max_length units
	//   - In progress trials contribute units trained
	// trialsProgress records units trained for all trials except for InvalidHP trials.
	trialProgresses := 0.

	for k, v := range trialProgress {
		if trialsClosed[k] {
			trialProgresses += 1.0
		} else {
			trialProgresses += v
		}
	}

	return trialProgresses / float64(len(trialProgress))
}

// trialExitedEarly creates a new trial upon receiving an InvalidHP workload.
// Otherwise, it does nothing since actions are not taken based on search status.
func (s *randomSearch) trialExitedEarly(
	ctx context, trialID int32, exitedReason model.ExitedReason,
) ([]Action, error) {
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

func (s *randomSearch) trialExited(ctx context, trialID int32) ([]Action, error) {
	s.PendingTrials--
	var actions []Action
	if s.CreatedTrials < s.MaxTrials() {
		create := NewCreate(ctx.rand, sampleAll(ctx.hparams, ctx.rand))
		actions = append(actions, create)
		s.CreatedTrials++
		s.PendingTrials++
	}
	return actions, nil
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

func (s *randomSearch) Type() SearchMethodType {
	return s.SearchMethodType
}
