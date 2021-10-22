package sproto

import (
	"regexp"
	"time"

	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/tasks"
)

// Task-related cluster level messages.
type (
	// AllocateRequest notifies resource managers to assign resources to a task.
	AllocateRequest struct {
		// Identifying information.
		AllocationID model.AllocationID
		TaskID       model.TaskID
		Name         string
		TaskActor    *actor.Ref
		Group        *actor.Ref

		// Resource configuration.
		SlotsNeeded         int
		Label               string
		ResourcePool        string
		FittingRequirements FittingRequirements

		// Behavioral configuration.
		Preemptible   bool
		DoRendezvous  bool
		IdleTimeout   *IdleTimeoutConfig
		ProxyPort     *PortProxyConfig
		StreamEvents  *EventStreamConfig
		LogBasedReady *LogBasedReadinessConfig
	}

	// IdleTimeoutConfig configures how idle timeouts should behave.
	IdleTimeoutConfig struct {
		ServiceID       string
		UseProxyState   bool
		UseRunnerState  bool
		TimeoutDuration time.Duration
	}

	// PortProxyConfig configures a proxy the allocation should start.
	PortProxyConfig struct {
		ServiceID string
		Port      int
		ProxyTCP  bool
	}

	// EventStreamConfig configures an event stream.
	EventStreamConfig struct {
		To *actor.Ref
	}

	// LogBasedReadinessConfig configures using logs as a ready check.
	LogBasedReadinessConfig struct {
		Pattern *regexp.Regexp
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
	// ReservationRuntimeInfo is all the inforamation provided at runtime to make a task spec.
	ReservationRuntimeInfo struct {
		Token        string
		AgentRank    int
		IsMultiAgent bool
	}
)

// Reservation is an interface that provides function for task actors
// to start tasks on assigned resources.
type Reservation interface {
	Summary() ContainerSummary
	Start(ctx *actor.Context, spec tasks.TaskSpec, rri ReservationRuntimeInfo)
	Kill(ctx *actor.Context)
}

// Event is the union of all event types during the parent lifecycle.
type Event struct {
	ParentID    string    `json:"parent_id"`
	ID          string    `json:"id"`
	Seq         int       `json:"seq"`
	Time        time.Time `json:"time"`
	Description string    `json:"description"`
	IsReady     bool      `json:"is_ready"`
	State       string    `json:"state"`

	ScheduledEvent *model.AllocationID `json:"scheduled_event"`
	// AssignedEvent is triggered when the parent was assigned to an agent.
	AssignedEvent *ResourcesAllocated `json:"assigned_event"`
	// ContainerStartedEvent is triggered when the container started on an agent.
	ContainerStartedEvent *TaskContainerStarted `json:"container_started_event"`
	// ServiceReadyEvent is triggered when the service running in the container is ready to serve.
	// TODO: Move to ServiceReadyEvent type to a specialized event with readiness checks.
	ServiceReadyEvent *ContainerLog `json:"service_ready_event"`
	// TerminateRequestEvent is triggered when the scheduler has requested the container to
	// terminate.
	TerminateRequestEvent *ReleaseResources `json:"terminate_request_event"`
	// ExitedEvent is triggered when the command has terminated.
	ExitedEvent *string `json:"exited_event"`
	// LogEvent is triggered when a new log message is available.
	LogEvent *string `json:"log_event"`
}
