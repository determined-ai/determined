package searcher

import (
	"encoding/json"
	"fmt"
	"math"

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
		TrialsRequested     int                              `json:"trials_requested"`
		TrialsCreated       map[model.RequestID]bool         `json:"trials_created"`
		TrialsClosed        map[model.RequestID]bool         `json:"trials_closed"`
		Exits               map[model.RequestID]bool         `json:"exits"`
		Cancels             map[model.RequestID]bool         `json:"cancels"`
		Failures            map[model.RequestID]bool         `json:"failures"`
		TrialProgress       map[model.RequestID]PartialUnits `json:"trial_progress"`
		Shutdown            bool                             `json:"shutdown"`
		CompletedOperations map[string]ValidateAfter         `json:"completed_operations"`

		Rand *nprand.State `json:"rand"`

		SearchMethodState json.RawMessage `json:"search_method_state"`
	}

	// Searcher encompasses the state as the searcher progresses using the provided search method.
	Searcher struct {
		hparams expconf.Hyperparameters
		method  SearchMethod
		SearcherState
	}
)

// NewSearcher creates a new Searcher configured with the provided searcher config.
func NewSearcher(seed uint32, method SearchMethod, hparams expconf.Hyperparameters) *Searcher {
	return &Searcher{
		hparams: hparams,
		method:  method,
		SearcherState: SearcherState{
			Rand:                nprand.New(seed),
			TrialsCreated:       map[model.RequestID]bool{},
			TrialsClosed:        map[model.RequestID]bool{},
			Exits:               map[model.RequestID]bool{},
			Cancels:             map[model.RequestID]bool{},
			Failures:            map[model.RequestID]bool{},
			TrialProgress:       map[model.RequestID]PartialUnits{},
			CompletedOperations: map[string]ValidateAfter{},
		},
	}
}

func (s *Searcher) context() context {
	return context{rand: s.Rand, hparams: s.hparams}
}

// InitialOperations return a set of initial operations that the searcher would like to take.
// This should be called only once after the searcher has been created.
func (s *Searcher) InitialOperations() ([]Operation, error) {
	operations, err := s.method.initialOperations(s.context())
	if err != nil {
		return nil, errors.Wrap(err, "error while fetching initial operations of search method")
	}
	s.Record(operations)
	return operations, nil
}

// TrialCreated informs the searcher that a trial has been created as a result of a Create
// operation.
func (s *Searcher) TrialCreated(requestID model.RequestID) ([]Operation, error) {
	s.TrialsCreated[requestID] = true
	s.TrialProgress[requestID] = 0
	operations, err := s.method.trialCreated(s.context(), requestID)
	if err != nil {
		return nil, errors.Wrapf(err,
			"error while handling a trial created event: %s", requestID)
	}
	s.Record(operations)
	return operations, nil
}

// TrialExitedEarly indicates to the searcher that the trial with the given trialID exited early.
func (s *Searcher) TrialExitedEarly(
	requestID model.RequestID, exitedReason model.ExitedReason,
) ([]Operation, error) {
	if s.Exits[requestID] {
		// If a trial reports an early exit twice, just ignore it (it can be convenient for each
		// rank to be allowed to report it without synchronization).
		return nil, nil
	}

	switch exitedReason {
	case model.InvalidHP, model.InitInvalidHP:
		delete(s.TrialProgress, requestID)
	case model.UserCanceled:
		s.Cancels[requestID] = true
	case model.Errored:
		// Only workload.Errored is considered a failure (since failures cause an experiment
		// to be in the failed state).
		s.Failures[requestID] = true
	}
	operations, err := s.method.trialExitedEarly(s.context(), requestID, exitedReason)
	if err != nil {
		return nil, errors.Wrapf(err, "error relaying trial exited early to trial %d", requestID)
	}
	s.Exits[requestID] = true
	s.Record(operations)
	return operations, nil
}

// SetTrialProgress informs the searcher of the progress of a given trial.
func (s *Searcher) SetTrialProgress(requestID model.RequestID, progress PartialUnits) {
	s.TrialProgress[requestID] = progress
}

// ValidationCompleted informs the searcher that a validation for the trial was completed.
func (s *Searcher) ValidationCompleted(
	requestID model.RequestID, metric float64, op ValidateAfter,
) ([]Operation, error) {
	if _, ok := s.CompletedOperations[op.String()]; ok {
		return nil, fmt.Errorf("operation %v was already completed", op)
	}

	operations, err := s.method.validationCompleted(s.context(), requestID, metric)
	if err != nil {
		return nil, errors.Wrapf(err, "error while handling a workload completed event: %s", requestID)
	}
	s.CompletedOperations[op.String()] = op
	s.Record(operations)
	return operations, nil
}

// TrialClosed informs the searcher that the trial has been closed as a result of a Close operation.
func (s *Searcher) TrialClosed(requestID model.RequestID) ([]Operation, error) {
	s.TrialsClosed[requestID] = true
	operations, err := s.method.trialClosed(s.context(), requestID)
	if err != nil {
		return nil, errors.Wrapf(err, "error while handling a trial closed event: %s", requestID)
	}
	s.Record(operations)
	if s.TrialsRequested == len(s.TrialsClosed) {
		shutdown := Shutdown{
			Cancel:  len(s.Cancels) >= s.TrialsRequested,
			Failure: len(s.Failures) >= s.TrialsRequested,
		}
		s.Record([]Operation{shutdown})
		operations = append(operations, shutdown)
	}
	return operations, nil
}

// Progress returns experiment progress as a float between 0.0 and 1.0.
func (s *Searcher) Progress() float64 {
	progress := s.method.progress(s.TrialProgress, s.TrialsClosed)
	if math.IsNaN(progress) || math.IsInf(progress, 0) {
		return 0.0
	}
	return progress
}

// Record records operations that were requested by the searcher for a specific trial.
func (s *Searcher) Record(ops []Operation) {
	for _, op := range ops {
		switch op.(type) {
		case Create:
			s.TrialsRequested++
		case Shutdown:
			s.Shutdown = true
		}
	}
}

// Snapshot returns a searchers current state.
func (s *Searcher) Snapshot() (json.RawMessage, error) {
	b, err := s.method.Snapshot()
	if err != nil {
		return nil, errors.Wrap(err, "failed to save search method")
	}
	s.SearcherState.SearchMethodState = b
	return json.Marshal(s.SearcherState)
}

// Restore loads a searcher from prior state.
func (s *Searcher) Restore(state json.RawMessage) error {
	if err := json.Unmarshal(state, &s.SearcherState); err != nil {
		return errors.Wrap(err, "failed to unmarshal searcher snapshot")
	}
	return s.method.Restore(s.SearchMethodState)
}
