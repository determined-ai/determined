package model

// JobType is the type of a job. All user facing things that run on the cluster
// are a "job".
type JobType string

const (
	// JobTypeTrial is the "TRIAL" job type for the enum public.job_type in Postgres.
	JobTypeTrial = "TRIAL"
	// JobTypeNotebook is the "NOTEBOOK" job type for the enum public.job_type in Postgres.
	JobTypeNotebook = "NOTEBOOK"
	// JobTypeShell is the "SHELL" job type for the enum public.job_type in Postgres.
	JobTypeShell = "SHELL"
	// JobTypeCommand is the "COMMAND" job type for the enum public.job_type in Postgres.
	JobTypeCommand = "COMMAND"
	// JobTypeCheckpointGC is the "CHECKPOINT_GC" job type for the enum public.job_type in Postgres.
	JobTypeCheckpointGC = "CHECKPOINT_GC"
)
