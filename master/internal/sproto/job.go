package sproto

import (
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/jobv1"
)

// TODO here or in model/job.go

type SchedulingState uint8

const (
	SchedulingStateQueued              SchedulingState = 0
	SchedulingStateScheduledBackfilled SchedulingState = 1
	SchedulingStateScheduled           SchedulingState = 2
)

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

// JobSummary contains information about a task for external display.
type JobSummary struct {
	// model.Job
	JobID    model.JobID
	JobType  model.JobType
	EntityID string `json:"entity_id"`
	State    SchedulingState
}
