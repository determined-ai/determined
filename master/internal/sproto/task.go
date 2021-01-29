package sproto

import (
	"github.com/google/uuid"

	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/tasks"
)

// Task-related cluster level messages.
type (
	// AllocateRequest notifies resource managers to assign resources to a task.
	AllocateRequest struct {
		ID                  TaskID
		Name                string
		Group               *actor.Ref
		SlotsNeeded         int
		NonPreemptible      bool
		Label               string
		ResourcePool        string
		FittingRequirements FittingRequirements
		TaskActor           *actor.Ref
	}
	// ResourcesReleased notifies resource providers to return resources from a task.
	ResourcesReleased struct {
		TaskActor *actor.Ref
	}
	// GetTaskSummary returns the summary of the specified task.
	GetTaskSummary struct{ ID *TaskID }
	// GetTaskSummaries returns the summaries of all the tasks in the cluster.
	GetTaskSummaries struct{}
	// SetTaskName sets the name of the task.
	SetTaskName struct {
		Name        string
		TaskHandler *actor.Ref
	}
)

// Incoming task actor messages; task actors must accept these messages.
type (
	// ResourcesAllocated notifies the task actor of assigned resources.
	ResourcesAllocated struct {
		ID           TaskID
		ResourcePool string
		Allocations  []Allocation
	}
	// ReleaseResources notifies the task actor to release resources.
	ReleaseResources struct {
		ResourcePool string
	}
)

// TaskID is the ID of a task.
type TaskID string

// NewTaskID returns a new unique task id.
func NewTaskID() TaskID {
	return TaskID(uuid.New().String())
}

// Allocation is an interface that provides function for task actors
// to start tasks on assigned resources.
type Allocation interface {
	Summary() ContainerSummary
	Start(ctx *actor.Context, spec tasks.TaskSpec)
	Kill(ctx *actor.Context)
}
