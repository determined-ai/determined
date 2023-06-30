package sproto

import (
	"strconv"
	"time"

	"golang.org/x/exp/maps"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/aproto"
	"github.com/determined-ai/determined/master/pkg/cproto"
	"github.com/determined-ai/determined/master/pkg/device"
	"github.com/determined-ai/determined/master/pkg/logger"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/master/pkg/tasks"
	"github.com/determined-ai/determined/proto/pkg/taskv1"
)

// Task-related cluster level messages.
type (
	// AllocateRequest notifies resource managers to assign resources to a task.
	AllocateRequest struct {
		// Identifying information.
		AllocationID      model.AllocationID
		TaskID            model.TaskID
		JobID             model.JobID
		RequestTime       time.Time
		JobSubmissionTime time.Time
		// IsUserVisible determines whether the AllocateRequest should
		// be considered in user-visible reports.
		IsUserVisible bool
		State         SchedulingState
		Name          string
		// Allocation actor
		Group *actor.Ref

		// Resource configuration.
		SlotsNeeded         int
		ResourcePool        string
		FittingRequirements FittingRequirements

		// Behavioral configuration.
		Preemptible bool
		IdleTimeout *IdleTimeoutConfig
		ProxyPorts  []*ProxyPortConfig
		Restore     bool

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

	// ResourcesReleased notifies resource providers to return resources from a task.
	ResourcesReleased struct {
		AllocationID model.AllocationID
		ResourcesID  *ResourcesID
	}
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
		Name         string
		AllocationID model.AllocationID
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
	// ResourceDisabled informs the allocation its resources have been disabled. It just causes
	// the allocation to kill its reasons. This probably shouldn't be an event, but the RM
	// should take the kill action since it knows it wants to (but that is difficult, because
	// it surfaces to the allocation as just a 137 exit). TODO(!!!): Let's separate all these
	// into some files, `events.go`.
	ResourceDisabled struct{ InformationReason string }
)

// AllocationEvent describes a change in status or state of an allocation or its resources.
type AllocationEvent interface{ AllocationEvent() }

// AllocationEvent implements AllocationEvent.
func (ResourcesAllocated) AllocationEvent() {}

// AllocationEvent implements AllocationEvent.
func (InvalidResourcesRequestError) AllocationEvent() {}

// AllocationEvent implements AllocationEvent.
func (ReleaseResources) AllocationEvent() {}

// AllocationEvent implements AllocationEvent.
func (ResourcesStateChanged) AllocationEvent() {}

// AllocationEvent implements AllocationEvent.
func (ResourcesFailure) AllocationEvent() {}

// AllocationEvent implements AllocationEvent.
func (ContainerLog) AllocationEvent() {}

// AllocationEvent implements AllocationEvent.
func (ResourceDisabled) AllocationEvent() {}

// AllocationUnsubscribeFn closes a subscription.
type AllocationUnsubscribeFn func()

// AllocationSubscription is a subscription for streaming AllocationEvent's. It must be closed when
// you are finished consuming events. Blocking on C forever can cause the publisher to backup
// and adversely affect the system.
type AllocationSubscription struct {
	// C is never closed, because only the consumer knows, by aggregating events, when events stop.
	C     <-chan AllocationEvent
	unsub AllocationUnsubscribeFn
}

// NewAllocationSubscription create a new subcription.
func NewAllocationSubscription(
	updates <-chan AllocationEvent,
	cl AllocationUnsubscribeFn,
) *AllocationSubscription {
	return &AllocationSubscription{
		C:     updates,
		unsub: cl,
	}
}

// Close unsubscribes us from further updates.
func (a *AllocationSubscription) Close() {
	a.unsub()
}

// Proto returns the proto representation of ProxyPortConfig.
func (p *ProxyPortConfig) Proto() *taskv1.ProxyPortConfig {
	if p == nil {
		return nil
	}

	return &taskv1.ProxyPortConfig{
		ServiceId:       p.ServiceID,
		Port:            int32(p.Port),
		ProxyTcp:        p.ProxyTCP,
		Unauthenticated: p.Unauthenticated,
	}
}

// Proto returns the proto representation of AllocationSummary.
func (a *AllocationSummary) Proto() *taskv1.AllocationSummary {
	if a == nil {
		return nil
	}

	pbResources := []*taskv1.ResourcesSummary{}
	for _, resource := range a.Resources {
		pbResourcesSummary := resource.Proto()
		pbResources = append(pbResources, pbResourcesSummary)
	}

	pbAllocationSummary := taskv1.AllocationSummary{
		TaskId:         string(a.TaskID),
		AllocationId:   string(a.AllocationID),
		Name:           a.Name,
		RegisteredTime: timestamppb.New(a.RegisteredTime),
		ResourcePool:   a.ResourcePool,
		SlotsNeeded:    int32((a.SlotsNeeded)),
		Resources:      pbResources,
		SchedulerType:  a.SchedulerType,
	}

	if a.Priority != nil {
		pbPriority := int32(*a.Priority)
		pbAllocationSummary.Priority = &pbPriority
	}

	if a.ProxyPorts != nil {
		pbProxyPorts := []*taskv1.ProxyPortConfig{}
		for _, proxyPortConfig := range a.ProxyPorts {
			pbProxyPorts = append(pbProxyPorts, proxyPortConfig.Proto())
		}

		pbAllocationSummary.ProxyPorts = pbProxyPorts
	}

	return &pbAllocationSummary
}

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
func (ra ResourcesAllocated) Clone() *ResourcesAllocated {
	return &ResourcesAllocated{
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

// Proto returns the proto representation of ResourcesSummary.
func (s *ResourcesSummary) Proto() *taskv1.ResourcesSummary {
	if s == nil {
		return nil
	}

	pbAgentDevices := make(map[string]*taskv1.ResourcesSummary_Devices)

	for agentID, devices := range s.AgentDevices {
		pbDevices := taskv1.ResourcesSummary_Devices{}

		for _, device := range devices {
			pbDevice := device.Proto()
			pbDevices.Devices = append(pbDevices.Devices, pbDevice)
		}
		pbAgentDevices[string(agentID)] = &pbDevices
	}

	pbResourcesSummary := taskv1.ResourcesSummary{
		ResourcesId:   string(s.ResourcesID),
		ResourcesType: string(s.ResourcesType),
		AllocationId:  string(s.AllocationID),
		AgentDevices:  pbAgentDevices,
		Started:       s.Started.Proto(),
		Exited:        s.Exited.Proto(),
	}

	if s.ContainerID != nil {
		pbContainerID := string(*s.ContainerID)
		pbResourcesSummary.ContainerId = &pbContainerID
	}

	return &pbResourcesSummary
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
	// TODO(!!!): Remove `*actor.System` from this interface.
	Start(*actor.System, logger.Context, tasks.TaskSpec, ResourcesRuntimeInfo) error
	Kill(*actor.System, logger.Context)
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
