package searcher

import (
	"github.com/determined-ai/determined/master/pkg/workload"
	"math"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/nprand"
)

// Searcher encompasses the state as the searcher progresses using the provided search method.
type Searcher struct {
	rand     *nprand.State
	hparams  model.Hyperparameters
	method   SearchMethod
	eventLog *EventLog
}

// NewSearcher creates a new Searcher configured with the provided searcher config.
func NewSearcher(seed uint32, method SearchMethod, hparams model.Hyperparameters) *Searcher {
	return &Searcher{
		rand:     nprand.New(seed),
		hparams:  hparams,
		method:   method,
		eventLog: NewEventLog(method.Unit()),
	}
}

func (s *Searcher) context() context {
	return context{rand: s.rand, hparams: s.hparams}
}

// InitialOperations return a set of initial operations that the searcher would like to take.
// This should be called only once after the searcher has been created.
func (s *Searcher) InitialOperations() ([]Operation, error) {
	operations, err := s.method.initialOperations(s.context())
	if err != nil {
		return nil, errors.Wrap(err, "error while fetching initial operations of search method")
	}
	s.eventLog.OperationsCreated(operations...)
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
	return operations, nil
}

// TrialExitedEarly indicates to the searcher that the trial with the given trialID exited early.
func (s *Searcher) TrialExitedEarly(trialID int) ([]Operation, error) {
	requestID, ok := s.eventLog.RequestIDs[trialID]
	if !ok {
		return nil, errors.Errorf("unexpected trial ID sent to searcher: %d", trialID)
	}

	s.eventLog.TrialExitedEarly(requestID)
	operations, err := s.method.trialExitedEarly(s.context(), requestID)
	s.eventLog.OperationsCreated(operations...)
	if err != nil {
		return nil, errors.Wrapf(err, "error relaying trial exited early to trial %d", trialID)
	}
	return operations, nil
}

// WorkloadCompleted informs the searcher that the workload is completed. This relays the message
// to the event log and records the units as complete for search method progress.
func (s *Searcher) WorkloadCompleted(msg workload.CompletedMessage, unitsCompleted float64) {
	s.eventLog.WorkloadCompleted(msg, unitsCompleted)
}

// OperationCompleted informs the searcher that the given workload initiated by the same searcher
// has completed. Returns any new operations as a result of this workload completing.
func (s *Searcher) OperationCompleted(
	trialID int, op Runnable, metrics interface{},
) ([]Operation, error) {
	requestID, ok := s.eventLog.RequestIDs[trialID]
	if !ok {
		return nil, errors.Errorf("unexpected trial ID sent to searcher: %d", trialID)
	}

	var operations []Operation
	var err error

	switch tOp := op.(type) {
	case Train:
		operations, err = s.method.trainCompleted(s.context(), requestID, tOp)
	case Checkpoint:
		operations, err = s.method.checkpointCompleted(
			s.context(), requestID, tOp, *metrics.(*workload.CheckpointMetrics))
	case Validate:
		operations, err = s.method.validationCompleted(
			s.context(), requestID, tOp, *metrics.(*workload.ValidationMetrics))
	default:
		return nil, errors.Errorf("unexpected op: %s", tOp)
	}

	if err != nil {
		return nil, errors.Wrapf(err, "error while handling a workload completed event: %s", requestID)
	}
	s.eventLog.OperationsCreated(operations...)
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
		shutdown := Shutdown{Failure: len(s.eventLog.earlyExits) >= s.eventLog.TrialsRequested}
		s.eventLog.OperationsCreated(shutdown)
		operations = append(operations, shutdown)
	}
	return operations, nil
}

// Progress returns experiment progress as a float between 0.0 and 1.0.
func (s *Searcher) Progress() float64 {
	progress := s.method.progress(s.eventLog.TotalUnitsCompleted)
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
