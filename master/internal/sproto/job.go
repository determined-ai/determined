package sproto

import "github.com/determined-ai/determined/master/pkg/model"

// JobSummary contains information about a task for external display.
type JobSummary struct {
	// model.Job
	JobID    model.JobID
	JobType  model.JobType
	EntityID string `json:"entity_id"`
}
