package model

// JobID is the unique ID of a job among all jobs.
type JobID string

// JobType is the type of a job.
type JobType string

const (
	// JobTypeNotebook is the "NOTEBOOK" job type for the enum public.job_type in Postgres.
	JobTypeNotebook = "NOTEBOOK"
	// JobTypeShell is the "SHELL" job type for the enum public.job_type in Postgres.
	JobTypeShell = "SHELL"
	// JobTypeCommand is the "COMMAND" job type for the enum public.job_type in Postgres.
	JobTypeCommand = "COMMAND"
	// JobTypeTensorboard is the "TENSORBOARD" job type for the enum.job_type in Postgres.
	JobTypeTensorboard = "TENSORBOARD"
)

// Job is the model for a job in the database.
type Job struct {
	JobID   JobID   `db:"job_id"`
	JobType JobType `db:"job_type"`
}
