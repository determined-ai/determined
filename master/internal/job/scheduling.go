package job

import "github.com/determined-ai/determined/proto/pkg/jobv1"

// SchedulingState denotes the scheduling state of a job and in order of its progression value.
type SchedulingState uint8

const (
	// SchedulingStateQueued denotes a queued job waiting to be scheduled.
	SchedulingStateQueued SchedulingState = 0
	// SchedulingStateScheduledBackfilled denotes a job that is scheduled for execution as a backfill.
	SchedulingStateScheduledBackfilled SchedulingState = 1
	// SchedulingStateScheduled denotes a job that is scheduled for execution.
	SchedulingStateScheduled SchedulingState = 2
)

// Proto returns proto representation of SchedulingState.
func (s SchedulingState) Proto() jobv1.State {
	switch s {
	case SchedulingStateQueued:
		return jobv1.State_STATE_QUEUED
	case SchedulingStateScheduledBackfilled:
		return jobv1.State_STATE_SCHEDULED_BACKFILLED
	case SchedulingStateScheduled:
		return jobv1.State_STATE_SCHEDULED
	default:
		return jobv1.State_STATE_UNSPECIFIED
	}
}

func SchedulingStateFromProto(state jobv1.State) SchedulingState {
	switch state {
	case jobv1.State_STATE_QUEUED:
		return SchedulingStateQueued
	case jobv1.State_STATE_SCHEDULED_BACKFILLED:
		return SchedulingStateScheduledBackfilled
	case jobv1.State_STATE_SCHEDULED:
		return SchedulingStateScheduled
	default:
		panic("unexpected state")
	}
}

// ScheduledStates provides a list of ScheduledStates that are considered scheduled.
var ScheduledStates = map[SchedulingState]bool{
	SchedulingStateScheduled:           true,
	SchedulingStateScheduledBackfilled: true,
}
