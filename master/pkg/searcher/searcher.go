package searcher

import (
	"encoding/json"
	"math"

	"github.com/determined-ai/determined/master/pkg/workload"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/nprand"
)

type (
	// SearcherState encapsulates all persisted searcher state.
	SearcherState struct {
		TrialOperations OperationList                          `json:"trial_operations"`
		TrialsRequested int                                    `json:"trials_requested"`
		TrialsClosed    map[model.RequestID]bool               `json:"trials_closed"`
		TrialIDs        map[model.RequestID]int                `json:"trial_ids"`
		RequestIDs      map[int]model.RequestID                `json:"request_ids"`
		Failures        map[model.RequestID]bool               `json:"failures"`
		TrialProgress   map[model.RequestID]model.PartialUnits `json:"trial_progress"`
		Shutdown        bool                                   `json:"shutdown"`

		Rand *nprand.State `json:"rand"`

		SearchMethodState json.RawMessage `json:"search_method_state"`
	}

	// Searcher encompasses the state as the searcher progresses using the provided search method.
	Searcher struct {
		hparams model.Hyperparameters
		method  SearchMethod
		SearcherState
	}
)

// NewSearcher creates a new Searcher configured with the provided searcher config.
func NewSearcher(seed uint32, method SearchMethod, hparams model.Hyperparameters) *Searcher {
	return &Searcher{
		hparams: hparams,
		method:  method,
		SearcherState: SearcherState{
			Rand:          nprand.New(seed),
			TrialsClosed:  map[model.RequestID]bool{},
			TrialIDs:      map[model.RequestID]int{},
			RequestIDs:    map[int]model.RequestID{},
			Failures:      map[model.RequestID]bool{},
			TrialProgress: map[model.RequestID]model.PartialUnits{},
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
func (s *Searcher) TrialCreated(create Create, trialID int) ([]Operation, error) {
	s.TrialIDs[create.RequestID] = trialID
	s.RequestIDs[trialID] = create.RequestID
	s.TrialProgress[create.RequestID] = 0
	operations, err := s.method.trialCreated(s.context(), create.RequestID)
	if err != nil {
		return nil, errors.Wrapf(err,
			"error while handling a trial created event: %s", create.RequestID)
	}
	s.Record(operations)
	return operations, nil
}

// TrialExitedEarly indicates to the searcher that the trial with the given trialID exited early.
func (s *Searcher) TrialExitedEarly(
	trialID int, exitedReason workload.ExitedReason,
) ([]Operation, error) {
	requestID, ok := s.RequestIDs[trialID]
	if !ok {
		return nil, errors.Errorf("unexpected trial ID sent to searcher: %d", trialID)
	}

	switch exitedReason {
	case workload.InvalidHP:
		delete(s.TrialProgress, requestID)
	case workload.Errored:
		// Only workload.Errored is considered a failure (since failures cause an experiment
		// to be in the failed state).
		s.Failures[requestID] = true
	}
	operations, err := s.method.trialExitedEarly(s.context(), requestID, exitedReason)
	if err != nil {
		return nil, errors.Wrapf(err, "error relaying trial exited early to trial %d", trialID)
	}
	s.Record(operations)
	return operations, nil
}

// SetTrialProgress informs the searcher of the progress of a given trial.
func (s *Searcher) SetTrialProgress(requestID model.RequestID, progress model.PartialUnits) {
	s.TrialProgress[requestID] = progress
}

// ValidationCompleted informs the searcher that a validation for the trial was completed.
func (s *Searcher) ValidationCompleted(
	trialID int, metric float64,
) ([]Operation, error) {
	requestID, ok := s.RequestIDs[trialID]
	if !ok {
		return nil, errors.Errorf("unexpected trial ID sent to searcher: %d", trialID)
	}

	operations, err := s.method.validationCompleted(s.context(), requestID, metric)
	if err != nil {
		return nil, errors.Wrapf(err, "error while handling a workload completed event: %s", requestID)
	}
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
	if s.TrialsRequested == len(s.TrialsClosed) {
		shutdown := Shutdown{Failure: len(s.Failures) >= s.TrialsRequested}
		operations = append(operations, shutdown)
	}
	s.Record(operations)
	return operations, nil
}

// Progress returns experiment progress as a float between 0.0 and 1.0.
func (s *Searcher) Progress() float64 {
	progress := s.method.progress(s.TrialProgress)
	if math.IsNaN(progress) || math.IsInf(progress, 0) {
		return 0.0
	}
	return progress
}

// TrialID finds the trial ID for the provided request ID. The first return value is the trial ID
// if the trial has been created; otherwise, it is 0. The second is whether or not the trial has
// been created.
func (s *Searcher) TrialID(id model.RequestID) (int, bool) {
	trialID, ok := s.TrialIDs[id]
	return trialID, ok
}

// RequestID finds the request ID for the provided trial ID. The first return value is the request
// ID if the trial has been created; otherwise, it is undefined. The second is whether or not the
// trial has been created.
func (s *Searcher) RequestID(id int) (model.RequestID, bool) {
	requestID, ok := s.RequestIDs[id]
	return requestID, ok
}

// Record records operations that were requested by the searcher for a specific trial.
func (s *Searcher) Record(ops []Operation) {
	s.TrialOperations = append(s.TrialOperations, ops...)
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
