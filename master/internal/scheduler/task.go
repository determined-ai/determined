package scheduler

import (
	"github.com/determined-ai/determined/master/pkg/actor"
)

// Task-related cluster level messages.
type (
	// AddTask notifies resource providers to assign resources to a task.
	AddTask struct {
		ID                  TaskID
		Name                string
		Group               *actor.Ref
		SlotsNeeded         int
		CanTerminate        bool
		Label               string
		FittingRequirements FittingRequirements
		Handler             *actor.Ref
	}
	// RemoveTask notifies resource providers to return resources from a task.
	RemoveTask struct{ Handler *actor.Ref }
	// GetTaskSummary returns the summary of the specified task.
	GetTaskSummary struct{ ID *TaskID }
	// GetTaskSummaries returns the summaries of all the tasks in the cluster.
	GetTaskSummaries struct{}
)

// Incoming task actor messages; task actors must accept these messages.
type (
	// ResourceAssigned notifies the task actor of assigned resources.
	ResourceAssigned struct{ Assignments []Assignment }
	// ReleaseResource notifies the task actor to release resources.
	ReleaseResource struct{}
)

// TaskID is the ID of a task.
type TaskID string
