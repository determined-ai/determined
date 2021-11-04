package sproto

import (
	"github.com/determined-ai/determined/master/internal/job"
	"github.com/determined-ai/determined/master/pkg/model"
)

// TODO here or in resourcemanager/job.go

// JobSummary contains information about a job for external display.
type JobSummary struct { // FIXME same as job.RMJobInfo
	JobID          model.JobID
	State          job.SchedulingState
	RequestedSlots int
	AllocatedSlots int
}
