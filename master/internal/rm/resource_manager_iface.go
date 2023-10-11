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
	GetAllocationSummary(sproto.GetAllocationSummary) (*sproto.AllocationSummary, error)
	GetAllocationSummaries(sproto.GetAllocationSummaries) (map[model.AllocationID]sproto.AllocationSummary, error)
	SetAllocationName(sproto.SetAllocationName)
	Allocate(sproto.AllocateRequest) (*sproto.ResourcesSubscription, error)
	Release(sproto.ResourcesReleased)
	ValidateCommandResources(sproto.ValidateCommandResourcesRequest) (sproto.ValidateCommandResourcesResponse, error)
	ValidateResources(name string, slots int, command bool) error
	DeleteJob(sproto.DeleteJob) (sproto.DeleteJobResponse, error)
	NotifyContainerRunning(sproto.NotifyContainerRunning) error

	// Scheduling related stuff
	SetGroupMaxSlots(sproto.SetGroupMaxSlots)
	SetGroupWeight(sproto.SetGroupWeight) error
	SetGroupPriority(sproto.SetGroupPriority) error
	ExternalPreemptionPending(sproto.PendingPreemption) error
	IsReattachableOnlyAfterStarted() bool

	// Resource pool stuff.
	GetResourcePoolRef(name string) (*actor.Ref, error)
	GetResourcePools(*apiv1.GetResourcePoolsRequest) (*apiv1.GetResourcePoolsResponse, error)
	GetDefaultComputeResourcePool(
		sproto.GetDefaultComputeResourcePoolRequest,
	) (sproto.GetDefaultComputeResourcePoolResponse, error)
	GetDefaultAuxResourcePool(sproto.GetDefaultAuxResourcePoolRequest) (sproto.GetDefaultAuxResourcePoolResponse, error)
	ValidateResourcePool(name string) error
	ResolveResourcePool(name string, workspace, slots int) (string, error)
	ValidateResourcePoolAvailability(name string, slots int) ([]command.LaunchWarning, error)
	TaskContainerDefaults(
		resourcePoolName string,
		fallbackConfig model.TaskContainerDefaultsConfig,
	) (model.TaskContainerDefaultsConfig, error)

	// Job queue
	GetJobQ(sproto.GetJobQ) (map[model.JobID]*sproto.RMJobInfo, error)
	GetJobQueueStatsRequest(*apiv1.GetJobQueueStatsRequest) (*apiv1.GetJobQueueStatsResponse, error)
	MoveJob(sproto.MoveJob) error
	RecoverJobPosition(sproto.RecoverJobPosition)
	GetExternalJobs(sproto.GetExternalJobs) ([]*jobv1.Job, error)

	// Cluster Management APIs
	GetAgents(*apiv1.GetAgentsRequest) (*apiv1.GetAgentsResponse, error)
	GetAgent(*apiv1.GetAgentRequest) (*apiv1.GetAgentResponse, error)
	EnableAgent(*apiv1.EnableAgentRequest) (*apiv1.EnableAgentResponse, error)
	DisableAgent(*apiv1.DisableAgentRequest) (*apiv1.DisableAgentResponse, error)
	GetSlots(*apiv1.GetSlotsRequest) (*apiv1.GetSlotsResponse, error)
	GetSlot(*apiv1.GetSlotRequest) (*apiv1.GetSlotResponse, error)
	EnableSlot(*apiv1.EnableSlotRequest) (*apiv1.EnableSlotResponse, error)
	DisableSlot(*apiv1.DisableSlotRequest) (*apiv1.DisableSlotResponse, error)

	// Escape hatch, do not use.
	Ref() *actor.Ref
}
