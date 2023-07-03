package rm

import (
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/command"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/jobv1"
)

// ResourceManager is an interface for a resource manager, which can allocate and manage resources.
type ResourceManager interface {
	// Basic functionality
	// Returns an error if the allocation ID is not able to be found.
	GetAllocationHandler(actor.Messenger, sproto.GetAllocationHandler) (*actor.Ref, error)
	GetAllocationSummary(
		actor.Messenger,
		sproto.GetAllocationSummary,
	) (*sproto.AllocationSummary, error)
	GetAllocationSummaries(
		actor.Messenger,
		sproto.GetAllocationSummaries,
	) (map[model.AllocationID]sproto.AllocationSummary, error)
	SetAllocationName(actor.Messenger, sproto.SetAllocationName)
	Allocate(actor.Messenger, sproto.AllocateRequest) error
	Release(actor.Messenger, sproto.ResourcesReleased)
	ValidateCommandResources(
		actor.Messenger,
		sproto.ValidateCommandResourcesRequest,
	) (sproto.ValidateCommandResourcesResponse, error)
	ValidateResources(
		ctx actor.Messenger, name string, slots int, command bool,
	) error
	DeleteJob(actor.Messenger, sproto.DeleteJob) (sproto.DeleteJobResponse, error)
	NotifyContainerRunning(actor.Messenger, sproto.NotifyContainerRunning) error

	// Scheduling related stuff
	SetGroupMaxSlots(actor.Messenger, sproto.SetGroupMaxSlots)
	SetGroupWeight(actor.Messenger, sproto.SetGroupWeight) error
	SetGroupPriority(actor.Messenger, sproto.SetGroupPriority) error
	ExternalPreemptionPending(actor.Messenger, sproto.PendingPreemption) error
	IsReattachEnabled(ctx actor.Messenger) bool
	IsReattachableOnlyAfterStarted(ctx actor.Messenger) bool
	IsReattachEnabledForRP(ctx actor.Messenger, rpName string) bool

	// Resource pool stuff.
	GetResourcePoolRef(ctx actor.Messenger, name string) (*actor.Ref, error)
	GetResourcePools(
		actor.Messenger,
		*apiv1.GetResourcePoolsRequest,
	) (*apiv1.GetResourcePoolsResponse, error)
	GetDefaultComputeResourcePool(
		actor.Messenger,
		sproto.GetDefaultComputeResourcePoolRequest,
	) (sproto.GetDefaultComputeResourcePoolResponse, error)
	GetDefaultAuxResourcePool(
		actor.Messenger,
		sproto.GetDefaultAuxResourcePoolRequest,
	) (sproto.GetDefaultAuxResourcePoolResponse, error)
	ValidateResourcePool(ctx actor.Messenger, name string) error
	ResolveResourcePool(
		ctx actor.Messenger,
		name string,
		workspace,
		slots int,
	) (string, error)
	ValidateResourcePoolAvailability(
		ctx actor.Messenger,
		name string,
		slots int) ([]command.LaunchWarning, error)
	TaskContainerDefaults(
		ctx actor.Messenger,
		resourcePoolName string,
		fallbackConfig model.TaskContainerDefaultsConfig,
	) (model.TaskContainerDefaultsConfig, error)

	// Agents
	GetAgents(actor.Messenger, *apiv1.GetAgentsRequest) (*apiv1.GetAgentsResponse, error)

	// Job queue
	GetJobQ(actor.Messenger, sproto.GetJobQ) (map[model.JobID]*sproto.RMJobInfo, error)
	GetJobQStats(actor.Messenger, sproto.GetJobQStats) (*jobv1.QueueStats, error)
	GetJobQueueStatsRequest(
		actor.Messenger,
		*apiv1.GetJobQueueStatsRequest,
	) (*apiv1.GetJobQueueStatsResponse, error)
	MoveJob(actor.Messenger, sproto.MoveJob) error
	RecoverJobPosition(actor.Messenger, sproto.RecoverJobPosition)

	EnableSlot(
		actor.Messenger,
		*apiv1.EnableSlotRequest,
	) (*apiv1.EnableSlotResponse, error)
	DisableSlot(
		actor.Messenger,
		*apiv1.DisableSlotRequest,
	) (*apiv1.DisableSlotResponse, error)

	// Escape hatch, do not use.
	Ref() *actor.Ref
}
