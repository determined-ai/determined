package searcher

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

	inFlightWorkloads map[WorkloadOperation]bool

	// completedWorkloads contains a bool for each WorkloadOperation which has been completed. The
	// value indicates whether EventLog.WorkloadCompleted has returned true for that
	// WorkloadOperation, which must only happen once for each distinct operation. The operation is
	// logged the first time it is seen, regardless.
	completedWorkloads map[WorkloadOperation]bool

	// completedCheckpointMsgs are WorkloadCompleted messages for CheckpointModel operations which
	// the searcher might request later.
	completedCheckpointMsgs map[WorkloadOperation]CompletedMessage

	// Searcher state.
	Shutdown bool

	// Trial state and metrics.
	TrialsRequested int
	TrialsClosed    int
	TrialIDs        map[RequestID]int
	RequestIDs      map[int]RequestID

	// Step state and metrics.
	TotalWorkloadsCompleted int
	TotalStepsStarted       int
	TotalStepsCompleted     int
}

// NewEventLog initializes an empty event log.
func NewEventLog() *EventLog {
	return &EventLog{
		inFlightWorkloads:       map[WorkloadOperation]bool{},
		completedWorkloads:      map[WorkloadOperation]bool{},
		completedCheckpointMsgs: map[WorkloadOperation]CompletedMessage{},
		Shutdown:                false,
		TrialsRequested:         0,
		TrialsClosed:            0,
		TrialIDs:                map[RequestID]int{},
		RequestIDs:              map[int]RequestID{},
		TotalStepsStarted:       0,
		TotalStepsCompleted:     0,
	}
}

// OperationsCreated records that the provided operations have been created by the searcher.
func (el *EventLog) OperationsCreated(operations ...Operation) {
	for _, operation := range operations {
		switch operation := operation.(type) {
		case Create:
			el.TrialsRequested++
		case WorkloadOperation:
			el.inFlightWorkloads[operation] = true
			switch operation.Kind {
			case RunStep:
				el.TotalStepsStarted++
			}
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

// WorkloadCompleted records that the workload has been completed. The return value indicates
// whether or not the CompletedMessage should be passed to the SearchMethod. Messages for
// unrequested workloads and useless duplicate messages will result in a return value of false.
func (el *EventLog) WorkloadCompleted(message CompletedMessage) bool {
	op := WorkloadOperation{
		Kind:      message.Workload.Kind,
		RequestID: el.RequestIDs[message.Workload.TrialID],
		StepID:    message.Workload.StepID,
	}

	// We log events the first time we see them, even if we are going to ignore them, because
	// otherwise during replay we can end up in a state where the trial disagrees with the
	// searcher_events that are replayed.
	if _, ok := el.completedWorkloads[op]; !ok {
		el.uncommitted = append(el.uncommitted, message)
		el.completedWorkloads[op] = false
	}

	// Check if we did not initiate this workload.
	if _, ok := el.inFlightWorkloads[op]; !ok {
		// In the case of a checkpoint which wasn't requested by the search method, we cache the
		// message to be replayed in case the search method requests this checkpoint later.
		if op.Kind == CheckpointModel {
			el.completedCheckpointMsgs[op] = message
		}
		return false
	}
	delete(el.inFlightWorkloads, op)

	// Check if we've already returned true once for this exact workload.
	if el.completedWorkloads[op] {
		return false
	}
	el.completedWorkloads[op] = true

	el.TotalWorkloadsCompleted++
	if message.Workload.Kind == RunStep {
		el.TotalStepsCompleted++
	}
	return true
}

// TrialClosed records that a trial with the specified trial id has been closed.
func (el *EventLog) TrialClosed(requestID RequestID) {
	trialClosed := TrialClosedEvent{
		RequestID: requestID,
	}
	el.uncommitted = append(el.uncommitted, trialClosed)
	el.TrialsClosed++
}

// FilterCompletedCheckpoints is meant to be called by Searcher.filterCompletedCheckpoints(). It is
// for identifying CheckpointModel operations for which WorkloadCompleted messages have already
// been received, and passing those messages back to the caller. The returned operations will not
// include already-completed checkpoints.
func (el *EventLog) FilterCompletedCheckpoints(
	ops []Operation) ([]Operation, []CompletedMessage) {
	var filteredOps []Operation
	var replayMsgs []CompletedMessage
	for _, op := range ops {
		if workloadOp, ok := op.(WorkloadOperation); ok {
			if msg, ok := el.completedCheckpointMsgs[workloadOp]; ok {
				replayMsgs = append(replayMsgs, msg)
				delete(el.completedCheckpointMsgs, workloadOp)
				continue
			}
		}
		filteredOps = append(filteredOps, op)
	}
	return filteredOps, replayMsgs
}
