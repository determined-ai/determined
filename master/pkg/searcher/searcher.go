package searcher

import (
	"math"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/nprand"
)

// Searcher encompasses the state as the searcher progresses using the provided search method.
type Searcher struct {
	rand     *nprand.State
	hparams  model.Hyperparameters
	eventLog *EventLog
	method   SearchMethod
}

// NewSearcher creates a new Searcher configured with the provided searcher config.
func NewSearcher(seed uint32, method SearchMethod, hparams model.Hyperparameters) *Searcher {
	rand := nprand.New(seed)
	return &Searcher{rand: rand, hparams: hparams, eventLog: NewEventLog(), method: method}
}

func (s *Searcher) context() context {
	return context{rand: s.rand, hparams: s.hparams}
}

// filterCompletedCheckpoints identifies operations which request checkpoints that have already
// been completed and replays the corresponding WorkloadCompleted messages to the SearchMethod.
// This situation arises when a SearchMethod requests a checkpoint after the scheduler has forced
// that trial to checkpoint and be descheduled. The Searcher intercepts and saves that checkpoint
// completed message, and this function is where that message gets replayed to the SearchMethod.
// The returned operations will not include already-completed checkpoints.
func (s *Searcher) filterCompletedCheckpoints(ops []Operation) ([]Operation, error) {
	var filteredOps []Operation
	for len(ops) > 0 {
		newFilteredOps, replayMsgs := s.eventLog.FilterCompletedCheckpoints(ops)
		filteredOps = append(filteredOps, newFilteredOps...)
		// Replay WorkloadCompleted messges and get additional operations.
		ops = nil
		for _, msg := range replayMsgs {
			// The EventLog should internally recognize the message as replayed and not duplicate
			// the message in the searcher_events, but it should not tell us to ignore the message.
			if !s.eventLog.WorkloadCompleted(msg) {
				return nil, errors.Errorf("event log ignored a cached WorkloadCompleted message")
			}
			requestID := s.eventLog.RequestIDs[msg.Workload.TrialID]
			moreOps, err := s.method.checkpointCompleted(
				s.context(), requestID, msg.Workload, *msg.CheckpointMetrics)
			if err != nil {
				return filteredOps, errors.Wrapf(err,
					"error while replaying WorkloadCompleted message for workload: %v",
					msg.Workload)
			}
			ops = append(ops, moreOps...)
		}
		s.eventLog.OperationsCreated(ops...)
	}
	return filteredOps, nil
}

// InitialOperations return a set of initial operations that the searcher would like to take.
// This should be called only once after the searcher has been created.
func (s *Searcher) InitialOperations() ([]Operation, error) {
	operations, err := s.method.initialOperations(s.context())
	if err != nil {
		return nil, errors.Wrap(err, "error while fetching initial operations of search method")
	}
	s.eventLog.OperationsCreated(operations...)
	operations, err = s.filterCompletedCheckpoints(operations)
	if err != nil {
		return nil, errors.Wrap(err, "error while filtering initial operations of search method")
	}
	return operations, nil
}

// TrialCreated informs the searcher that a trial has been created as a result of a Create
// operation.
func (s *Searcher) TrialCreated(create Create, trialID int) ([]Operation, error) {
	s.eventLog.TrialCreated(create, trialID)
	operations, err := s.method.trialCreated(s.context(), create.RequestID)
	if err != nil {
		return nil, errors.Wrapf(err,
			"error while handling a trial created event: %s", create.RequestID)
	}
	s.eventLog.OperationsCreated(operations...)
	operations, err = s.filterCompletedCheckpoints(operations)
	if err != nil {
		return nil, errors.Wrap(err, "error while filtering operations after trial created event")
	}
	return operations, nil
}

// WorkloadCompleted informs the searcher that the given workload initiated by the same searcher
// has completed. Returns any new operations as a result of this workload completing.
func (s *Searcher) WorkloadCompleted(message CompletedMessage) ([]Operation, error) {
	requestID, ok := s.eventLog.RequestIDs[message.Workload.TrialID]
	if !ok {
		return nil, errors.Errorf("unexpected trial ID sent to searcher: %d",
			message.Workload.TrialID)
	}

	// The event log will tell us if this workload should not be sent to the search method (either
	// because it was not initiated by the search method, or because it is a duplicate).
	if !s.eventLog.WorkloadCompleted(message) {
		return nil, nil
	}

	var operations []Operation
	var err error

	switch message.Workload.Kind {
	case RunStep:
		operations, err = s.method.trainCompleted(
			s.context(), requestID, message.Workload)
	case CheckpointModel:
		operations, err = s.method.checkpointCompleted(
			s.context(), requestID, message.Workload, *message.CheckpointMetrics)
	case ComputeValidationMetrics:
		operations, err = s.method.validationCompleted(
			s.context(), requestID, message.Workload, *message.ValidationMetrics)
	default:
		return nil, errors.Errorf("unexpected workload: %s", message.Workload.Kind)
	}
	if err != nil {
		return nil, errors.Wrapf(err, "error while handling a workload completed event: %s", requestID)
	}
	s.eventLog.OperationsCreated(operations...)
	operations, err = s.filterCompletedCheckpoints(operations)
	if err != nil {
		return nil, errors.Wrap(
			err, "error while filtering operations after workload complete event")
	}
	return operations, nil
}

// TrialClosed informs the searcher that the trial has been closed as a result of a Close operation.
func (s *Searcher) TrialClosed(requestID RequestID) ([]Operation, error) {
	s.eventLog.TrialClosed(requestID)
	operations, err := s.method.trialClosed(s.context(), requestID)
	if err != nil {
		return nil, errors.Wrapf(err, "error while handling a trial closed event: %s", requestID)
	}
	s.eventLog.OperationsCreated(operations...)
	if s.eventLog.TrialsRequested == s.eventLog.TrialsClosed {
		shutdown := NewShutdown()
		s.eventLog.OperationsCreated(shutdown)
		operations = append(operations, shutdown)
	}
	operations, err = s.filterCompletedCheckpoints(operations)
	if err != nil {
		return nil, errors.Wrap(err, "error while filtering operations after trial closed event")
	}
	return operations, nil
}

// Progress returns experiment progress as a float between 0.0 and 1.0.
func (s *Searcher) Progress() float64 {
	progress := s.method.progress(s.eventLog.TotalWorkloadsCompleted)
	if math.IsNaN(progress) || math.IsInf(progress, 0) {
		return 0.0
	}
	return progress
}

// TrialID finds the trial ID for the provided request ID. The first return value is the trial ID
// if the trial has been created; otherwise, it is 0. The second is whether or not the trial has
// been created.
func (s *Searcher) TrialID(id RequestID) (int, bool) {
	trialID, ok := s.eventLog.TrialIDs[id]
	return trialID, ok
}

// RequestID finds the request ID for the provided trial ID. The first return value is the request
// ID if the trial has been created; otherwise, it is undefined. The second is whether or not the
// trial has been created.
func (s *Searcher) RequestID(id int) (RequestID, bool) {
	requestID, ok := s.eventLog.RequestIDs[id]
	return requestID, ok
}

// UncommittedEvents returns the searcher events that have occurred since the last call to
// UncommittedEvents.
func (s *Searcher) UncommittedEvents() []Event {
	defer func() { s.eventLog.uncommitted = nil }()
	return s.eventLog.uncommitted
}
