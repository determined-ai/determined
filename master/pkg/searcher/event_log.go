package searcher

import (
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/workload"
)

// Event is the type of searcher events stored by the event log.
type Event interface{}

// TrialCreatedEvent denotes that a trial has been created with the specified Create operation
// parameters.
type TrialCreatedEvent struct {
	Create  Create
	TrialID int
}

// TrialClosedEvent denotes that a trial with the specified trial id has been closed.
type TrialClosedEvent struct {
	RequestID RequestID
}

// EventLog records all actions coming to and from a searcher.
type EventLog struct {
	uncommitted []Event

	// Searcher state.
	earlyExits          map[RequestID]bool
	TotalUnitsCompleted float64
	Shutdown            bool

	// Trial state and metrics.
	TrialsRequested int
	TrialsClosed    int
	TrialIDs        map[RequestID]int
	RequestIDs      map[int]RequestID
}

// NewEventLog initializes an empty event log.
func NewEventLog(unit model.Unit) *EventLog {
	return &EventLog{
		earlyExits:          map[RequestID]bool{},
		TotalUnitsCompleted: 0,
		Shutdown:            false,
		TrialsRequested:     0,
		TrialsClosed:        0,
		TrialIDs:            map[RequestID]int{},
		RequestIDs:          map[int]RequestID{},
	}
}

// OperationsCreated records that the provided operations have been created by the searcher.
func (el *EventLog) OperationsCreated(operations ...Operation) {
	for _, operation := range operations {
		switch operation.(type) {
		case Create:
			el.TrialsRequested++
		case Shutdown:
			el.Shutdown = true
		}
	}
}

// TrialCreated records that a trial has been created with the specified request ID.
func (el *EventLog) TrialCreated(create Create, trialID int) {
	trialCreated := TrialCreatedEvent{
		Create:  create,
		TrialID: trialID,
	}
	el.uncommitted = append(el.uncommitted, trialCreated)
	el.TrialIDs[create.RequestID] = trialID
	el.RequestIDs[trialID] = create.RequestID
}

// TrialExitedEarly marks the trial with the given requestID as exited early.
func (el *EventLog) TrialExitedEarly(requestID RequestID) {
	// If we are exiting early and we haven't seen this exiting early
	// message before, return true to send the message down to the search
	// method.
	if _, ok := el.earlyExits[requestID]; !ok {
		el.earlyExits[requestID] = true
	}
}

// WorkloadCompleted records that the workload has been completed.
func (el *EventLog) WorkloadCompleted(msg workload.CompletedMessage, unitsCompleted float64) {
	el.TotalUnitsCompleted += unitsCompleted
	el.uncommitted = append(el.uncommitted, msg)
}

// TrialClosed records that a trial with the specified trial id has been closed.
func (el *EventLog) TrialClosed(requestID RequestID) {
	trialClosed := TrialClosedEvent{
		RequestID: requestID,
	}
	el.uncommitted = append(el.uncommitted, trialClosed)
	el.TrialsClosed++
}
