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
		inFlightWorkloads:   map[WorkloadOperation]bool{},
		completedWorkloads:  map[WorkloadOperation]bool{},
		Shutdown:            false,
		TrialsRequested:     0,
		TrialsClosed:        0,
		TrialIDs:            map[RequestID]int{},
		RequestIDs:          map[int]RequestID{},
		TotalStepsStarted:   0,
		TotalStepsCompleted: 0,
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
