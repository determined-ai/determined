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
		RunsRequested int               `json:"runs_requested"`
		RunsCreated   map[int32]bool    `json:"runs_created"`
		RunsClosed    map[int32]bool    `json:"runs_closed"`
		Exits         map[int32]bool    `json:"exits"`
		Cancels       map[int32]bool    `json:"cancels"`
		Failures      map[int32]bool    `json:"failures"`
		RunProgress   map[int32]float64 `json:"run_progress"`

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
			Rand:        nprand.New(seed),
			RunsCreated: map[int32]bool{},
			RunsClosed:  map[int32]bool{},
			Exits:       map[int32]bool{},
			Cancels:     map[int32]bool{},
			Failures:    map[int32]bool{},
			RunProgress: map[int32]float64{},
		},
	}
}

func (s *Searcher) context() context {
	return context{rand: s.state.Rand, hparams: s.hparams}
}

// xxx: comment
// This should be called only once after the searcher has been created.
func (s *Searcher) InitialRuns() ([]Action, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	creates, err := s.method.initialRuns(s.context())
	if err != nil {
		return nil, errors.Wrap(err, "error while fetching initial operations of search method")
	}
	s.record(creates)
	return creates, nil
}

// xxx: comment
func (s *Searcher) RunCreated(runID int32, action Create) ([]Action, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.state.RunsCreated[runID] = true
	s.state.RunProgress[runID] = 0
	operations, err := s.method.runCreated(s.context(), runID, action)
	if err != nil {
		return nil, errors.Wrapf(err,
			"error while handling a trial created event: %d", runID)
	}
	s.record(operations)
	return operations, nil
}

// RunIsCreated returns true if the creation has been recorded with a TrialCreated call.
func (s *Searcher) RunIsCreated(runID int32) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.state.RunsCreated[runID]
}

// xxx: comment
func (s *Searcher) RunExitedEarly(
	runID int32, exitedReason model.ExitedReason,
) ([]Action, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state.Exits[runID] {
		// If a trial reports an early exit twice, just ignore it (it can be convenient for each
		// rank to be allowed to report it without synchronization).
		return nil, nil
	}

	switch exitedReason {
	case model.InvalidHP, model.InitInvalidHP:
		delete(s.state.RunProgress, runID)
	case model.UserCanceled:
		s.state.Cancels[runID] = true
	case model.Errored:
		// Only workload.Errored is considered a failure (since failures cause an experiment
		// to be in the failed state).
		s.state.Failures[runID] = true
	}
	operations, err := s.method.runExitedEarly(s.context(), runID, exitedReason)
	if err != nil {
		return nil, errors.Wrapf(err, "error relaying trial exited early to trial %d", runID)
	}
	s.state.Exits[runID] = true
	s.record(operations)

	if s.state.RunsRequested == len(s.state.RunsClosed) {
		shutdown := Shutdown{Failure: len(s.state.Failures) >= s.state.RunsRequested}
		s.record([]Action{shutdown})
		operations = append(operations, shutdown)
	}

	return operations, nil
}

// SetRunProgress informs the searcher of the progress of a given trial.
func (s *Searcher) SetRunProgress(runID int32, progress float64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.state.RunProgress[runID] = progress
}

// ValidationCompleted informs the searcher that a validation for the trial was completed.
func (s *Searcher) ValidationCompleted(
	runID int32, metrics map[string]interface{},
) ([]Action, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	operations, err := s.method.validationCompleted(s.context(), runID, metrics)
	if err != nil {
		return nil, errors.Wrapf(err, "error while handling a validation completed event: %d", runID)
	}
	s.record(operations)
	return operations, nil
}

// xxx: comment
func (s *Searcher) RunClosed(runID int32) ([]Action, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.state.RunsClosed[runID] = true
	actions, err := s.method.runClosed(s.context(), runID)
	if err != nil {
		return nil, errors.Wrapf(err, "error while handling a trial closed event: %d", runID)
	}
	s.record(actions)

	if s.state.RunsRequested == len(s.state.RunsClosed) {
		shutdown := Shutdown{
			Cancel:  len(s.state.Cancels) >= s.state.RunsRequested,
			Failure: len(s.state.Failures) >= s.state.RunsRequested,
		}
		s.record([]Action{shutdown})
		actions = append(actions, shutdown)
	}

	return actions, nil
}

// TrialIsClosed returns true if the close has been recorded with a RunClosed call.
func (s *Searcher) TrialIsClosed(runID int32) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.state.RunsClosed[runID]
}

// Progress returns experiment progress as a float between 0.0 and 1.0.
func (s *Searcher) Progress() float64 {
	s.mu.Lock()
	defer s.mu.Unlock()

	progress := s.method.progress(s.state.RunProgress, s.state.RunsClosed)
	if math.IsNaN(progress) || math.IsInf(progress, 0) {
		return 0.0
	}
	return progress
}

// Record records operations that were requested by the searcher for a specific trial.
func (s *Searcher) Record(ops []Action) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.record(ops)
}

func (s *Searcher) record(ops []Action) {
	for _, op := range ops {
		switch op.(type) {
		case Create:
			s.state.RunsRequested++
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
