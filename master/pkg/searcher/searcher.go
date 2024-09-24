package searcher

import (
	"encoding/json"
	"math"
	"sync"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/nprand"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
)

// PartialUnits represent partial epochs, batches or records where the Unit is implied.
type PartialUnits float64

type (
	// SearcherState encapsulates all persisted searcher state.
	SearcherState struct {
		TrialsRequested int               `json:"trials_requested"`
		TrialsCreated   map[int32]bool    `json:"trials_created"`
		TrialsClosed    map[int32]bool    `json:"trials_closed"`
		Exits           map[int32]bool    `json:"exits"`
		Cancels         map[int32]bool    `json:"cancels"`
		Failures        map[int32]bool    `json:"failures"`
		TrialProgress   map[int32]float64 `json:"trial_progress"`

		Rand *nprand.State `json:"rand"`

		SearchMethodState json.RawMessage `json:"search_method_state"`
	}

	// Searcher encompasses the state as the searcher progresses using the provided search method.
	Searcher struct {
		mu sync.Mutex

		hparams expconf.Hyperparameters
		method  SearchMethod
		state   SearcherState
	}
)

// NewSearcher creates a new Searcher configured with the provided searcher config.
func NewSearcher(seed uint32, method SearchMethod, hparams expconf.Hyperparameters) *Searcher {
	return &Searcher{
		hparams: hparams,
		method:  method,
		state: SearcherState{
			Rand:          nprand.New(seed),
			TrialsCreated: map[int32]bool{},
			TrialsClosed:  map[int32]bool{},
			Exits:         map[int32]bool{},
			Cancels:       map[int32]bool{},
			Failures:      map[int32]bool{},
			TrialProgress: map[int32]float64{},
		},
	}
}

func (s *Searcher) context() context {
	return context{rand: s.state.Rand, hparams: s.hparams}
}

// InitialTrials returns the initial trials the searcher intends to create at the start of a search.
// This should be called only once after the searcher has been created.
func (s *Searcher) InitialTrials() ([]Action, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	creates, err := s.method.initialTrials(s.context())
	if err != nil {
		return nil, errors.Wrap(err, "error while fetching initial operations of search method")
	}
	s.record(creates)
	return creates, nil
}

// TrialCreated informs the searcher that a new trial has been created.
func (s *Searcher) TrialCreated(trialID int32, action Create) ([]Action, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.state.TrialsCreated[trialID] = true
	s.state.TrialProgress[trialID] = 0
	operations, err := s.method.trialCreated(s.context(), trialID, action)
	if err != nil {
		return nil, errors.Wrapf(err,
			"error while handling a trial created event: %d", trialID)
	}
	s.record(operations)
	return operations, nil
}

// TrialIsCreated returns true if the creation has been recorded with a TrialCreated call.
func (s *Searcher) TrialIsCreated(trialID int32) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.state.TrialsCreated[trialID]
}

// TrialExitedEarly informs the searcher that a trial has exited early.
func (s *Searcher) TrialExitedEarly(
	trialID int32, exitedReason model.ExitedReason,
) ([]Action, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state.Exits[trialID] {
		// If a trial reports an early exit twice, just ignore it (it can be convenient for each
		// rank to be allowed to report it without synchronization).
		return nil, nil
	}

	switch exitedReason {
	case model.InvalidHP, model.InitInvalidHP:
		delete(s.state.TrialProgress, trialID)
	case model.UserCanceled:
		s.state.Cancels[trialID] = true
	case model.Errored:
		// Only workload.Errored is considered a failure (since failures cause an experiment
		// to be in the failed state).
		s.state.Failures[trialID] = true
	}
	operations, err := s.method.trialExitedEarly(s.context(), trialID, exitedReason)
	if err != nil {
		return nil, errors.Wrapf(err, "error relaying trial exited early to trial %d", trialID)
	}
	s.state.Exits[trialID] = true
	s.record(operations)

	if s.state.TrialsRequested == len(s.state.TrialsClosed) {
		shutdown := Shutdown{Failure: len(s.state.Failures) >= s.state.TrialsRequested}
		s.record([]Action{shutdown})
		operations = append(operations, shutdown)
	}

	return operations, nil
}

// SetTrialProgress informs the searcher of the progress of a given trial.
func (s *Searcher) SetTrialProgress(trialID int32, progress float64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.state.TrialProgress[trialID] = progress
}

// ValidationCompleted informs the searcher that a validation for the trial was completed.
func (s *Searcher) ValidationCompleted(
	trialID int32, metrics map[string]interface{},
) ([]Action, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	operations, err := s.method.validationCompleted(s.context(), trialID, metrics)
	if err != nil {
		return nil, errors.Wrapf(err, "error while handling a validation completed event: %d", trialID)
	}
	s.record(operations)
	return operations, nil
}

// TrialExited informs the searcher that a trial has exited.
func (s *Searcher) TrialExited(trialID int32) ([]Action, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.state.TrialsClosed[trialID] = true
	actions, err := s.method.trialExited(s.context(), trialID)
	if err != nil {
		return nil, errors.Wrapf(err, "error while handling a trial closed event: %d", trialID)
	}
	s.record(actions)

	if s.state.TrialsRequested == len(s.state.TrialsClosed) {
		shutdown := Shutdown{
			Cancel:  len(s.state.Cancels) >= s.state.TrialsRequested,
			Failure: len(s.state.Failures) >= s.state.TrialsRequested,
		}
		s.record([]Action{shutdown})
		actions = append(actions, shutdown)
	}

	return actions, nil
}

// TrialIsClosed returns true if the close has been recorded with a TrialExited call.
func (s *Searcher) TrialIsClosed(trialID int32) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.state.TrialsClosed[trialID]
}

// Progress returns experiment progress as a float between 0.0 and 1.0.
func (s *Searcher) Progress() float64 {
	s.mu.Lock()
	defer s.mu.Unlock()

	progress := s.method.progress(s.state.TrialProgress, s.state.TrialsClosed)
	if math.IsNaN(progress) || math.IsInf(progress, 0) {
		return 0.0
	}
	return progress
}

// Record records actions that were requested by the searcher for a specific trial.
func (s *Searcher) Record(ops []Action) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.record(ops)
}

func (s *Searcher) record(ops []Action) {
	for _, op := range ops {
		if _, ok := op.(Create); ok {
			s.state.TrialsRequested++
		}
	}
}

// Snapshot returns a searchers current state.
func (s *Searcher) Snapshot() (json.RawMessage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	b, err := s.method.Snapshot()
	if err != nil {
		return nil, errors.Wrap(err, "failed to save search method")
	}
	s.state.SearchMethodState = b
	return json.Marshal(&s.state)
}

// Restore loads a searcher from prior state.
func (s *Searcher) Restore(state json.RawMessage) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := json.Unmarshal(state, &s.state); err != nil {
		return errors.Wrap(err, "failed to unmarshal searcher snapshot")
	}
	return s.method.Restore(s.state.SearchMethodState)
}
