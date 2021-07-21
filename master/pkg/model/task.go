package model

import "time"

// AllocationID is the ID of an allocation of a task.
type AllocationID string

// NewAllocationID returns a new unique task id.
func NewAllocationID(name string) AllocationID {
	return AllocationID(name)
}

// Allocation is the model for an allocation in the database.
type Allocation struct {
	ID           int          `db:"id"`
	TaskID       TaskID       `db:"task_id"`
	AllocationID AllocationID `db:"allocation_id"`
	ResourcePool string       `db:"resource_pool"`
	StartTime    time.Time    `db:"start_time"`
	EndTime      *time.Time   `db:"end_time"`
}

// TaskID is the unique ID of a task among all tasks.
type TaskID string

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
	ID        int        `db:"id" `
	TaskID    TaskID     `db:"task_id"`
	StartTime time.Time  `db:"start_time"`
	EndTime   *time.Time `db:"end_time"`
}
