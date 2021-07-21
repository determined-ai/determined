package sproto

import (
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/tasks"
)

// Task-related cluster level messages.
type (
	// AllocateRequest notifies resource managers to assign resources to a task.
	AllocateRequest struct {
		AllocationID        model.AllocationID
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
	// GetTaskHandler returns a ref to the handler for the specified task.
	GetTaskHandler struct{ ID model.AllocationID }
	// GetTaskSummary returns the summary of the specified task.
	GetTaskSummary struct{ ID *model.AllocationID }
	// GetTaskSummaries returns the summaries of all the tasks in the cluster.
	GetTaskSummaries struct{}
	// SetTaskName sets the name of the task.
	SetTaskName struct {
		Name        string
		TaskHandler *actor.Ref
	}

	// ValidateCommandResourcesRequest is a message asking resource manager whether the given
	// resource pool can (or, rather, if it's not impossible to) fulfill the command request
	// for the given amount of slots.
	ValidateCommandResourcesRequest struct {
		ResourcePool string
		Slots        int
	}

	// ValidateCommandResourcesResponse is the response to ValidateCommandResourcesRequest.
	ValidateCommandResourcesResponse struct {
		// Fulfillable values:
		// - false: impossible to fulfill
		// - true: ok or unknown
		Fulfillable bool
	}
)

// ValidateRPResources checks if the resource pool can fulfill resource request for single-node
// notebook/command/shell etc. Returns &true if yes, &false if not, and nil if unknown.
func ValidateRPResources(system *actor.System, resourcePoolName string, slots int) (bool, error) {
	resp := system.Ask(
		GetCurrentRM(system), ValidateCommandResourcesRequest{
			ResourcePool: resourcePoolName,
			Slots:        slots,
		})
	if resp.Error() != nil {
		return false, resp.Error()
	}
	return resp.Get().(ValidateCommandResourcesResponse).Fulfillable, nil
}

// Incoming task actor messages; task actors must accept these messages.
type (
	// ResourcesAllocated notifies the task actor of assigned resources.
	ResourcesAllocated struct {
		ID           model.AllocationID
		ResourcePool string
		Reservations []Reservation
	}
	// ReleaseResources notifies the task actor to release resources.
	ReleaseResources struct {
		ResourcePool string
	}
)

// Reservation is an interface that provides function for task actors
// to start tasks on assigned resources.
type Reservation interface {
	Summary() ContainerSummary
	Start(ctx *actor.Context, spec tasks.TaskSpec, rank int)
	Kill(ctx *actor.Context)
}
