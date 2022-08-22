package rm

import (
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/jobv1"
)

// ResourceManager is an interface for a resource manager, which can allocate and manage resources.
type ResourceManager interface {
	// Basic functionality
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
	DeleteJob(actor.Messenger, sproto.DeleteJob) (sproto.DeleteJobResponse, error)

	// Scheduling related stuff
	SetGroupMaxSlots(actor.Messenger, sproto.SetGroupMaxSlots)
	SetGroupWeight(actor.Messenger, sproto.SetGroupWeight) error
	SetGroupPriority(actor.Messenger, sproto.SetGroupPriority) error
	ExternalPreemptionPending(actor.Messenger, sproto.PendingPreemption) error

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
	ResolveResourcePool(ctx actor.Messenger, name string, slots int, command bool) (string, error)

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

	// Escape hatch, do not use.
	Ref() *actor.Ref
}
