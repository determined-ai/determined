package model

import (
	"github.com/google/uuid"

	"github.com/determined-ai/determined/proto/pkg/jobv1"
)

// JobID is the unique ID of a job among all jobs.
type JobID string

// String represents the job ID as a string.
func (id JobID) String() string {
	return string(id)
}

// JobType is the type of a job.
type JobType string

// NewJobID returns a random, globally unique job ID.
func NewJobID() JobID {
	return JobID(uuid.New().String())
}

const (
	// JobTypeNotebook is the "NOTEBOOK" job type for the enum public.job_type in Postgres.
	JobTypeNotebook JobType = "NOTEBOOK"
	// JobTypeShell is the "SHELL" job type for the enum public.job_type in Postgres.
	JobTypeShell JobType = "SHELL"
	// JobTypeCommand is the "COMMAND" job type for the enum public.job_type in Postgres.
	JobTypeCommand JobType = "COMMAND"
	// JobTypeTensorboard is the "TENSORBOARD" job type for the enum.job_type in Postgres.
	JobTypeTensorboard JobType = "TENSORBOARD"
	// JobTypeExperiment is the "EXPERIMENT" job type for the enum.job_type in Postgres.
	JobTypeExperiment JobType = "EXPERIMENT"
)

// Proto returns the proto representation of the job type.
func (jt JobType) Proto() jobv1.Type {
	switch jt {
	case JobTypeExperiment:
		return jobv1.Type_TYPE_EXPERIMENT
	case JobTypeCommand:
		return jobv1.Type_TYPE_COMMAND
	case JobTypeShell:
		return jobv1.Type_TYPE_SHELL
	case JobTypeNotebook:
		return jobv1.Type_TYPE_NOTEBOOK
	case JobTypeTensorboard:
		return jobv1.Type_TYPE_TENSORBOARD
	default:
		panic("unknown job type")
	}
}

// JobTypeFromProto maps a jobv1.Type to JobType.
func JobTypeFromProto(t jobv1.Type) JobType {
	switch t {
	case jobv1.Type_TYPE_EXPERIMENT:
		return JobTypeExperiment
	case jobv1.Type_TYPE_COMMAND:
		return JobTypeCommand
	case jobv1.Type_TYPE_SHELL:
		return JobTypeShell
	case jobv1.Type_TYPE_NOTEBOOK:
		return JobTypeNotebook
	case jobv1.Type_TYPE_TENSORBOARD:
		return JobTypeTensorboard
	default:
		panic("unknown job type")
	}
}

// Job is the model for a job in the database.
type Job struct {
	JobID   JobID   `db:"job_id"`
	JobType JobType `db:"job_type"`
	OwnerID *UserID `db:"owner_id"`
	QPos    float64 `db:"q_position"`
}
