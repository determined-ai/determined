package rm

import (
	"google.golang.org/protobuf/proto"

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
	HealthCheck() []model.ResourceManagerHealth

	// Kubernetes Namespaces and Quotas.
	DefaultNamespace(string) (*string, error)
	VerifyNamespaceExists(string, string) error
	CreateNamespace(string, string, bool) error
	DeleteNamespace(string) error
	RemoveEmptyNamespace(string, string) error
	GetNamespaceResourceQuota(string, string) (*float64, error)
	SetResourceQuota(int, string, string) error
}

// ResourcePoolName holds the name of the resource pool, and describes the input/output
// of several ResourceManager methods.
type ResourcePoolName string

// String converts a ResourcePoolName to String.
func (r ResourcePoolName) String() string {
	return string(r)
}

// CopyGetAgentsResponse returns a deep copy of GetAgentsResponse struct.
func CopyGetAgentsResponse(resp *apiv1.GetAgentsResponse) (*apiv1.GetAgentsResponse, error) {
	respBytes, err := proto.Marshal(resp)
	if err != nil {
		return nil, err
	}
	var respCopy apiv1.GetAgentsResponse
	err = proto.Unmarshal(respBytes, &respCopy)
	if err != nil {
		return nil, err
	}
	return &respCopy, nil
}

// ClusterName is the name of the cluster within which we want to send a request.
type ClusterName string

func (c ClusterName) String() string {
	return string(c)
}
