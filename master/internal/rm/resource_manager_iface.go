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
	GetAllocationSummaries() (map[model.AllocationID]sproto.AllocationSummary, error)
	Allocate(rmName string, req sproto.AllocateRequest) (*sproto.ResourcesSubscription, error)
	Release(rmName string, req sproto.ResourcesReleased)
	ValidateResources(rmName string, req sproto.ValidateResourcesRequest) ([]command.LaunchWarning, error)
	DeleteJob(sproto.DeleteJob) (sproto.DeleteJobResponse, error)   // only used in dispatcherrm
	NotifyContainerRunning(req sproto.NotifyContainerRunning) error // only used in dispatcherrm

	// Scheduling related stuff
	SetGroupMaxSlots(rmName string, req sproto.SetGroupMaxSlots)
	SetGroupWeight(rmName string, req sproto.SetGroupWeight) error
	SetGroupPriority(rmName string, req sproto.SetGroupPriority) error
	ExternalPreemptionPending(allocID model.AllocationID) error // only used in dispatcherrm
	IsReattachableOnlyAfterStarted(rmName string) bool

	// Resource pool stuff
	GetResourcePools() (*apiv1.GetResourcePoolsResponse, error)
	GetDefaultComputeResourcePool(rmName string) (sproto.GetDefaultComputeResourcePoolResponse, error)
	GetDefaultAuxResourcePool(rmName string) (sproto.GetDefaultAuxResourcePoolResponse, error)
	ValidateResourcePool(rmName string, rpName string) error
	ResolveResourcePool(rmName string, req sproto.ResolveResourcesRequest) (rm string, rp string, err error)
	TaskContainerDefaults(rmName string, rpName string, fallbackConfig model.TaskContainerDefaultsConfig) (
		model.TaskContainerDefaultsConfig, error)

	// Job queue
	GetJobQ(rmName string, rpName string) (map[model.JobID]*sproto.RMJobInfo, error)
	GetJobQueueStatsRequest(rmName string, req *apiv1.GetJobQueueStatsRequest) (*apiv1.GetJobQueueStatsResponse, error)
	MoveJob(rmName string, req sproto.MoveJob) error
	RecoverJobPosition(rmName string, req sproto.RecoverJobPosition)
	GetExternalJobs(rmName string, rpName string) ([]*jobv1.Job, error)

	// Cluster Management APIs
	GetAgents() (*apiv1.GetAgentsResponse, error)
	GetAgent(rmName string, req *apiv1.GetAgentRequest) (*apiv1.GetAgentResponse, error)
	EnableAgent(rmName string, req *apiv1.EnableAgentRequest) (*apiv1.EnableAgentResponse, error)
	DisableAgent(rmName string, req *apiv1.DisableAgentRequest) (*apiv1.DisableAgentResponse, error)
	GetSlots(rmName string, req *apiv1.GetSlotsRequest) (*apiv1.GetSlotsResponse, error)
	GetSlot(rmName string, req *apiv1.GetSlotRequest) (*apiv1.GetSlotResponse, error)
	EnableSlot(rmName string, req *apiv1.EnableSlotRequest) (*apiv1.EnableSlotResponse, error)
	DisableSlot(rmName string, req *apiv1.DisableSlotRequest) (*apiv1.DisableSlotResponse, error)
}
