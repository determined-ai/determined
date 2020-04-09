package searcher

import (
	"math"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/nprand"
)

// Searcher encompasses the state as the searcher progresses using the provided search method.
type Searcher struct {
	pendingCheckpoints map[WorkloadOperation]Create
	pendingTrials      map[RequestID][]Operation
	checkpoints        map[RequestID]int
	samples            map[RequestID]hparamSample
	steps              map[RequestID]int
	rand               *nprand.State
	hparams            model.Hyperparameters
	eventLog           *EventLog
	method             SearchMethod
}

// NewSearcher creates a new Searcher configured with the provided searcher config.
func NewSearcher(seed uint32, method SearchMethod, hparams model.Hyperparameters) *Searcher {
	return &Searcher{
		pendingCheckpoints: make(map[WorkloadOperation]Create),
		pendingTrials:      make(map[RequestID][]Operation),
		checkpoints:        make(map[RequestID]int),
		samples:            make(map[RequestID]hparamSample),
		steps:              make(map[RequestID]int),
		rand:               nprand.New(seed),
		hparams:            hparams,
		eventLog:           NewEventLog(),
		method:             method,
	}
}

func (s *Searcher) context() *context {
	return &context{searcher: s}
}

// InitialOperations return a set of initial operations that the searcher would like to take.
// This should be called only once after the searcher has been created.
func (s *Searcher) InitialOperations() ([]Operation, error) {
	ctx := s.context()
	s.method.initialOperations(ctx)
	s.eventLog.OperationsCreated(ctx.pendingOperations()...)
	return ctx.pendingOperations(), nil
}

// TrialCreated informs the searcher that a trial has been created as a result of a Create
// operation.
func (s *Searcher) TrialCreated(create Create, trialID int) ([]Operation, error) {
	s.eventLog.TrialCreated(create, trialID)
	return nil, nil
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

	ctx := s.context()
	switch message.Workload.Kind {
	case RunStep:
		s.method.trainCompleted(ctx, requestID, message.Workload)
	case ComputeValidationMetrics:
		metrics := *message.ValidationMetrics
		err := s.method.validationCompleted(ctx, requestID, message.Workload, metrics)
		if err != nil {
			return nil, errors.Wrapf(err, "error handling workload completed event: %s", requestID)
		}
	case CheckpointModel:
		checkpoint := WorkloadOperation{
			RequestID: requestID,
			Kind:      message.Workload.Kind,
			StepID:    message.Workload.StepID,
		}
		if create, ok := s.pendingCheckpoints[checkpoint]; ok {
			ctx.ops = append(ctx.ops, create)
			delete(s.pendingCheckpoints, checkpoint)
			ctx.ops = append(ctx.ops, s.pendingTrials[create.RequestID]...)
			delete(s.pendingTrials, create.RequestID)
		}
	default:
		return nil, errors.Errorf("unexpected workload: %s", message.Workload.Kind)
	}
	s.eventLog.OperationsCreated(ctx.pendingOperations()...)
	return ctx.pendingOperations(), nil
}

// TrialClosed informs the searcher that the trial has been closed as a result of a Close operation.
func (s *Searcher) TrialClosed(requestID RequestID) ([]Operation, error) {
	s.eventLog.TrialClosed(requestID)
	if s.eventLog.TrialsRequested == s.eventLog.TrialsClosed {
		shutdown := Shutdown{}
		s.eventLog.OperationsCreated(shutdown)
		return []Operation{shutdown}, nil
	}
	return nil, nil
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
