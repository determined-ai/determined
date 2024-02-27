package rm

import (
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/command"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/jobv1"
)

// ResourceManager is an interface for a resource manager, which can allocate and manage resources.
type ResourceManager interface {
	// Basic functionality
	GetAllocationSummaries(sproto.GetAllocationSummaries) (map[model.AllocationID]sproto.AllocationSummary, error)
	Allocate(sproto.AllocateRequest) (*sproto.ResourcesSubscription, error)
	Release(sproto.ResourcesReleased)
	ValidateResources(sproto.ValidateResourcesRequest) (sproto.ValidateResourcesResponse, []command.LaunchWarning, error)
	DeleteJob(sproto.DeleteJob) (sproto.DeleteJobResponse, error)
	NotifyContainerRunning(sproto.NotifyContainerRunning) error

	// Scheduling related stuff
	SetGroupMaxSlots(sproto.SetGroupMaxSlots)
	SetGroupWeight(rmName string, req sproto.SetGroupWeight) error
	SetGroupPriority(rmName string, req sproto.SetGroupPriority) error
	ExternalPreemptionPending(sproto.PendingPreemption) error
	IsReattachableOnlyAfterStarted() bool

	// Resource pool stuff.
	GetResourcePools(*apiv1.GetResourcePoolsRequest) (*apiv1.GetResourcePoolsResponse, error)
	GetDefaultComputeResourcePool(
		sproto.GetDefaultComputeResourcePoolRequest,
	) (sproto.GetDefaultComputeResourcePoolResponse, error)
	GetDefaultAuxResourcePool(sproto.GetDefaultAuxResourcePoolRequest) (sproto.GetDefaultAuxResourcePoolResponse, error)
	ValidateResourcePool(name string) error
	ResolveResourcePool(name string, workspace, slots int) (string, error)
	TaskContainerDefaults(
		resourcePoolName string,
		fallbackConfig model.TaskContainerDefaultsConfig,
	) (model.TaskContainerDefaultsConfig, error)

	// Job queue
	GetJobQ(rmName string, rpName string) (map[model.JobID]*sproto.RMJobInfo, error)
	GetJobQueueStatsRequest(*apiv1.GetJobQueueStatsRequest) (*apiv1.GetJobQueueStatsResponse, error)
	MoveJob(rmName string, req sproto.MoveJob) error
	RecoverJobPosition(rmName string, req sproto.RecoverJobPosition)
	GetExternalJobs(rmName string, rpName string) ([]*jobv1.Job, error)

	// Cluster Management APIs
	GetAgents(*apiv1.GetAgentsRequest) (*apiv1.GetAgentsResponse, error)
	GetAgent(*apiv1.GetAgentRequest) (*apiv1.GetAgentResponse, error)
	EnableAgent(*apiv1.EnableAgentRequest) (*apiv1.EnableAgentResponse, error)
	DisableAgent(*apiv1.DisableAgentRequest) (*apiv1.DisableAgentResponse, error)
	GetSlots(*apiv1.GetSlotsRequest) (*apiv1.GetSlotsResponse, error)
	GetSlot(*apiv1.GetSlotRequest) (*apiv1.GetSlotResponse, error)
	EnableSlot(*apiv1.EnableSlotRequest) (*apiv1.EnableSlotResponse, error)
	DisableSlot(*apiv1.DisableSlotRequest) (*apiv1.DisableSlotResponse, error)
}
