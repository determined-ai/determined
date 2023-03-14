package sproto

import (
	"fmt"
	"strconv"
	"time"

	"golang.org/x/exp/maps"

	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/aproto"
	"github.com/determined-ai/determined/master/pkg/cproto"
	"github.com/determined-ai/determined/master/pkg/device"
	"github.com/determined-ai/determined/master/pkg/logger"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/master/pkg/tasks"
)

// Task-related cluster level messages.
type (
	// AllocateRequest notifies resource managers to assign resources to a task.
	AllocateRequest struct {
		// Identifying information.
		AllocationID      model.AllocationID
		TaskID            model.TaskID
		JobID             model.JobID
		JobSubmissionTime time.Time
		// IsUserVisible determines whether the AllocateRequest should
		// be considered in user-visible reports.
		IsUserVisible bool
		State         SchedulingState
		Name          string
		// Allocation actor
		AllocationRef *actor.Ref
		Group         *actor.Ref

		// Resource configuration.
		SlotsNeeded         int
		ResourcePool        string
		FittingRequirements FittingRequirements

		// Behavioral configuration.
		Preemptible  bool
		IdleTimeout  *IdleTimeoutConfig
		ProxyPorts   []*ProxyPortConfig
		StreamEvents *EventStreamConfig
		Restore      bool

		// Logging context of the allocation actor.
		LogContext logger.Context
	}

	// IdleTimeoutConfig configures how idle timeouts should behave.
	IdleTimeoutConfig struct {
		ServiceID       string
		UseProxyState   bool
		UseRunnerState  bool
		TimeoutDuration time.Duration
		Debug           bool
	}

	// ProxyPortConfig configures a proxy the allocation should start.
	ProxyPortConfig struct {
		ServiceID       string `json:"service_id"`
		Port            int    `json:"port"`
		ProxyTCP        bool   `json:"proxy_tcp"`
		Unauthenticated bool   `json:"unauthenticated"`
	}

	// EventStreamConfig configures an event stream.
	EventStreamConfig struct {
		To *actor.Ref
	}

	// ResourcesReleased notifies resource providers to return resources from a task.
	ResourcesReleased struct {
		AllocationRef *actor.Ref
		ResourcesID   *ResourcesID
	}
	// GetAllocationHandler returns a ref to the handler for the specified task.
	GetAllocationHandler struct{ ID model.AllocationID }
	// GetAllocationSummary returns the summary of the specified task.
	GetAllocationSummary struct{ ID model.AllocationID }
	// GetAllocationSummaries returns the summaries of all the tasks in the cluster.
	GetAllocationSummaries struct{}
	// AllocationSummary contains information about a task for external display.
	AllocationSummary struct {
		TaskID         model.TaskID       `json:"task_id"`
		AllocationID   model.AllocationID `json:"allocation_id"`
		Name           string             `json:"name"`
		RegisteredTime time.Time          `json:"registered_time"`
		ResourcePool   string             `json:"resource_pool"`
		SlotsNeeded    int                `json:"slots_needed"`
		Resources      []ResourcesSummary `json:"resources"`
		SchedulerType  string             `json:"scheduler_type"`
		Priority       *int               `json:"priority"`
		ProxyPorts     []*ProxyPortConfig `json:"proxy_ports,omitempty"`
	}
	// SetAllocationName sets the name of the task.
	SetAllocationName struct {
		Name          string
		AllocationRef *actor.Ref
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
	// AllocationSignal is an interface for signals that can be sent to an allocation.
	AllocationSignal string
	// AllocationSignalWithReason is an message for signals that can be sent to an allocation
	// along with an informational reason about why the signal was sent.
	AllocationSignalWithReason struct {
		AllocationSignal    AllocationSignal
		InformationalReason string
	}
)

const (
	// KillAllocation is the signal to kill an allocation; analogous to in SIGKILL.
	KillAllocation AllocationSignal = "kill"
	// TerminateAllocation is the signal to kill an allocation; analogous to in SIGTERM.
	TerminateAllocation AllocationSignal = "terminate"
)

// Incoming task actor messages; task actors must accept these messages.
type (
	// ChangeRP notifies the task actor that to set itself for a new resource pool.
	ChangeRP struct {
		ResourcePool string
	}
	// ResourcesAllocated notifies the task actor of assigned resources.
	ResourcesAllocated struct {
		ID                model.AllocationID
		ResourcePool      string
		Resources         ResourceList
		JobSubmissionTime time.Time
		Recovered         bool
	}
	// PendingPreemption notifies the task actor that it should release
	// resources due to a pending system-triggered preemption.
	PendingPreemption struct {
		AllocationID model.AllocationID
	}

	// NotifyContainerRunning notifies the launcher (dispatcher) resource
	// manager that the container is running.
	NotifyContainerRunning struct {
		AllocationID model.AllocationID
		Rank         int32
		NumPeers     int32
		NodeName     string
	}

	// ReleaseResources notifies the task actor to release resources.
	ReleaseResources struct {
		ResourcePool string
		// If specified as true (default false), Requestor wants to force
		// a preemption attempt instead of an immediate kill.
		ForcePreemption bool
	}
	// ResourcesRuntimeInfo is all the inforamation provided at runtime to make a task spec.
	ResourcesRuntimeInfo struct {
		Token        string
		AgentRank    int
		IsMultiAgent bool
	}
)

const (
	// ResourcesTypeEnvVar is the name of the env var indicating the resource type to a task.
	ResourcesTypeEnvVar = "DET_RESOURCES_TYPE"
	// SlurmRendezvousIfaceEnvVar is the name of the env var for indicating the net iface on which
	// to rendezvous (horovodrun will use the IPs of the nodes on this interface to launch).
	SlurmRendezvousIfaceEnvVar = "DET_SLURM_RENDEZVOUS_IFACE"
	// SlurmProxyIfaceEnvVar is the env var for overriding the net iface used to proxy between
	// the master and agents.
	SlurmProxyIfaceEnvVar = "DET_SLURM_PROXY_IFACE"
	// ResourcesTypeK8sPod indicates the resources are a handle for a k8s pod.
	ResourcesTypeK8sPod ResourcesType = "k8s-pod"
	// ResourcesTypeDockerContainer indicates the resources are a handle for a docker container.
	ResourcesTypeDockerContainer ResourcesType = "docker-container"
	// ResourcesTypeSlurmJob indicates the resources are a handle for a slurm job.
	ResourcesTypeSlurmJob ResourcesType = "slurm-job"
)

// Clone clones ResourcesAllocated. Used to not pass mutable refs to other actors.
func (ra ResourcesAllocated) Clone() ResourcesAllocated {
	return ResourcesAllocated{
		ID:                ra.ID,
		ResourcePool:      ra.ResourcePool,
		Resources:         maps.Clone(ra.Resources),
		JobSubmissionTime: ra.JobSubmissionTime,
		Recovered:         ra.Recovered,
	}
}

// ResourcesSummary provides a summary of the resources comprising what we know at the time the
// allocation is granted, but for k8s it is granted before being scheduled so it isn't really much
// and `agent_devices` are missing for k8s.
type ResourcesSummary struct {
	ResourcesID   ResourcesID                   `json:"resources_id"`
	ResourcesType ResourcesType                 `json:"resources_type"`
	AllocationID  model.AllocationID            `json:"allocation_id"`
	AgentDevices  map[aproto.ID][]device.Device `json:"agent_devices"`

	// Available if the RM can give information on the container level.
	ContainerID *cproto.ID `json:"container_id"`

	// Available if the RM knows the resource is already started / exited.
	Started *ResourcesStarted
	Exited  *ResourcesStopped
}

// Slots returns slot count for the resources.
func (s ResourcesSummary) Slots() int {
	var res int
	for _, devs := range s.AgentDevices {
		res += len(devs)
	}
	return res
}

// Resources is an interface that provides function for task actors
// to start tasks on assigned resources.
type Resources interface {
	Summary() ResourcesSummary
	Start(*actor.Context, logger.Context, tasks.TaskSpec, ResourcesRuntimeInfo) error
	Kill(*actor.Context, logger.Context)
}

// Event is the union of all event types during the parent lifecycle.
type Event struct {
	ParentID    string    `json:"parent_id"`
	ID          string    `json:"id"`
	Seq         int       `json:"seq"`
	Time        time.Time `json:"time"`
	Description string    `json:"description"`
	IsReady     bool      `json:"is_ready"`
	ContainerID string    `json:"container_id"`
	Level       *string   `json:"level"`

	ScheduledEvent *model.AllocationID `json:"scheduled_event"`
	// AssignedEvent is triggered when the parent was assigned to an agent.
	AssignedEvent *ResourcesAllocated `json:"assigned_event"`
	// ResourcesStartedEvent is triggered when the resources started on an agent.
	ResourcesStartedEvent *ResourcesStarted `json:"resources_started_event"`
	// ServiceReadyEvent is triggered when the service running in the container is ready to serve.
	ServiceReadyEvent *bool `json:"service_ready_event"`
	// TerminateRequestEvent is triggered when the scheduler has requested the container to
	// terminate.
	TerminateRequestEvent *ReleaseResources `json:"terminate_request_event"`
	// ExitedEvent is triggered when the command has terminated.
	ExitedEvent *string `json:"exited_event"`
	// LogEvent is triggered when a new log message is available.
	LogEvent *string `json:"log_event"`
}

// ToTaskLog converts an event to a task log.
func (ev *Event) ToTaskLog() model.TaskLog {
	description := ev.Description
	var message string
	switch {
	case ev.ScheduledEvent != nil:
		message = fmt.Sprintf("Scheduling %s (id: %s)", description, ev.ParentID)
	case ev.ResourcesStartedEvent != nil:
		message = fmt.Sprintf("Resources for %s have started", description)
	case ev.TerminateRequestEvent != nil:
		message = fmt.Sprintf("%s was requested to terminate", description)
	case ev.ExitedEvent != nil:
		message = fmt.Sprintf("%s was terminated: %s", description, *ev.ExitedEvent)
	case ev.LogEvent != nil:
		message = fmt.Sprintf(*ev.LogEvent)
	case ev.ServiceReadyEvent != nil:
		message = fmt.Sprintf("Service of %s is available", description)
	case ev.AssignedEvent != nil:
		if ev.AssignedEvent.Recovered {
			message = fmt.Sprintf("%s was recovered on an agent", description)
		} else {
			message = fmt.Sprintf("%s was assigned to an agent", description)
		}
	default:
		// The client could rely on logEntry IDs and since some of these events aren't actually log
		// events we'd need to notify of them about these non existing logs either by adding a new
		// attribute to our response or a sentient log entry or we could keep it simple and normalize
		// command events as log struct by setting a special message.
		message = ""
	}

	level := ptrs.Ptr(model.LogLevelInfo)
	if ev.Level != nil {
		level = ev.Level
	}
	return model.TaskLog{
		Level:       level,
		ContainerID: &ev.ContainerID,
		Timestamp:   &ev.Time,
		Log:         message,
	}
}

// ResourceList is a wrapper for a list of resources.
type ResourceList map[ResourcesID]Resources

// NewProxyPortConfig converts expconf proxy configs into internal representation.
func NewProxyPortConfig(input expconf.ProxyPortsConfig, taskID model.TaskID) []*ProxyPortConfig {
	out := []*ProxyPortConfig{}
	for _, epp := range input {
		serviceID := string(taskID)
		if !epp.DefaultServiceID() {
			serviceID = string(taskID) + ":" + strconv.Itoa(epp.ProxyPort())
		}
		out = append(out, &ProxyPortConfig{
			Port:            epp.ProxyPort(),
			ProxyTCP:        epp.ProxyTCP(),
			Unauthenticated: epp.Unauthenticated(),
			ServiceID:       serviceID,
		})
	}

	return out
}
