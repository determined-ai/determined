package command

import (
	"time"

	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/tasks"
)

// CommandSnapshot is a db representation of a generic command.
type CommandSnapshot struct {
	bun.BaseModel `bun:"table:command_state"`

	TaskID         model.TaskID `bun:"task_id"`
	RegisteredTime time.Time    `bun:"registered_time"`
	// taskType can get from task.
	// jobType can get from task -> job_id -> job_type
	// jobid can get from task -> job_id
	AllocationID model.AllocationID `bun:"allocation_id"`
	// last state from allocation
	// exit status from allocation

	// GenericCommandSpec
	GenericCommandSpec tasks.GenericCommandSpec `bun:"generic_command_spec"`

	// Relations
	Task       model.Task       `bun:"rel:belongs-to,join:task_id=task_id"`
	Allocation model.Allocation `bun:"rel:belongs-to,join:allocation_id=allocation_id"`
}
