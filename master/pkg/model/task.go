package model

import (
	"time"

	"github.com/google/uuid"

	"github.com/determined-ai/determined/proto/pkg/taskv1"
)

// JobID is the unique ID of a job among all jobs.
type JobID string

// NewJobID returns a random, globally unique job ID.
func NewJobID() JobID {
	return JobID(uuid.New().String())
}

// JobType is the type of a job.
type JobType string

const (
	// JobTypeExperiment is the "EXPERIMENT" job type for the enum public.job_type in Postgres.
	JobTypeExperiment = "EXPERIMENT"
	// JobTypeNotebook is the "NOTEBOOK" job type for the enum public.job_type in Postgres.
	JobTypeNotebook = "NOTEBOOK"
	// JobTypeShell is the "SHELL" job type for the enum public.job_type in Postgres.
	JobTypeShell = "SHELL"
	// JobTypeCommand is the "COMMAND" job type for the enum public.job_type in Postgres.
	JobTypeCommand = "COMMAND"
	// JobTypeTensorboard is the "TENSORBOARD" task type for the enum.task_type in Postgres.
	JobTypeTensorboard = "TENSORBOARD"
)

// TaskID is the unique ID of a task among all tasks.
type TaskID string

// NewTaskID returns a random, globally unique task ID.
func NewTaskID() TaskID {
	return TaskID(uuid.New().String())
}

// TaskType is the type of a task.
type TaskType string

const (
	// TaskTypeTrial is the "TRIAL" job type for the enum public.job_type in Postgres.
	TaskTypeTrial = "TRIAL"
	// TaskTypeNotebook is the "NOTEBOOK" job type for the enum public.job_type in Postgres.
	TaskTypeNotebook = "NOTEBOOK"
	// TaskTypeShell is the "SHELL" job type for the enum public.job_type in Postgres.
	TaskTypeShell = "SHELL"
	// TaskTypeCommand is the "COMMAND" job type for the enum public.job_type in Postgres.
	TaskTypeCommand = "COMMAND"
	// TaskTypeTensorboard is the "TENSORBOARD" task type for the enum.task_type in Postgres.
	TaskTypeTensorboard = "TENSORBOARD"
	// TaskTypeCheckpointGC is the "CHECKPOINT_GC" job type for the enum public.job_type in Postgres.
	TaskTypeCheckpointGC = "CHECKPOINT_GC"
)

// Task is the model for a task in the database.
type Task struct {
	TaskID    TaskID     `db:"task_id"`
	TaskType  TaskType   `db:"task_type"`
	StartTime time.Time  `db:"start_time"`
	EndTime   *time.Time `db:"end_time"`
}

// AllocationID is the ID of an allocation of a task. It is usually of the form
// TaskID.allocation_number, maybe with some other metadata if different types of
// allocations run.
type AllocationID string

// NewAllocationID returns a new unique task id.
func NewAllocationID(name string) AllocationID {
	return AllocationID(name)
}

func (a AllocationID) String() string {
	return string(a)
}

// Allocation is the model for an allocation in the database.
type Allocation struct {
	AllocationID AllocationID `db:"allocation_id"`
	TaskID       TaskID       `db:"task_id"`
	Slots        int          `db:"slots"`
	AgentLabel   string       `db:"agent_label"`
	ResourcePool string       `db:"resource_pool"`
	StartTime    time.Time    `db:"start_time"`
	EndTime      *time.Time   `db:"end_time"`
}

// AllocationState represents the current state of the task. Value indicates a partial ordering.
type AllocationState int

const (
	// AllocationStatePending state denotes that the command is awaiting allocation.
	AllocationStatePending AllocationState = 0
	// AllocationStateAssigned state denotes that the command has been assigned to an agent but has
	// not started yet.
	AllocationStateAssigned AllocationState = 1
	// AllocationStatePulling state denotes that the command's base image is being pulled from the
	// Docker registry.
	AllocationStatePulling AllocationState = 2
	// AllocationStateStarting state denotes that the image has been pulled and the task is being
	// started, but the task is not ready yet.
	AllocationStateStarting AllocationState = 3
	// AllocationStateRunning state denotes that the service in the command is running.
	AllocationStateRunning AllocationState = 4
	// AllocationStateTerminating state denotes that the command is terminating.
	AllocationStateTerminating AllocationState = 5
	// AllocationStateTerminated state denotes that the command has exited or has been aborted
	AllocationStateTerminated AllocationState = 6
)

// MostProgressedAllocationState returns the further progressed state. E.G. a call
// with PENDING, PULLING and STARTING returns PULLING.
func MostProgressedAllocationState(states ...AllocationState) AllocationState {
	if len(states) == 0 {
		return AllocationStatePending
	}

	max := states[0]
	for _, state := range states {
		if state > max {
			max = state
		}
	}
	return max
}

// String returns the string representation of the task state.
func (s AllocationState) String() string {
	switch s {
	case AllocationStatePending:
		return "PENDING"
	case AllocationStateAssigned:
		return "ASSIGNED"
	case AllocationStatePulling:
		return "PULLING"
	case AllocationStateStarting:
		return "STARTING"
	case AllocationStateRunning:
		return "RUNNING"
	case AllocationStateTerminating:
		return "TERMINATING"
	case AllocationStateTerminated:
		return "TERMINATED"
	default:
		return "UNSPECIFIED"
	}
}

// Proto returns the proto representation of the task state.
func (s AllocationState) Proto() taskv1.State {
	switch s {
	case AllocationStatePending:
		return taskv1.State_STATE_PENDING
	case AllocationStateAssigned:
		return taskv1.State_STATE_ASSIGNED
	case AllocationStatePulling:
		return taskv1.State_STATE_PULLING
	case AllocationStateStarting:
		return taskv1.State_STATE_STARTING
	case AllocationStateRunning:
		return taskv1.State_STATE_RUNNING
	case AllocationStateTerminating:
		return taskv1.State_STATE_TERMINATING
	case AllocationStateTerminated:
		return taskv1.State_STATE_TERMINATED
	default:
		return taskv1.State_STATE_UNSPECIFIED
	}
}
