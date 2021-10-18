package sproto

import (
	"time"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/jobv1"
)

// TODO here or in model/job.go

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

// JobSummary contains information about a job for external display.
type JobSummary struct {
	//Job      model.Job
	JobID          model.JobID
	JobType        model.JobType
	EntityID       string `json:"entity_id"`
	State          SchedulingState
	RequestedSlots int
	AllocatedSlots int
}

// JobTaskInfo is a message that carries information needed to augment
// jobs.
type JobTaskInfo struct {
	Username       string
	Name           string
	SubmissionTime time.Time
}

// ScheduledStates provides a list of ScheduledStates that are considered scheduled.
var ScheduledStates = map[SchedulingState]bool{
	SchedulingStateScheduled:           true,
	SchedulingStateScheduledBackfilled: true,
}
