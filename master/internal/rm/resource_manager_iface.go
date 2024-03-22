package rm

import (
	"fmt"

	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/command"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/agentv1"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/jobv1"
	"github.com/google/uuid"
)

func ScaleUpAgents(baseAgent *agentv1.Agent, baseSlot *agentv1.Slot) []*agentv1.Agent {
	slotsPerNode := 512
	nodes := 2000 // *2 somehow

	newSlots := make(map[string]*agentv1.Slot, slotsPerNode)
	for i := 0; i < slotsPerNode; i++ {
		randStrId := uuid.New().String()
		newSlots[randStrId] = baseSlot
	}
	baseAgent.Slots = newSlots

	newAgents := make([]*agentv1.Agent, 0, nodes)
	for i := 0; i < nodes; i++ {
		newAgents = append(newAgents, baseAgent)
	}
	fmt.Println("Total Agents: ", len(newAgents))
	fmt.Println("Total Slots: ", len(newAgents)*slotsPerNode)
	return newAgents
}

// ResourceManager is an interface for a resource manager, which can allocate and manage resources.
type ResourceManager interface {
	// Basic functionality
	GetAllocationSummaries() (map[model.AllocationID]sproto.AllocationSummary, error)
	Allocate(sproto.AllocateRequest) (*sproto.ResourcesSubscription, error)
	Release(sproto.ResourcesReleased)
	ValidateResources(sproto.ValidateResourcesRequest) ([]command.LaunchWarning, error)
	DeleteJob(sproto.DeleteJob) (sproto.DeleteJobResponse, error)
	NotifyContainerRunning(sproto.NotifyContainerRunning) error

	// Scheduling related stuff
	SetGroupMaxSlots(sproto.SetGroupMaxSlots)
	SetGroupWeight(sproto.SetGroupWeight) error
	SetGroupPriority(sproto.SetGroupPriority) error
	ExternalPreemptionPending(sproto.PendingPreemption) error
	IsReattachableOnlyAfterStarted() bool

	// Resource pool stuff.
	GetResourcePools() (*apiv1.GetResourcePoolsResponse, error)
	GetDefaultComputeResourcePool() (ResourcePoolName, error)
	GetDefaultAuxResourcePool() (ResourcePoolName, error)
	ValidateResourcePool(ResourcePoolName) error
	ResolveResourcePool(name ResourcePoolName, workspace, slots int) (ResourcePoolName, error)
	TaskContainerDefaults(
		ResourcePoolName, model.TaskContainerDefaultsConfig,
	) (model.TaskContainerDefaultsConfig, error)

	// Job queue
	GetJobQ(ResourcePoolName) (map[model.JobID]*sproto.RMJobInfo, error)
	GetJobQueueStatsRequest(*apiv1.GetJobQueueStatsRequest) (*apiv1.GetJobQueueStatsResponse, error)
	MoveJob(sproto.MoveJob) error
	RecoverJobPosition(sproto.RecoverJobPosition)
	GetExternalJobs(ResourcePoolName) ([]*jobv1.Job, error)

	// Cluster Management APIs
	GetAgents() (*apiv1.GetAgentsResponse, error)
	GetAgent(*apiv1.GetAgentRequest) (*apiv1.GetAgentResponse, error)
	EnableAgent(*apiv1.EnableAgentRequest) (*apiv1.EnableAgentResponse, error)
	DisableAgent(*apiv1.DisableAgentRequest) (*apiv1.DisableAgentResponse, error)
	GetSlots(*apiv1.GetSlotsRequest) (*apiv1.GetSlotsResponse, error)
	GetSlot(*apiv1.GetSlotRequest) (*apiv1.GetSlotResponse, error)
	EnableSlot(*apiv1.EnableSlotRequest) (*apiv1.EnableSlotResponse, error)
	DisableSlot(*apiv1.DisableSlotRequest) (*apiv1.DisableSlotResponse, error)
}

// ResourcePoolName holds the name of the resource pool, and describes the input/output
// of several ResourceManager methods.
type ResourcePoolName string

// String converts a ResourcePoolName to String.
func (r ResourcePoolName) String() string {
	return string(r)
}
