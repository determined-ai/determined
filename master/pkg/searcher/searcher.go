package searcher

import (
	"encoding/json"
	"fmt"
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

func unsupportedMethodError(method SearchMethod, unsupportedOp string) error {
	return fmt.Errorf("%T search method does not support %s", method, unsupportedOp)
}

func (s *Searcher) context() context {
	return context{rand: s.state.Rand, hparams: s.hparams}
}

// InitialOperations return a set of initial operations that the searcher would like to take.
// This should be called only once after the searcher has been created.
func (s *Searcher) InitialOperations() ([]Operation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	operations, err := s.method.initialOperations(s.context())
	if err != nil {
		return nil, errors.Wrap(err, "error while fetching initial operations of search method")
	}
	s.record(operations)
	return operations, nil
}

// TrialCreated informs the searcher that a trial has been created as a result of a Create
// operation.
func (s *Searcher) TrialCreated(requestID model.RequestID) ([]Operation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.state.TrialsCreated[requestID] = true
	s.state.TrialProgress[requestID] = 0
	operations, err := s.method.trialCreated(s.context(), requestID)
	if err != nil {
		return nil, errors.Wrapf(err,
			"error while handling a trial created event: %s", requestID)
	}
	s.record(operations)
	return operations, nil
}

// TrialIsCreated returns true if the creation has been recorded with a TrialCreated call.
func (s *Searcher) TrialIsCreated(requestID model.RequestID) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.state.TrialsCreated[requestID]
}

// TrialExitedEarly indicates to the searcher that the trial with the given trialID exited early.
func (s *Searcher) TrialExitedEarly(
	requestID model.RequestID, exitedReason model.ExitedReason,
) ([]Operation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state.Exits[requestID] {
		// If a trial reports an early exit twice, just ignore it (it can be convenient for each
		// rank to be allowed to report it without synchronization).
		return nil, nil
	}

	switch exitedReason {
	case model.InvalidHP, model.InitInvalidHP:
		delete(s.state.TrialProgress, requestID)
	case model.UserCanceled:
		s.state.Cancels[requestID] = true
	case model.Errored:
		// Only workload.Errored is considered a failure (since failures cause an experiment
		// to be in the failed state).
		s.state.Failures[requestID] = true
	}
	operations, err := s.method.trialExitedEarly(s.context(), requestID, exitedReason)
	if err != nil {
		return nil, errors.Wrapf(err, "error relaying trial exited early to trial %d", requestID)
	}
	s.state.Exits[requestID] = true
	s.record(operations)

	_, isCustom := s.method.(*customSearch)
	// For non-custom-search methods, you can assume that trials will be created immediately.
	if s.state.TrialsRequested == len(s.state.TrialsClosed) && !isCustom {
		shutdown := Shutdown{Failure: len(s.state.Failures) >= s.state.TrialsRequested}
		s.record([]Operation{shutdown})
		operations = append(operations, shutdown)
	}

	return operations, nil
}

// SetTrialProgress informs the searcher of the progress of a given trial.
func (s *Searcher) SetTrialProgress(requestID model.RequestID, progress PartialUnits) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if sMethod, ok := s.method.(*customSearch); ok {
		sMethod.trialProgress(s.context(), requestID, progress)
	}
	s.state.TrialProgress[requestID] = progress
}

// ValidationCompleted informs the searcher that a validation for the trial was completed.
func (s *Searcher) ValidationCompleted(
	requestID model.RequestID, metric interface{}, op ValidateAfter,
) ([]Operation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.state.CompletedOperations[op.String()]; ok {
		return nil, fmt.Errorf("operation %v was already completed", op)
	}

	operations, err := s.method.validationCompleted(s.context(), requestID, metric, op)
	if err != nil {
		return nil, errors.Wrapf(err, "error while handling a workload completed event: %s", requestID)
	}
	s.state.CompletedOperations[op.String()] = op
	s.record(operations)
	return operations, nil
}

// TrialClosed informs the searcher that the trial has been closed as a result of a Close operation.
func (s *Searcher) TrialClosed(requestID model.RequestID) ([]Operation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.state.TrialsClosed[requestID] = true
	operations, err := s.method.trialClosed(s.context(), requestID)
	if err != nil {
		return nil, errors.Wrapf(err, "error while handling a trial closed event: %s", requestID)
	}
	s.record(operations)

	_, isCustom := s.method.(*customSearch)
	// For non-custom-search methods, you can assume that trials will be created immediately.
	if s.state.TrialsRequested == len(s.state.TrialsClosed) && !isCustom {
		shutdown := Shutdown{
			Cancel:  len(s.state.Cancels) >= s.state.TrialsRequested,
			Failure: len(s.state.Failures) >= s.state.TrialsRequested,
		}
		s.record([]Operation{shutdown})
		operations = append(operations, shutdown)
	}

	return operations, nil
}

// TrialIsClosed returns true if the close has been recorded with a TrialClosed call.
func (s *Searcher) TrialIsClosed(requestID model.RequestID) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.state.TrialsClosed[requestID]
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

// GetCustomSearcherEventQueue returns the searcher's custom searcher event queue. It returns an
// error if the search method is not a custom searcher.
func (s *Searcher) GetCustomSearcherEventQueue() (*SearcherEventQueue, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if sMethod, ok := s.method.(*customSearch); ok {
		return sMethod.getSearcherEventQueue(), nil
	}
	return nil, unsupportedMethodError(s.method, "GetCustomSearcherEventQueue")
}

// SetCustomSearcherProgress sets the custom searcher progress.
func (s *Searcher) SetCustomSearcherProgress(progress float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if sMethod, ok := s.method.(*customSearch); ok {
		sMethod.setCustomSearcherProgress(progress)
		return nil
	}
	return unsupportedMethodError(s.method, "SetCustomSearcherProgress")
}

// Record records operations that were requested by the searcher for a specific trial.
func (s *Searcher) Record(ops []Operation) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.record(ops)
}

func (s *Searcher) record(ops []Operation) {
	for _, op := range ops {
		switch op.(type) {
		case Create:
			s.state.TrialsRequested++
		case Shutdown:
			s.state.Shutdown = true
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
