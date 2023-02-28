package dispatcherrm

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	semvar "github.com/Masterminds/semver/v3"
	"github.com/ghodss/yaml"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	launcher "github.hpe.com/hpe/hpc-ard-launcher-go/launcher"
	"golang.org/x/exp/maps"
	"google.golang.org/protobuf/proto"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/rm/actorrm"
	"github.com/determined-ai/determined/master/internal/rm/rmerrors"
	"github.com/determined-ai/determined/master/internal/rm/tasklist"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/actor/actors"
	"github.com/determined-ai/determined/master/pkg/aproto"
	"github.com/determined-ai/determined/master/pkg/device"
	"github.com/determined-ai/determined/master/pkg/logger"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/tasks"
	"github.com/determined-ai/determined/proto/pkg/agentv1"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/containerv1"
	"github.com/determined-ai/determined/proto/pkg/devicev1"
	"github.com/determined-ai/determined/proto/pkg/resourcepoolv1"
)

const maxResourceDetailsSampleAgeSeconds = 60
const (
	slurmSchedulerType     = "slurm"
	pbsSchedulerType       = "pbs"
	slurmResourcesCarrier  = "com.cray.analytics.capsules.carriers.hpc.slurm.SlurmResources"
	pbsResourcesCarrier    = "com.cray.analytics.capsules.carriers.hpc.pbs.PbsResources"
	launcherMinimumVersion = "3.2.2"
	root                   = "root"
)

// schedulerTick periodically triggers the scheduler to act.
type schedulerTick struct{}

// actionCoolDown is the rate limit for queue submission.
const actionCoolDown = 500 * time.Millisecond

// maxFailedTerminationRetries is the number of times we're willing to retry a
// failed job termination.
const maxFailedTerminationRetries = 3

// waitTimeBeforeRetryingTermination is the number of seconds to wait before
// sending the next termination request to the launcher.
const waitTimeBeforeRetryingTermination = 20

// hpcResources is a data type describing the HPC resources available
// to Slurm on the Launcher node.
// Example output of the HPC resource details from the Launcher.
// ---
// partitions:
// - totalAvailableNodes: 293
// totalAllocatedNodes: 21
// partitionName: workq
// totalAvailableGpuSlots: 16
// totalNodes: 314
// totalGpuSlots: 16
// - totalAvailableNodes: 293
// ...more partitions.
type hpcResources struct {
	Partitions                  []hpcPartitionDetails `json:"partitions,flow"` //nolint:staticcheck
	Nodes                       []hpcNodeDetails      `json:"nodes,flow"`      //nolint:staticcheck
	DefaultComputePoolPartition string                `json:"defaultComputePoolPartition"`
	DefaultAuxPoolPartition     string                `json:"defaultAuxPoolPartition"`
}

// hpcPartitionDetails holds HPC Slurm partition details.
type hpcPartitionDetails struct {
	TotalAvailableNodes    int    `json:"totalAvailableNodes"`
	PartitionName          string `json:"partitionName"`
	IsDefault              bool   `json:"default"`
	TotalAllocatedNodes    int    `json:"totalAllocatedNodes"`
	TotalAvailableGpuSlots int    `json:"totalAvailableGpuSlots"`
	TotalNodes             int    `json:"totalNodes"`
	TotalGpuSlots          int    `json:"totalGpuSlots"`
	TotalAvailableCPUSlots int    `json:"totalAvailableCpuSlots"`
	TotalCPUSlots          int    `json:"totalCpuSlots"`
}

// hpcNodeDetails holds HPC Slurm node details.
type hpcNodeDetails struct {
	Partitions    []string `json:"partitions"`
	Addresses     []string `json:"addresses"`
	Draining      bool     `json:"draining"`
	Allocated     bool     `json:"allocated"`
	Name          string   `json:"name"`
	GpuCount      int      `json:"gpuCount"`
	GpuInUseCount int      `json:"gpuInUseCount"`
	CPUCount      int      `json:"cpuCount"`
	CPUInUseCount int      `json:"cpuInUseCount"`
}

// hpcResourceDetailsCache stores details of the HPC resource information cache.
type hpcResourceDetailsCache struct {
	mu         sync.RWMutex
	lastSample hpcResources
	sampleTime time.Time
	isUpdating bool
}

type (
	// hasSlurmPartitionRequest is a message querying the presence of the specified resource pool.
	hasSlurmPartitionRequest struct {
		PoolName string
	}

	// hasSlurmPartitionResponse is the response to HasResourcePoolRequest.
	hasSlurmPartitionResponse struct {
		HasResourcePool    bool
		ProvidingPartition string // Set for launcher-provided resource pools
		ValidationErrors   []error
	}
)

// DispatcherResourceManager is a resource manager for managing slurm resources.
type DispatcherResourceManager struct {
	*actorrm.ResourceManager
}

// failedJobsTerminations map keeps track of the number of job termination
// retries. The key is the dispatch ID and the value is the retry count.
// Whenever a call to "terminateDispatcherJob()" fails (e.g., the launcher
// returns an HTTP 500 error), we schedule another retry. This map is used
// to keep track of the number of retries that have been performed, so that
// once we reach the maximum number of retries, we stop retrying.
var failedJobTerminations sync.Map

// GetResourcePoolRef just returns a ref to the dispatcher RM since it doesn't have separate
// pool actors.
func (d *DispatcherResourceManager) GetResourcePoolRef(
	ctx actor.Messenger, name string,
) (*actor.Ref, error) {
	return d.Ref(), nil
}

// ResolveResourcePool returns the resolved slurm partition or an error if it doesn't exist or
// can't be resolved due to internal errors.
func (d *DispatcherResourceManager) ResolveResourcePool(
	ctx actor.Messenger, name string, slots int,
) (string, error) {
	// If the resource pool isn't set, fill in the default at creation time.
	if name == "" && slots == 0 {
		req := sproto.GetDefaultAuxResourcePoolRequest{}
		resp, err := d.GetDefaultAuxResourcePool(ctx, req)
		if err != nil {
			return "", fmt.Errorf("defaulting to aux pool: %w", err)
		}
		return resp.PoolName, nil
	}

	if name == "" && slots >= 0 {
		req := sproto.GetDefaultComputeResourcePoolRequest{}
		resp, err := d.GetDefaultComputeResourcePool(ctx, req)
		if err != nil {
			return "", fmt.Errorf("defaulting to compute pool: %w", err)
		}
		return resp.PoolName, nil
	}
	providingPartition, err := d.validateResourcePool(ctx, name)
	if err != nil {
		return "", fmt.Errorf("validating resource pool: %w", err)
	}
	if providingPartition != "" {
		return providingPartition, nil
	}
	return name, nil
}

// ValidateResourcePool validates that the given resource pool exists.
func (d *DispatcherResourceManager) ValidateResourcePool(ctx actor.Messenger, name string) error {
	_, err := d.validateResourcePool(ctx, name)
	return err
}

func (d *DispatcherResourceManager) validateResourcePool(ctx actor.Messenger,
	name string,
) (string, error) {
	var resp hasSlurmPartitionResponse
	switch err := d.Ask(ctx, hasSlurmPartitionRequest{PoolName: name}, &resp); {
	case err != nil:
		return "", fmt.Errorf("requesting resource pool: %w", err)
	case !resp.HasResourcePool && resp.ProvidingPartition != "":
		return "", fmt.Errorf(
			"resource pool %s is configured to use partition %s that does not exist "+
				"-- verify the cluster configuration", name, resp.ProvidingPartition)
	case !resp.HasResourcePool:
		return "", fmt.Errorf("resource pool not found: %s", name)
	case len(resp.ValidationErrors) > 0:
		// Return the first of any validation errors -- this will inform the user
		// at experiment creation/command run time that a configuration issue exists.
		return resp.ProvidingPartition, resp.ValidationErrors[0]
	default:
		return resp.ProvidingPartition, nil
	}
}

// IsReattachEnabled is always true for dispatcher-based job schedulers.
func (d *DispatcherResourceManager) IsReattachEnabled(ctx actor.Messenger) bool {
	return true
}

// IsReattachableOnlyAfterStarted is always false for dispatcher-based job schedulers
// as the start_time is not set on our allocations.
func (d *DispatcherResourceManager) IsReattachableOnlyAfterStarted(ctx actor.Messenger) bool {
	return false
}

// IsReattachEnabledForRP returns true for all resource pools.
func (d *DispatcherResourceManager) IsReattachEnabledForRP(
	ctx actor.Messenger, rpName string,
) bool {
	return true
}

// New returns a new dispatcher resource manager.
func New(
	system *actor.System,
	db *db.PgDB,
	echo *echo.Echo,
	config *config.ResourceConfig,
	opts *aproto.MasterSetAgentOptions,
	cert *tls.Certificate,
) *DispatcherResourceManager {
	tlsConfig, err := model.MakeTLSConfig(cert)
	if err != nil {
		panic(errors.Wrap(err, "failed to set up TLS config"))
	}

	var rm *dispatcherResourceManager
	if config.ResourceManager.DispatcherRM != nil {
		// slurm type is configured
		rm = newDispatcherResourceManager(
			config.ResourceManager.DispatcherRM, config.ResourcePools, tlsConfig, opts.LoggingOptions,
		)
	} else {
		// pbs type is configured
		rm = newDispatcherResourceManager(
			config.ResourceManager.PbsRM, config.ResourcePools, tlsConfig, opts.LoggingOptions,
		)
	}
	ref, _ := system.ActorOf(
		sproto.DispatcherRMAddr,
		rm,
	)
	system.Ask(ref, actor.Ping{}).Get()
	return &DispatcherResourceManager{ResourceManager: actorrm.Wrap(ref)}
}

// dispatcherResourceProvider manages the lifecycle of dispatcher resources.
type dispatcherResourceManager struct {
	rmConfig                                   *config.DispatcherResourceManagerConfig
	poolConfig                                 []config.ResourcePoolConfig
	apiClient                                  *launcher.APIClient
	hpcResourcesManifest                       *launcher.Manifest
	reqList                                    *tasklist.TaskList
	groups                                     map[*actor.Ref]*tasklist.Group
	dispatchIDToAllocationID                   map[string]model.AllocationID
	masterTLSConfig                            model.TLSClientConfig
	loggingConfig                              model.LoggingConfig
	jobWatcher                                 *launcherMonitor
	authToken                                  string
	resourceDetails                            hpcResourceDetailsCache
	wlmType                                    string
	launcherVersionIsOK                        bool
	poolProviderMap                            map[string][]string
	dispatchIDToHPCJobID                       map[string]string
	dispatchIDToAllocationIDMutex              sync.RWMutex
	dispatchIDToHPCJobIDMutex                  sync.RWMutex
	allocationsCanceledBeforeDispatchIDCreated sync.Map
}

func newDispatcherResourceManager(
	rmConfig *config.DispatcherResourceManagerConfig,
	poolConfig []config.ResourcePoolConfig,
	masterTLSConfig model.TLSClientConfig,
	loggingConfig model.LoggingConfig,
) *dispatcherResourceManager {
	// Set up the host address and IP address of the "launcher".
	clientConfiguration := launcher.NewConfiguration()

	// Host, port, and protocol are configured in the "resource_manager" section
	// of the "tools/devcluster.yaml" file. The host address and port refer to the
	// system where the "launcher" is running.
	clientConfiguration.Host = fmt.Sprintf("%s:%d", rmConfig.LauncherHost, rmConfig.LauncherPort)
	clientConfiguration.Scheme = rmConfig.LauncherProtocol // "http" or "https"
	if rmConfig.Security != nil {
		logrus.Debugf("Launcher communications InsecureSkipVerify: %t", rmConfig.Security.TLS.SkipVerify)
		tlsConfig := tls.Config{InsecureSkipVerify: rmConfig.Security.TLS.SkipVerify} //nolint:gosec
		transCfg := &http.Transport{
			TLSClientConfig: &tlsConfig,
		}
		clientConfiguration.HTTPClient = &http.Client{Transport: transCfg}
	}

	apiClient := launcher.NewAPIClient(clientConfiguration)

	// One time activity to create a manifest using SlurmResources carrier.
	// This manifest is used on demand to retrieve details regarding HPC resources
	// e.g., nodes, GPUs etc
	hpcResourcesManifest := createSlurmResourcesManifest()

	// Authentication token that gets passed to the "launcher" REST API.
	authToken := loadAuthToken(rmConfig)

	return &dispatcherResourceManager{
		rmConfig: rmConfig,

		apiClient:                apiClient,
		hpcResourcesManifest:     hpcResourcesManifest,
		reqList:                  tasklist.New(),
		groups:                   make(map[*actor.Ref]*tasklist.Group),
		dispatchIDToAllocationID: make(map[string]model.AllocationID),

		masterTLSConfig:      masterTLSConfig,
		loggingConfig:        loggingConfig,
		jobWatcher:           newDispatchWatcher(apiClient, authToken, rmConfig.LauncherAuthFile),
		authToken:            authToken,
		launcherVersionIsOK:  false,
		poolConfig:           poolConfig,
		poolProviderMap:      makeProvidedPoolsMap(poolConfig),
		dispatchIDToHPCJobID: make(map[string]string),
	}
}

// makeProvidedPoolsMap returns a map where the key is the providing partition
// and the values are the launcher-provided pools provided by the partition.
// This is all static configuration data, so we can make this map just once
// in the lifetime of this RM.
func makeProvidedPoolsMap(poolConfig []config.ResourcePoolConfig) map[string][]string {
	poolProviderMap := make(map[string][]string)
	for _, pool := range poolConfig {
		if isValidProvider(pool) {
			partitionName := pool.Provider.HPC.Partition
			poolProviderMap[partitionName] = append(poolProviderMap[partitionName], pool.PoolName)
		}
	}
	return poolProviderMap
}

// Return a starting context for the API client call that includes the authToken
// (may be empty if disabled).
func (m *dispatcherResourceManager) authContext(ctx *actor.Context) context.Context {
	return context.WithValue(context.Background(), launcher.ContextAccessToken, m.authToken)
}

func (m *dispatcherResourceManager) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart:
		m.debugKillAllInactiveDispatches(ctx, ctx.Self())
		go m.jobWatcher.watch(ctx)
		// SLURM Resource Manager always fulfills requests for resource pool details using the
		// value from the cache. This call will ensure there is an initial value in the cache at
		// the start of the resource manager.
		m.fetchHpcResourceDetails(ctx)
		actors.NotifyAfter(ctx, actionCoolDown, schedulerTick{})

	case
		sproto.AllocateRequest,
		StartDispatcherResources,
		KillDispatcherResources,
		DispatchStateChange,
		DispatchExited,
		sproto.SetGroupMaxSlots,
		sproto.SetAllocationName,
		sproto.PendingPreemption,
		sproto.NotifyContainerRunning,
		sproto.ResourcesReleased,
		tasklist.GroupActorStopped:
		return m.receiveRequestMsg(ctx)

	case
		sproto.GetJobQ,
		sproto.GetJobSummary,
		sproto.GetJobQStats,
		sproto.SetGroupWeight,
		sproto.SetGroupPriority,
		sproto.MoveJob,
		sproto.DeleteJob,
		*apiv1.GetJobQueueStatsRequest:
		return m.receiveJobQueueMsg(ctx)

	case sproto.GetAllocationHandler:
		handler := m.reqList.TaskHandler(msg.ID)
		if handler == nil {
			ctx.Respond(fmt.Errorf("allocation handler for allocation ID %s not found", msg.ID))
			return nil
		}
		ctx.Respond(handler)

	case sproto.GetAllocationSummary:
		if resp := m.reqList.TaskSummary(msg.ID, m.groups, slurmSchedulerType); resp != nil {
			ctx.Respond(*resp)
		}

	case sproto.GetAllocationSummaries:
		ctx.Respond(m.reqList.TaskSummaries(m.groups, slurmSchedulerType))

	case *apiv1.GetResourcePoolsRequest:
		resourcePoolSummary, err := m.summarizeResourcePool(ctx)
		if err != nil {
			ctx.Respond(err)
			return nil
		}
		ctx.Respond(&apiv1.GetResourcePoolsResponse{
			ResourcePools: resourcePoolSummary,
		})

	case sproto.GetDefaultComputeResourcePoolRequest:
		_, _ = m.fetchHpcResourceDetailsCached(ctx)
		// Don't bother to check for errors, a response is required (may have no name)
		m.resourceDetails.mu.RLock()
		defer m.resourceDetails.mu.RUnlock()
		ctx.Respond(sproto.GetDefaultComputeResourcePoolResponse{
			PoolName: m.resourceDetails.lastSample.DefaultComputePoolPartition,
		})

	case sproto.GetDefaultAuxResourcePoolRequest:
		_, _ = m.fetchHpcResourceDetailsCached(ctx)
		// Don't bother to check for errors, a response is required (may have no name)
		m.resourceDetails.mu.RLock()
		defer m.resourceDetails.mu.RUnlock()
		ctx.Respond(sproto.GetDefaultAuxResourcePoolResponse{
			PoolName: m.resourceDetails.lastSample.DefaultAuxPoolPartition,
		})

	case hasSlurmPartitionRequest:
		// This is a query to see if the specified resource pool exists
		hpcDetails, err := m.fetchHpcResourceDetailsCached(ctx)
		var response hasSlurmPartitionResponse
		if err != nil {
			response = hasSlurmPartitionResponse{HasResourcePool: false}
		} else {
			response = m.getPartitionValidationResponse(hpcDetails, msg.PoolName)
		}
		ctx.Respond(response)

	case sproto.ValidateCommandResourcesRequest:
		// TODO(HAL-2862): Use inferred value here if possible.
		// fulfillable := m.config.MaxSlotsPerContainer >= msg.Slots
		ctx.Respond(sproto.ValidateCommandResourcesResponse{Fulfillable: true})

	case schedulerTick:
		m.schedulePendingTasks(ctx)
		actors.NotifyAfter(ctx, actionCoolDown, schedulerTick{})

	case *apiv1.GetAgentsRequest:
		ctx.Respond(m.generateGetAgentsResponse(ctx))

	case taskContainerDefaults:
		ctx.Respond(m.getTaskContainerDefaults(msg))

	default:
		ctx.Log().Errorf("unexpected message %T", msg)
		return actor.ErrUnexpectedMessage(ctx)
	}

	return nil
}

func (m *dispatcherResourceManager) getTaskContainerDefaults(
	msg taskContainerDefaults,
) model.TaskContainerDefaultsConfig {
	result := msg.fallbackDefault
	if tcd := m.rmConfig.ResolveTaskContainerDefaults(msg.resourcePool); tcd != nil {
		result = *tcd
	}
	// Iterate through configured pools looking for a TaskContainerDefaults setting.
	for _, pool := range m.poolConfig {
		if msg.resourcePool == pool.PoolName {
			if pool.TaskContainerDefaults == nil {
				break
			}
			result = *pool.TaskContainerDefaults
		}
	}
	return result
}

// getPartitionValidationResponse computes a response to a resource pool
// validation request. The target may be either a HPC native partition/queue,
// or a launcher-provided pool. In the latter case we verify that the providing
// partition exists on the cluster.
func (m *dispatcherResourceManager) getPartitionValidationResponse(
	hpcDetails hpcResources, targetPartitionName string,
) hasSlurmPartitionResponse {
	result := false
	providingPartition := ""
	var validationErrors []error
	result = partitionExists(targetPartitionName, hpcDetails.Partitions)
	if !result {
		for _, pool := range m.poolConfig {
			if pool.PoolName == targetPartitionName && isValidProvider(pool) {
				basePartition := pool.Provider.HPC.Partition
				providingPartition = basePartition
				if partitionExists(basePartition, hpcDetails.Partitions) {
					result = true
					validationErrors = performValidation(pool)
				}
				break // on the first name match
			}
		}
	}
	return hasSlurmPartitionResponse{
		HasResourcePool:    result,
		ProvidingPartition: providingPartition,
		ValidationErrors:   validationErrors,
	}
}

func performValidation(pool config.ResourcePoolConfig) []error {
	var validationErrors []error
	if pool.TaskContainerDefaults != nil {
		e := tasks.ValidatePbs(pool.TaskContainerDefaults.Pbs.SbatchArgs())
		validationErrors = append(validationErrors, e...)
		e = tasks.ValidateSlurm(pool.TaskContainerDefaults.Slurm.SbatchArgs())
		validationErrors = append(validationErrors, e...)
	}
	return validationErrors
}

// partitionExists return true if the specified partition exists on the HPC cluster.
func partitionExists(targetPartition string, knowPartitions []hpcPartitionDetails) bool {
	for _, p := range knowPartitions {
		if p.PartitionName == targetPartition {
			return true
		}
	}
	return false
}

// generateGetAgentsResponse returns a suitable response to the GetAgentsRequest request.
func (m *dispatcherResourceManager) generateGetAgentsResponse(
	ctx *actor.Context,
) *apiv1.GetAgentsResponse {
	response := apiv1.GetAgentsResponse{
		Agents: []*agentv1.Agent{},
	}
	_, _ = m.fetchHpcResourceDetailsCached(ctx)
	m.resourceDetails.mu.RLock()
	defer m.resourceDetails.mu.RUnlock()
	for _, node := range m.resourceDetails.lastSample.Nodes {
		agent := agentv1.Agent{
			Id:             node.Name,
			RegisteredTime: nil,
			Slots:          map[string]*agentv1.Slot{},
			ResourcePools:  node.Partitions,
			Addresses:      node.Addresses,
			Enabled:        true,
			Draining:       node.Draining,
		}
		m.updateAgentWithAnyProvidedResourcePools(&agent)
		response.Agents = append(response.Agents, &agent)
		if node.GpuCount == 0 {
			// Adds a slot ID (e.g., 0, 1, 2, ..., N) to the agent for every
			// CPU being used on the node. This is needed so that the
			// "Resource Pools" page on the Determined AI User Interface
			// correctly shows the "N/M CPU Slots Allocated".
			for i := 0; i < node.CPUCount; i++ {
				addSlotToAgent(
					&agent, devicev1.Type_TYPE_CPU, node, i, i < node.CPUInUseCount)
			}
		} else {
			for i := 0; i < node.GpuCount; i++ {
				slotType := computeSlotType(node, m)
				addSlotToAgent(
					&agent, slotType, node, i, i < node.GpuInUseCount) // [1:N] CUDA slots
			}
		}
	}
	return &response
}

func (m *dispatcherResourceManager) updateAgentWithAnyProvidedResourcePools(
	agent *agentv1.Agent,
) {
	for _, poolName := range agent.ResourcePools {
		agent.ResourcePools = append(agent.ResourcePools, m.poolProviderMap[poolName]...)
	}
}

// computeSlotType computes an agent GPU slot type from the configuration data available.
// For nodes that are members of multiple partitions, take the first configured slot type found,
// falling back to CUDA if nothing found.
func computeSlotType(node hpcNodeDetails, m *dispatcherResourceManager) devicev1.Type {
	for _, partition := range node.Partitions {
		slotType := m.rmConfig.ResolveSlotType(partition)
		if slotType != nil {
			return slotType.Proto()
		}
	}
	return devicev1.Type_TYPE_CUDA
}

// addSlotToAgent adds to the specifies agent a slot populated with a device of the specified type.
func addSlotToAgent(
	agent *agentv1.Agent,
	deviceType devicev1.Type,
	node hpcNodeDetails,
	slotID int,
	slotInUse bool,
) {
	device := devicev1.Device{
		Id:    0,
		Brand: "",
		Uuid:  "",
		Type:  deviceType,
	}
	slotRef := fmt.Sprintf("/agents/%s/slots/%d", node.Name, slotID)
	slot := agentv1.Slot{
		Id:       fmt.Sprintf("%d", slotID),
		Device:   &device,
		Enabled:  true,
		Draining: false,
	}
	if slotInUse {
		// Claiming a container causes the DAI GUI dashboard to consider the
		// slot to be not available; other implications TBD.
		slot.Container = &containerv1.Container{}
		slot.Container.State = containerv1.State_STATE_RUNNING
	}
	agent.Slots[slotRef] = &slot
}

func (m *dispatcherResourceManager) receiveRequestMsg(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case sproto.AllocateRequest:
		m.addTask(ctx, msg)

	case StartDispatcherResources:
		// Start each launcher job in a goroutine to prevent incoming messages
		// from backing up, due to the main thread being busy handling one
		// message at a time. Adaptive searches may create many launcher jobs
		// for a single experiment, so we must allow the main thread to continue
		// handling incoming messages while the previous messages are still
		// being processed. The UI will become unresponsive if the messages
		// start backing up.
		go m.startLauncherJob(ctx, msg)

	case sproto.PendingPreemption:
		ctx.Log().Infof("PendingPreemption of %s.  Terminating.", msg.AllocationID)
		allocReq, ok := m.reqList.TaskByID(msg.AllocationID)
		if ok {
			ctx.Tell(allocReq.AllocationRef, sproto.ReleaseResources{ForcePreemption: true})
		} else {
			ctx.Log().Errorf("unable to find Allocation actor for AllocationID %s",
				msg.AllocationID)
		}

	case sproto.NotifyContainerRunning:
		dispatches, err := db.ListDispatchesByAllocationID(context.TODO(), msg.AllocationID)
		if err != nil {
			ctx.Log().WithError(err).Errorf(
				"Failed to retrieve the DispatchIDs associated with AllocationID %s",
				msg.AllocationID)
			return nil
		}

		foundMonitoredDispatch := false
		for _, dispatch := range dispatches {
			dispatchID := dispatch.DispatchID
			if m.jobWatcher.isJobBeingMonitored(dispatchID) {
				foundMonitoredDispatch = true
				m.jobWatcher.notifyContainerRunning(ctx, dispatchID, msg.Rank, msg.NumPeers, msg.NodeName)
			}
		}
		if !foundMonitoredDispatch {
			ctx.Log().WithField("allocation-id", msg.AllocationID).Warnf(
				"NotifyContainerRunning did not find an active, monitored dispatch")
		}

	case KillDispatcherResources:
		ctx.Log().Debugf("Received request to terminate jobs associated with AllocationID %s",
			msg.AllocationID)

		// Find the Dispatch IDs associated with the allocation ID. We'll need the
		// Dispatch ID to cancel the job on the launcher side.
		dispatches, err := db.ListDispatchesByAllocationID(context.TODO(), msg.AllocationID)
		if err != nil {
			ctx.Log().WithError(err).Errorf(
				"Failed to retrieve the DispatchIDs associated with AllocationID %s",
				msg.AllocationID)

			m.allocationsCanceledBeforeDispatchIDCreated.Store(string(msg.AllocationID), "")

			return nil
		}

		ctx.Log().Debugf("Found %d jobs associated with AllocationID %s",
			len(dispatches), msg.AllocationID)

		// The job cancelation message arrived before the launcher created the
		// dispatch ID. Without the dispatch ID, we cannot ask the launcher to
		// terminate the job.
		if len(dispatches) == 0 {
			// Make a note that we tried to cancel a job whose dispatch ID was
			// not created yet. This way, when the dispatch ID gets created and
			// the workload manager returns an HPC job ID, we can go ahead
			// terminate the job.
			//
			// The value we associate with the allocation ID doesn't really
			// matter, so we use "". We could have used a list instead of a map,
			// but then we have to iterate through the list every time. A map
			// seemed more efficient.
			m.allocationsCanceledBeforeDispatchIDCreated.Store(string(msg.AllocationID), "")
			return nil
		}

		// We have a dispatch ID for this allocation ID, so remove the entry
		// for this allocation ID from the map, if one previously existed. This
		// ensures the map doesn't grow indefinitely from stale entries.
		m.allocationsCanceledBeforeDispatchIDCreated.Delete(string(msg.AllocationID))

		for _, dispatch := range dispatches {
			dispatchID := dispatch.DispatchID
			impersonatedUser := dispatch.ImpersonatedUser

			ctx.Log().WithField("allocation-id", msg.AllocationID).Infof(
				"Terminating job with DispatchID %s initiated by %s", dispatchID, impersonatedUser)

			// Terminate and cleanup, on failure leave Dispatch in DB for later retry
			if m.terminateDispatcherJob(ctx, dispatchID, impersonatedUser, false) {
				// Remove the dispatch ID from the failed job termination maps
				// in case we had previously failed to terminate the job
				// associated with the dispatch ID. This prevents the map from
				// growing indefinitely.
				failedJobTerminations.Delete(dispatchID)

				// Do not remove the dispatch environment if the job is being
				// monitored by the job watcher, as it is needed in order for
				// the launcher to report the job status. If we remove the
				// dispatch environment, then the launcher will no longer be
				// able to provide job information and will return an HTTP 404
				// status when the job watcher asks it for status. As a result,
				// the Detemined AI job status will never get updated from
				// "Running" to "Canceled", for example.  When the job watcher
				// gets a terminatal state from the launcher, it will take care
				// of removing the dispatch environment at that time.
				if m.jobWatcher.isJobBeingMonitored(dispatchID) {
					ctx.Log().WithField("allocation-id", msg.AllocationID).Debugf(
						"Not removing dispatch environment for dispatchID '%s' because job is being monitored",
						dispatchID)
				} else {
					// If we are here, then we are likely being called from
					// startup, as opposed to a user explicitly canceling
					// a job. It's OK to remove the environment in this case
					// because we aren't actively monitoring any jobs, but we need to wait
					// for the terminate request above to complete, before we can actually
					// do the delete of the environment to avoid a 500 error response.
					m.waitForDispatchTerminalState(ctx, impersonatedUser, dispatchID)
					m.removeDispatchEnvironment(ctx, impersonatedUser, dispatchID)
				}
			} else {
				// Use a variable to store the job termination function, so
				// that we can mock the function for the unit test.
				var jobTerminationFunc = func(
					ctx *actor.Context,
					allocationID model.AllocationID,
				) {
					ctx.Log().WithField("allocation-id", msg.AllocationID).
						Info("Sending KillDispatcherResources message")

					// Send the dispatcher resource manager a request to terminate the job.
					ctx.Tell(ctx.Self(), KillDispatcherResources{
						AllocationID: allocationID,
					})
				}

				// Schedule another retry, if we haven't exhausted all retries.
				go performFailedJobTerminationRetries(ctx,
					dispatchID,
					msg.AllocationID,
					waitTimeBeforeRetryingTermination,
					maxFailedTerminationRetries,
					jobTerminationFunc)
			}
		}

	case DispatchStateChange:
		log := ctx.Log().WithField("dispatch-id", msg.DispatchID)
		allocationID, ok := m.getAllocationIDFromDispatchID(msg.DispatchID)
		if !ok {
			log.Warnf("received DispatchStateChange for unknown dispatch %s", msg.DispatchID)
			return nil
		}

		task, ok := m.reqList.TaskByID(allocationID)
		if !ok {
			log.Warnf("received DispatchStateChange for dispatch unknown to task list: %s", allocationID)
			return nil
		}

		alloc := m.reqList.Allocation(task.AllocationRef)
		if len(alloc.Resources) != 1 {
			log.Warnf("allocation has malformed resources: %v", alloc)
			return nil
		}

		_, exist := m.getHpcJobIDFromDispatchID(msg.DispatchID)
		if !exist && msg.HPCJobID != "" {
			hpcJobIDMsg := "HPC Job ID: " + msg.HPCJobID
			ctx.Tell(task.AllocationRef, sproto.ContainerLog{
				AuxMessage: &hpcJobIDMsg,
			})
			m.addDispatchIDToHpcJobIDMap(msg.DispatchID, msg.HPCJobID)

			// We have a job ID from the workload manager. Check if the user
			// canceled the job immediately after he/she submitted it.
			if _, ok = m.allocationsCanceledBeforeDispatchIDCreated.Load(string(allocationID)); ok {
				ctx.Log().WithField("allocation-id", allocationID).
					WithField("dispatch-id", msg.DispatchID).
					WithField("hpc-job-id", msg.HPCJobID).
					Infof("Received HPC job ID for job that has been canceled. Terminating launcher job.")

				// Send the dispatcher resource manager a request to terminate the job.
				ctx.Tell(ctx.Self(), KillDispatcherResources{
					AllocationID: allocationID,
				})
			}
		}

		r := maps.Values(alloc.Resources)[0]
		rID := r.Summary().ResourcesID

		task.State = schedulingStateFromDispatchState(msg.State)
		ctx.Tell(task.AllocationRef, sproto.ResourcesStateChanged{
			ResourcesID:      rID,
			ResourcesState:   resourcesStateFromDispatchState(msg.IsPullingImage, msg.State),
			ResourcesStarted: &sproto.ResourcesStarted{},
		})

	case DispatchExited:
		log := ctx.Log().WithField("dispatch-id", msg.DispatchID)
		allocationID, ok := m.getAllocationIDFromDispatchID(msg.DispatchID)
		if !ok {
			log.Warnf("received DispatchExited for unknown dispatch %s", msg.DispatchID)
			return nil
		}

		task, ok := m.reqList.TaskByID(allocationID)
		if !ok {
			log.Warnf("received DispatchExited for dispatch unknown to task list: %s", allocationID)
			return nil
		}

		alloc := m.reqList.Allocation(task.AllocationRef)
		if len(alloc.Resources) != 1 {
			log.Warnf("allocation has malformed resources: %v", alloc)
			return nil
		}
		r := maps.Values(alloc.Resources)[0]
		rID := r.Summary().ResourcesID

		if strings.TrimSpace(msg.Message) != "" {
			ctx.Tell(task.AllocationRef, sproto.ContainerLog{
				AuxMessage: &msg.Message,
				Level:      ptrs.Ptr("ERROR"),
			})
		}

		stopped := sproto.ResourcesStopped{}
		if msg.ExitCode > 0 {
			stopped.Failure = sproto.NewResourcesFailure(
				sproto.ResourcesFailed,
				"",
				ptrs.Ptr(sproto.ExitCode(msg.ExitCode)),
			)
		}

		// Turn off printing the last line (exit code 1) from resources.go
		if msg.ExitCode == -1 {
			stopped.Failure = sproto.NewResourcesFailure(
				sproto.ResourcesFailed,
				"",
				nil,
			)
		}

		ctx.Tell(task.AllocationRef, sproto.ResourcesStateChanged{
			ResourcesID:      rID,
			ResourcesState:   sproto.Terminated,
			ResourcesStopped: &stopped,
		})

		// Find the Dispatch IDs associated with the allocation ID. We'll need the
		// Dispatch ID to clean up the dispatcher environments for the job.
		dispatches, err := db.ListDispatchesByAllocationID(context.TODO(), allocationID)
		if err != nil {
			ctx.Log().WithError(err).Errorf(
				"Failed to retrieve the DispatchIDs associated with AllocationID %s",
				allocationID)
			return nil
		}
		ctx.Log().Debugf("Found %d jobs associated with AllocationID %s",
			len(dispatches), allocationID)

		// Cleanup all the dispatcher environments associated with current allocation
		for _, dispatch := range dispatches {
			dispatchID := dispatch.DispatchID
			impersonatedUser := dispatch.ImpersonatedUser

			if ctx.Log().Logger.Level < logrus.DebugLevel {
				ctx.Log().WithField("allocation-id", allocationID).Infof(
					"Deleting dispatcher environment for job with DispatchID %s initiated by %s",
					dispatchID, impersonatedUser)

				// Cleanup the dispatcher environment
				m.removeDispatchEnvironment(ctx, impersonatedUser, dispatchID)
			}
		}

		// Remove the dispatch from mapping tables.
		m.removeDispatchIDFromAllocationIDMap(msg.DispatchID)
		m.removeDispatchIDFromHpcJobIDMap(msg.DispatchID)

	case sproto.SetGroupMaxSlots:
		m.getOrCreateGroup(ctx, msg.Handler).MaxSlots = msg.MaxSlots

	case tasklist.GroupActorStopped:
		delete(m.groups, msg.Ref)

	case sproto.SetAllocationName:
		m.receiveSetTaskName(ctx, msg)

	case sproto.ResourcesReleased:
		m.resourcesReleased(ctx, msg.AllocationRef)

	default:
		ctx.Log().Errorf("receiveRequestMsg: unexpected message %T", msg)
		return actor.ErrUnexpectedMessage(ctx)
	}
	return nil
}

// Wait up to 2mins for the dispatch to be in a terminal state.
func (m *dispatcherResourceManager) waitForDispatchTerminalState(ctx *actor.Context,
	impersonatedUser string, dispatchID string,
) {
	for i := 0; i < 20; i++ {
		if m.jobWatcher.isDispatchInProgress(ctx, impersonatedUser, dispatchID) {
			ctx.Log().Debugf("Dispatch %s still active, waiting for termination.", dispatchID)
			time.Sleep(6 * time.Second)
		} else {
			return
		}
	}
	ctx.Log().Warnf("Dispatch %s still active, but wait time exceeded.  Continuing...", dispatchID)
}

func (m *dispatcherResourceManager) startLauncherJob(
	ctx *actor.Context,
	msg StartDispatcherResources,
) {
	req, ok := m.reqList.TaskByHandler(msg.TaskActor)
	if !ok {
		sendResourceStateChangedErrorResponse(ctx, errors.New("no such task"), msg,
			"task not found in the task list")
	}

	slotType, err := m.resolveSlotType(ctx, req.ResourcePool)
	if err != nil {
		sendResourceStateChangedErrorResponse(ctx, err, msg,
			"unable to access resource pool configuration")
		return
	}

	// Make sure we explicitly choose a partition.  Use default if unspecified.
	partition := req.ResourcePool
	if partition == "" {
		partition = m.selectDefaultPartition(slotType)
	}

	tresSupported := m.rmConfig.TresSupported
	gresSupported := m.rmConfig.GresSupported
	if m.rmConfig.TresSupported && !m.rmConfig.GresSupported {
		ctx.Log().Warnf("tres_supported: true cannot be used when " +
			"gres_supported: false is specified. Use tres_supported: false instead.")
		tresSupported = false
	}

	// Create the manifest that will be ultimately sent to the launcher.
	manifest, impersonatedUser, payloadName, err := msg.Spec.ToDispatcherManifest(
		m.rmConfig.MasterHost, m.rmConfig.MasterPort, m.masterTLSConfig.CertificateName,
		req.SlotsNeeded, slotType, partition, tresSupported, gresSupported,
		m.rmConfig.LauncherContainerRunType, m.wlmType == pbsSchedulerType,
		m.rmConfig.JobProjectSource,
	)
	if err != nil {
		sendResourceStateChangedErrorResponse(ctx, err, msg,
			"unable to create the launcher manifest")
		return
	}

	if impersonatedUser == root && m.rmConfig.UserName != root {
		sendResourceStateChangedErrorResponse(ctx,
			//nolint:stylecheck
			fmt.Errorf(
				"You are logged in as Determined user '%s', however the user ID on the "+
					"target HPC cluster for this user has either not been configured, or has "+
					"been set to the "+
					"disallowed value of 'root'. In either case, as a determined administrator, "+
					"use the command 'det user link-with-agent-user' to specify how jobs for "+
					"Determined user '%s' are to be launched on your HPC cluster.",
				msg.Spec.Owner.Username, msg.Spec.Owner.Username),
			msg, "")
		return
	}

	dispatchID, err := m.sendManifestToDispatcher(ctx, manifest, impersonatedUser)
	if err != nil {
		sendResourceStateChangedErrorResponse(ctx, err, msg,
			"unable to create the launcher job")
		return
	}

	//nolint:lll
	ctx.Log().WithField("allocation-id", msg.AllocationID).WithField("description", msg.Spec.Description).Infof("DispatchID is %s",
		dispatchID)

	m.addDispatchIDToAllocationMap(dispatchID, req.AllocationID)

	if err := db.InsertDispatch(context.TODO(), &db.Dispatch{
		DispatchID:       dispatchID,
		ResourceID:       msg.ResourcesID,
		AllocationID:     req.AllocationID,
		ImpersonatedUser: impersonatedUser,
	}); err != nil {
		ctx.Log().WithError(err).Errorf("failed to persist dispatch: %v", dispatchID)
	}

	// Check if the job was canceled before the launcher returned a dispatch ID.
	if _, ok = m.allocationsCanceledBeforeDispatchIDCreated.Load(string(req.AllocationID)); ok {
		ctx.Log().WithField("allocation-id", req.AllocationID).
			WithField("dispatch-id", dispatchID).
			Info("Obtained dispatch ID from launcher for job is pending cancelation.")
	}

	m.jobWatcher.monitorJob(impersonatedUser, dispatchID, payloadName)
}

// Adds the mapping of dispatch ID to allocation ID.
func (m *dispatcherResourceManager) addDispatchIDToAllocationMap(
	dispatchID string,
	allocationID model.AllocationID,
) {
	// Read/Write lock blocks other readers and writers.
	m.dispatchIDToAllocationIDMutex.Lock()
	defer m.dispatchIDToAllocationIDMutex.Unlock()

	m.dispatchIDToAllocationID[dispatchID] = allocationID
}

// Removes the mapping from dispatch ID to allocation ID.
func (m *dispatcherResourceManager) removeDispatchIDFromAllocationIDMap(
	dispatchID string,
) {
	// Read/Write lock blocks other readers and writers.
	m.dispatchIDToAllocationIDMutex.Lock()
	defer m.dispatchIDToAllocationIDMutex.Unlock()

	delete(m.dispatchIDToAllocationID, dispatchID)
}

// Gets the allocation ID for the specified dispatch ID.
func (m *dispatcherResourceManager) getAllocationIDFromDispatchID(
	dispatchID string,
) (model.AllocationID, bool) {
	// Read lock allows multiple readers, but block writers.
	m.dispatchIDToAllocationIDMutex.RLock()
	defer m.dispatchIDToAllocationIDMutex.RUnlock()

	allocationID, ok := m.dispatchIDToAllocationID[dispatchID]

	return allocationID, ok
}

// Adds the mapping of dispatch ID to HPC job ID.
func (m *dispatcherResourceManager) addDispatchIDToHpcJobIDMap(
	dispatchID string,
	hpcJobID string,
) {
	// Read/Write lock blocks other readers and writers.
	m.dispatchIDToHPCJobIDMutex.Lock()
	defer m.dispatchIDToHPCJobIDMutex.Unlock()

	m.dispatchIDToHPCJobID[dispatchID] = hpcJobID
}

// Removes the mapping from dispatch ID to allocaiton ID.
func (m *dispatcherResourceManager) removeDispatchIDFromHpcJobIDMap(
	dispatchID string,
) {
	// Read/Write lock blocks other readers and writers.
	m.dispatchIDToHPCJobIDMutex.Lock()
	defer m.dispatchIDToHPCJobIDMutex.Unlock()

	delete(m.dispatchIDToHPCJobID, dispatchID)
}

// Gets the HPC job ID for the specified dispatch ID.
func (m *dispatcherResourceManager) getHpcJobIDFromDispatchID(
	dispatchID string,
) (string, bool) {
	// Read lock allows multiple readers, but block writers.
	m.dispatchIDToHPCJobIDMutex.RLock()
	defer m.dispatchIDToHPCJobIDMutex.RUnlock()

	hpcJobID, ok := m.dispatchIDToHPCJobID[dispatchID]

	return hpcJobID, ok
}

// Log the failure, and send a ResourcesStateChanged describing the failure.
func sendResourceStateChangedErrorResponse(
	ctx *actor.Context, err error,
	msg StartDispatcherResources,
	errMessageStr string,
) {
	ctx.Log().WithError(err).Error(errMessageStr)
	stopped := sproto.ResourcesStopped{}
	stopped.Failure = sproto.NewResourcesFailure(
		sproto.ResourcesFailed,
		errors.Wrapf(err, errMessageStr).Error(),
		nil,
	)
	ctx.Tell(msg.TaskActor, sproto.ResourcesStateChanged{
		ResourcesID: msg.ResourcesID,
		// Could be a better message("container failed with non-zero exit code")
		ResourcesState:   sproto.Terminated,
		ResourcesStopped: &stopped,
	})
}

func (m *dispatcherResourceManager) receiveJobQueueMsg(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case sproto.GetJobQ:
		// TODO(HAL-2863): Get the job Q info from slurm, for the proper pool as per the message.
		ctx.Log().Debugf("GetJobQ for resource pool %s", msg.ResourcePool)
		ctx.Respond(m.jobQInfo(msg.ResourcePool))

	case *apiv1.GetJobQueueStatsRequest:
		// TODO(HAL-2863): Fill this in per-pool as discerned from the slurm resources info job.
		ctx.Log().Debugf("GetJobQueueStatsRequest, pool count %d", len(msg.ResourcePools))
		resp := &apiv1.GetJobQueueStatsResponse{
			Results: make([]*apiv1.RPQueueStat, 0),
		}
		// If no list of resource pools has been specified, return data for all pools.
		if (len(msg.ResourcePools)) == 0 {
			hpcDetails, err := m.fetchHpcResourceDetailsCached(ctx)
			if err != nil {
				ctx.Respond(resp)
				return nil
			}
			for _, p := range hpcDetails.Partitions {
				msg.ResourcePools = append(msg.ResourcePools, p.PartitionName)
			}
		}
		// Compute RPQueueStat results for each resource pool
		for _, resourcePool := range msg.ResourcePools {
			resp.Results = append(resp.Results, &apiv1.RPQueueStat{
				Stats:        tasklist.JobStatsByPool(m.reqList, resourcePool),
				ResourcePool: resourcePool,
			})
		}
		ctx.Respond(resp)

	case sproto.GetJobQStats:
		ctx.Log().Debugf("GetJobQStats for resource pool %s", msg.ResourcePool)
		// TODO(HAL-2863): Fill this in for the given pool as discerned from the slurm resources
		// info job.
		ctx.Respond(tasklist.JobStats(m.reqList))

	case sproto.SetGroupWeight, sproto.SetGroupPriority, sproto.MoveJob:
		// TODO(HAL-2863): We may not be able to support these specific actions, but how we
		// let people interact with the job queue in dispatcher/slurm world.
		// ctx.Respond(fmt.Errorf("modifying job positions is not yet supported in slurm"))
		if ctx.ExpectingResponse() {
			ctx.Respond(rmerrors.ErrUnsupported(
				fmt.Sprintf("%T unsupported in the dispatcher RM", msg)))
		}
		return nil

	case sproto.DeleteJob:
		ctx.Log().Infof("Delete job %s", string(msg.JobID))

		dispatches, err := db.ListDispatchesByJobID(context.TODO(), string(msg.JobID))
		if err != nil {
			ctx.Log().WithError(err).Errorf(
				"Failed to retrieve the DispatchIDs associated with Job %s",
				msg.JobID)
			ctx.Respond(sproto.DeleteJobResponseOf(err))
			return nil
		}
		for _, dispatch := range dispatches {
			ctx.Log().Debugf("Found dispatch %s associated with job %s", dispatch.DispatchID, msg.JobID)
			m.removeDispatchEnvironment(ctx, dispatch.ImpersonatedUser, dispatch.DispatchID)
		}
		ctx.Log().Debugf("Delete job successful %s", msg.JobID)
		ctx.Respond(sproto.EmptyDeleteJobResponse())

	default:
		return actor.ErrUnexpectedMessage(ctx)
	}
	return nil
}

// selectDefaultPools identifies partitions suitable as default compute and default
// aux partitions (if possible).
func (m *dispatcherResourceManager) selectDefaultPools(
	ctx *actor.Context, hpcResourceDetails []hpcPartitionDetails,
) (string, string) {
	// The default compute pool is the default partition if it has any GPUS,
	// otherwise select any partition with GPUs.
	// The AUX partition, use the default partition if available, otherwise any partition.

	defaultComputePar := "" // Selected default Compute/GPU partition
	defaultAuxPar := ""     // Selected default Aux partition

	fallbackComputePar := "" // Fallback Compute/GPU partition (has GPUs)
	fallbackAuxPar := ""     // Fallback partition if no default

	for _, v := range hpcResourceDetails {
		if v.IsDefault {
			defaultAuxPar = v.PartitionName
			if v.TotalGpuSlots > 0 {
				defaultComputePar = v.PartitionName
			}
		} else {
			fallbackAuxPar = v.PartitionName
			if v.TotalGpuSlots > 0 {
				fallbackComputePar = v.PartitionName
			}
		}
	}

	// Ensure we have a default aux, even if no partitions marked as such
	if defaultAuxPar == "" {
		defaultAuxPar = fallbackAuxPar
	}

	// If no default compute/GPU partitions, use a fallback partition
	if defaultComputePar == "" {
		if fallbackComputePar != "" {
			defaultComputePar = fallbackComputePar
		} else {
			defaultComputePar = defaultAuxPar
		}
	}

	// If explicitly configured, just override.
	if m.rmConfig.DefaultComputeResourcePool != nil {
		defaultComputePar = *m.rmConfig.DefaultComputeResourcePool
	}
	if m.rmConfig.DefaultAuxResourcePool != nil {
		defaultAuxPar = *m.rmConfig.DefaultAuxResourcePool
	}

	return defaultComputePar, defaultAuxPar
}

// selectDefaultPartition picks the default partition for the current system.
func (m *dispatcherResourceManager) selectDefaultPartition(slotType device.Type) string {
	m.resourceDetails.mu.RLock()
	defer m.resourceDetails.mu.RUnlock()
	if slotType == device.CPU {
		return m.resourceDetails.lastSample.DefaultAuxPoolPartition
	}
	return m.resourceDetails.lastSample.DefaultComputePoolPartition
}

// summarizeResourcePool retrieves details regarding hpc resources of the underlying system.
func (m *dispatcherResourceManager) summarizeResourcePool(
	ctx *actor.Context,
) ([]*resourcepoolv1.ResourcePool, error) {
	hpcResourceDetails, err := m.fetchHpcResourceDetailsCached(ctx)
	if err != nil {
		return nil, err
	}
	wlmName, schedulerType, fittingPolicy := m.getWlmResources()
	var result []*resourcepoolv1.ResourcePool
	poolNameMap := make(map[string]*resourcepoolv1.ResourcePool)

	for _, v := range hpcResourceDetails.Partitions {
		slotType, err := m.resolveSlotType(ctx, v.PartitionName)
		if err != nil {
			return nil, fmt.Errorf("resolving slot type: %w", err)
		}

		slotsAvailable := int32(v.TotalGpuSlots)
		slotsUsed := int32(v.TotalGpuSlots - v.TotalAvailableGpuSlots)
		if slotType == device.CPU {
			slotsAvailable = int32(v.TotalCPUSlots)
			slotsUsed = int32(v.TotalCPUSlots - v.TotalAvailableCPUSlots)
		}
		slotsPerAgent := 0
		if v.TotalNodes != 0 {
			slotsPerAgent = int(slotsAvailable) / v.TotalNodes
		}

		pool := resourcepoolv1.ResourcePool{
			Name:                         v.PartitionName,
			Description:                  wlmName + "-managed pool of resources",
			Type:                         resourcepoolv1.ResourcePoolType_RESOURCE_POOL_TYPE_STATIC,
			NumAgents:                    int32(v.TotalNodes),
			SlotType:                     slotType.Proto(),
			SlotsAvailable:               slotsAvailable,
			SlotsUsed:                    slotsUsed,
			AuxContainerCapacity:         int32(v.TotalCPUSlots),
			AuxContainersRunning:         int32(v.TotalCPUSlots - v.TotalAvailableCPUSlots),
			DefaultComputePool:           v.PartitionName == m.selectDefaultPartition(device.CUDA),
			DefaultAuxPool:               v.PartitionName == m.selectDefaultPartition(device.CPU),
			Preemptible:                  true,
			MinAgents:                    int32(v.TotalNodes),
			MaxAgents:                    int32(v.TotalNodes),
			SlotsPerAgent:                int32(slotsPerAgent),
			AuxContainerCapacityPerAgent: 0,
			SchedulerType:                schedulerType,
			SchedulerFittingPolicy:       fittingPolicy,
			Location:                     wlmName,
			ImageId:                      "",
			InstanceType:                 wlmName,
			Details:                      &resourcepoolv1.ResourcePoolDetail{},
		}
		poolNameMap[pool.Name] = &pool
		result = append(result, &pool)
	}
	result = append(result, m.getLauncherProvidedPools(poolNameMap, ctx)...)
	return result, nil
}

// getLauncherProvidedPools provides data for any launcher-provided resource pools
// from the master configuration.
func (m *dispatcherResourceManager) getLauncherProvidedPools(
	poolNameMap map[string]*resourcepoolv1.ResourcePool,
	ctx *actor.Context,
) []*resourcepoolv1.ResourcePool {
	var result []*resourcepoolv1.ResourcePool
	for _, pool := range m.poolConfig {
		if isValidProvider(pool) {
			basePoolName := pool.Provider.HPC.Partition
			basePool, found := poolNameMap[basePoolName]
			if !found {
				ctx.Log().Debugf("launcher-defined pool %s, base pool %s does not exist",
					pool.PoolName, basePoolName)
				continue
			}
			// If the base resource pool was located in the map provided, make
			// a copy, update the name to the launcher-provided pool name, and
			// include it in the result.
			launcherPoolResult := duplicateResourcePool(basePool)
			launcherPoolResult.Name = pool.PoolName
			if pool.Description != "" {
				launcherPoolResult.Description = pool.Description
			}
			result = append(result, launcherPoolResult)
		}
	}
	return result
}

// isValidProvider returns true is a usable Provider definition has been provided.
func isValidProvider(pool config.ResourcePoolConfig) bool {
	return pool.Provider != nil && pool.Provider.HPC != nil
}

func duplicateResourcePool(basePool *resourcepoolv1.ResourcePool) *resourcepoolv1.ResourcePool {
	return proto.Clone(basePool).(*resourcepoolv1.ResourcePool)
}

// getWlmResources returns various WLM-dependent resources used in constructing a resource pool.
func (m *dispatcherResourceManager) getWlmResources() (
	string, resourcepoolv1.SchedulerType, resourcepoolv1.FittingPolicy,
) {
	switch m.wlmType {
	case slurmSchedulerType:
		return "Slurm", resourcepoolv1.SchedulerType_SCHEDULER_TYPE_SLURM,
			resourcepoolv1.FittingPolicy_FITTING_POLICY_SLURM
	case pbsSchedulerType:
		return "PBS", resourcepoolv1.SchedulerType_SCHEDULER_TYPE_PBS,
			resourcepoolv1.FittingPolicy_FITTING_POLICY_PBS
	default:
		return "Unknown", resourcepoolv1.SchedulerType_SCHEDULER_TYPE_UNSPECIFIED,
			resourcepoolv1.FittingPolicy_FITTING_POLICY_UNSPECIFIED
	}
}

// fetchHpcResourceDetailsCached fetches cached Slurm resource details from the launcher node.
// If the cached info is too old, a cache reload will occur, and the candidates for the
// default compute & aux resource pools will be reevaluated.
func (m *dispatcherResourceManager) fetchHpcResourceDetailsCached(ctx *actor.Context) (
	hpcResources, error,
) {
	// If anyone is viewing the 'Cluster' section of the DAI GUI then there is activity here
	// about every 10s per user. To mitigate concerns of overloading slurmd with polling
	// activity, we will return a cached result, updating the cache only every so often.
	m.resourceDetails.mu.Lock()
	defer m.resourceDetails.mu.Unlock()
	if !m.resourceDetails.isUpdating &&
		(time.Since(m.resourceDetails.sampleTime).Seconds() > maxResourceDetailsSampleAgeSeconds) {
		m.resourceDetails.isUpdating = true
		go m.fetchHpcResourceDetails(ctx)
	}
	return m.resourceDetails.lastSample, nil
}

// resolveSlotType resolves the correct slot type for a job targeting the given partition. If the
// slot type is specified in the master config, use that. Otherwise if the partition is specified
// and known, and has no GPUs select CPU as the processor type, else default to CUDA.
func (m *dispatcherResourceManager) resolveSlotType(
	ctx *actor.Context,
	partition string,
) (device.Type, error) {
	if slotType := m.rmConfig.ResolveSlotType(partition); slotType != nil {
		return *slotType, nil
	}

	hpc, err := m.fetchHpcResourceDetailsCached(ctx)
	if err != nil {
		return "", fmt.Errorf("inferring slot type for resource info: %w", err)
	}

	for _, v := range hpc.Partitions {
		if v.PartitionName == partition && v.TotalGpuSlots == 0 {
			return device.CPU, nil
		}
	}
	return device.CUDA, nil
}

// retrieves the launcher version and log error if not meeting minimum required version.
func (m *dispatcherResourceManager) getAndCheckLauncherVersion(ctx *actor.Context) {
	resp, _, err := m.apiClient.InfoApi.GetServerVersion(m.authContext(ctx)).
		Execute() //nolint:bodyclose
	if err == nil {
		if checkMinimumLauncherVersion(resp) {
			if !m.launcherVersionIsOK {
				m.launcherVersionIsOK = true
				ctx.Log().Infof("Determined HPC launcher %s at %s:%d",
					resp, m.rmConfig.LauncherHost, m.rmConfig.LauncherPort)
			}
		}
	}
	if !m.launcherVersionIsOK {
		ctx.Log().Errorf("Launcher version %s does not meet the required minimum. "+
			"Upgrade to hpe-hpc-launcher version %s",
			resp, launcherMinimumVersion)
	}
}

func checkMinimumLauncherVersion(version string) bool {
	c, _ := semvar.NewConstraint(">=" + launcherMinimumVersion)
	launcherVersion, err := semvar.NewVersion(strings.TrimSuffix(version, "-SNAPSHOT"))
	if err != nil {
		return false
	}
	return c.Check(launcherVersion)
}

// fetchHpcResourceDetails retrieves the details about HPC Resources.
// This function uses HPC Resources manifest to retrieve the required details.
// This function performs the following steps:
//  1. Launch the manifest.
//  2. Read the log file with details on HPC resources.
//  3. Parse and load the details into a predefined struct - HpcResourceDetails
//  4. Terminate the manifest.
//
// Returns struct with HPC resource details - HpcResourceDetails.
// This function also queries launcher version and warns user if minimum required
// launcher version is not met.
func (m *dispatcherResourceManager) fetchHpcResourceDetails(ctx *actor.Context) {
	impersonatedUser := ""
	newSample := hpcResources{}

	// Below code will ensure isUpdating flag of the cache is always set to false,
	// while exiting the function.
	defer func() {
		m.resourceDetails.mu.Lock()
		m.resourceDetails.isUpdating = false
		m.resourceDetails.mu.Unlock()
	}()

	// Launch the HPC Resources manifest. Launch() method will ensure
	// the manifest is in the RUNNING state on successful completion.
	dispatchInfo, r, err := m.apiClient.LaunchApi.
		Launch(m.authContext(ctx)).
		Manifest(*m.hpcResourcesManifest).
		Impersonate(impersonatedUser).
		Execute() //nolint:bodyclose
	if err != nil {
		if r != nil && (r.StatusCode == http.StatusUnauthorized ||
			r.StatusCode == http.StatusForbidden) {
			ctx.Log().Errorf("Failed to communicate with launcher due to error: "+
				"{%v}. Reloaded the auth token file {%s}. If this error persists, restart "+
				"the launcher service followed by a restart of the determined-master service.",
				err, m.rmConfig.LauncherAuthFile)
			m.authToken = loadAuthToken(m.rmConfig)
			m.jobWatcher.ReloadAuthToken()
		} else {
			ctx.Log().Errorf("Failed to communicate with launcher due to error: "+
				"{%v}, response: {%v}. Verify that the launcher service is up and reachable."+
				" Try a restart the launcher service followed by a restart of the "+
				"determined-master service. ", err, r)
		}
		return
	}
	ctx.Log().Debugf("Launched Manifest with DispatchID %s", dispatchInfo.GetDispatchId())

	dispatchID := dispatchInfo.GetDispatchId()

	m.determineWlmType(dispatchInfo, ctx)

	owner := "launcher"

	defer m.resourceQueryPostActions(ctx, dispatchID, owner)

	logFileName := "slurm-resources-info"
	// HPC resource details will be listed in a log file with name
	// 'slurm-resources-info' in YAML format. Use LoadEnvironmentLog()
	// method to retrieve the log file.
	//
	// Because we're using "launch()" instead of "launchAsync()" to get
	// the HPC resources, we can expect that the "slurm-resources-info" log
	// file containing the SLURM partition info will be available, because
	// "launch()" will not return until the "slurm-resources-info" file is
	// written. Had we used "launchAsync()", we would have to poll the launcher
	// for job completion, but that's tricky, because the monitoring API will
	// go through the SlurmCarrier on the launcher side, which expects a job ID.
	// The SlurmCarrier will hang for a while waiting for the SLURM job ID to be
	// written, which it never will, because SlurmResources only queries SLURM
	// to get the partition info and does not create a job, so no job ID is ever
	// generated.  Eventually it will timeout waiting and return, but that's too
	// long of a delay for us to deal with.
	resp, _, err := m.apiClient.MonitoringApi.
		LoadEnvironmentLog(m.authContext(ctx), owner, dispatchID, logFileName).
		Execute() //nolint:bodyclose
	if err != nil {
		ctx.Log().WithError(err).Errorf("failed to retrieve HPC Resource details. response: {%v}", resp)
		return
	}

	// Parse the HPC resources file and extract the details into a
	// HpcResourceDetails object using YAML package.
	resourcesBytes, err := io.ReadAll(resp)
	if err != nil {
		ctx.Log().WithError(err).Errorf("failed to read response")
		return
	}
	if err = yaml.Unmarshal(resourcesBytes, &newSample); err != nil {
		ctx.Log().WithError(err).Errorf("failed to parse HPC Resource details")
		return
	}
	m.hpcResourcesToDebugLog(ctx, newSample)

	m.resourceDetails.mu.Lock()
	defer m.resourceDetails.mu.Unlock()
	m.resourceDetails.lastSample = newSample
	m.resourceDetails.sampleTime = time.Now()
	m.resourceDetails.lastSample.DefaultComputePoolPartition,
		m.resourceDetails.lastSample.DefaultAuxPoolPartition = m.selectDefaultPools(
		ctx, m.resourceDetails.lastSample.Partitions)
	ctx.Log().Debugf("default resource pools are '%s', '%s'",
		m.resourceDetails.lastSample.DefaultComputePoolPartition,
		m.resourceDetails.lastSample.DefaultAuxPoolPartition)

	if !m.launcherVersionIsOK {
		m.getAndCheckLauncherVersion(ctx)
	}
}

// determineWlmType determines the WLM type of the cluster from the dispatchInfo response.
func (m *dispatcherResourceManager) determineWlmType(
	dispatchInfo launcher.DispatchInfo, ctx *actor.Context,
) {
	reporter := ""
	for _, e := range *dispatchInfo.Events {
		if strings.HasPrefix(e.GetMessage(), "Successfully launched payload") {
			reporter = e.GetReporter()
			break
		}
	}
	switch reporter {
	case slurmResourcesCarrier:
		m.wlmType = slurmSchedulerType
	case pbsResourcesCarrier:
		m.wlmType = pbsSchedulerType
	}
}

// hpcResourcesToDebugLog puts a summary of the available HPC resources to the debug log.
func (m *dispatcherResourceManager) hpcResourcesToDebugLog(
	ctx *actor.Context, resources hpcResources,
) {
	if ctx.Log().Logger.Level != logrus.DebugLevel {
		return
	}
	ctx.Log().Debugf("HPC Resource details: %+v", resources.Partitions)
	nodesWithGpu := 0
	gpusFound := 0
	nodesAllocated := 0
	gpusAllocated := 0
	cpusFound := 0
	cpusAllocated := 0
	for _, node := range resources.Nodes {
		gpusFound += node.GpuCount
		cpusFound += node.CPUCount
		if node.GpuCount > 0 {
			nodesWithGpu++
		}
		if node.Allocated {
			nodesAllocated++
		}
		gpusAllocated += node.GpuInUseCount
		cpusAllocated += node.CPUInUseCount
	}
	ctx.Log().
		WithField("nodes", len(resources.Nodes)).
		WithField("allocated", nodesAllocated).
		WithField("nodes with GPU", nodesWithGpu).
		WithField("GPUs", gpusFound).
		WithField("GPUs allocated", gpusAllocated).
		WithField("CPUs", cpusFound).
		WithField("CPUs allocated", cpusAllocated).
		Debug("Node summary")
}

// resourceQueryPostActions performs actions to clean up after any dispatch
// completion (either a Slurm resource query, or launched manifest allocation).
// In the case of retrieving the details of HPC Resources, the job is synchronous
// and is not being monitored, removeDispatchEnvironment is called to remove the
// slurm-resources-info file.
// We use dispatcher REST API calls to instruct the dispatcher to clean up.
// On success, the Dispatch (if present) is removed from the DB (if present).
// When querying Slurm resource information, the DispatchID is not registered
// with the DB, so we do not log an error if we fail to delete it.
// On any REST failure where we cannot confirm the dispatch has been removed
// by the launcher, we skip any attempt to delete the Dispatch from the DB.
// The Dispatch is left in the DB, for a future cleanup attempt on startup.
func (m *dispatcherResourceManager) resourceQueryPostActions(ctx *actor.Context,
	dispatchID string, owner string,
) {
	if m.terminateDispatcherJob(ctx, dispatchID, owner, true) {
		m.removeDispatchEnvironment(ctx, owner, dispatchID)
	}
}

// terminateDispatcherJob terminates the dispatcher job with the given ID.
// Return true to indicate if the DB dispatch should additionally be deleted.
func (m *dispatcherResourceManager) terminateDispatcherJob(ctx *actor.Context,
	dispatchID string, owner string, slurmResourcesPolling bool,
) bool {
	if dispatchID == "" {
		ctx.Log().Warn("Missing dispatchID, so no environment clean-up")
		return false
	}

	var err error
	var response *http.Response
	if _, response, err = m.apiClient.RunningApi.TerminateRunning(m.authContext(ctx),
		owner, dispatchID).Force(true).Execute(); err != nil { //nolint:bodyclose
		if response == nil || response.StatusCode != 404 {
			ctx.Log().WithError(err).Errorf("Failed to terminate job with Dispatch ID %s, response: {%v}",
				dispatchID, response)
			// We failed to delete, and not 404/notfound so leave in DB.
			return false
		}
	}

	if slurmResourcesPolling {
		ctx.Log().Debugf("Terminated job with DispatchID %s", dispatchID)
	} else {
		ctx.Log().Infof("Terminated job with DispatchID %s", dispatchID)
	}

	// Let the job monitor know that the job was terminated, otherwise it
	// might get a 404 (Not Found) error from the launcher and not send
	// Determined notification that the job was terminated.
	m.jobWatcher.markJobAsTerminated(ctx, dispatchID)

	return true
}

// removeDispatchEnvironment uses the dispatcher REST API to remove
// the environment created on the launcher node in support of the
// job with the specified dispatch ID. This prevents stale information
// from accumulating in the dispatcher.  Upon success, it additionally
// attempts to remove the dispatchID association (if present) with the allocation
// in the DB.  On failure, the attempt to remove the Dispatch
// from the DB is skipped and left for a future cleanup attempt on startup.
// When querying Slurm resource information, the DispatchID is not registered
// with the DB, so we do not log an error if we fail to remove it.
func (m *dispatcherResourceManager) removeDispatchEnvironment(
	ctx *actor.Context, owner string, dispatchID string,
) {
	ctx.Log().Debugf("Deleting environment with DispatchID %s", dispatchID)

	if response, err := m.apiClient.MonitoringApi.DeleteEnvironment(m.authContext(ctx),
		owner, dispatchID).Execute(); err != nil { //nolint:bodyclose
		if response == nil || response.StatusCode != 404 {
			ctx.Log().WithError(err).Errorf("Failed to remove environment for Dispatch ID %s, response:{%v}",
				dispatchID, response)
			// We failed to delete, and not 404/notfound so leave in DB for later retry
			return
		}
	} else {
		ctx.Log().Debugf("Deleted environment with DispatchID %s", dispatchID)
	}
	count, err := db.DeleteDispatch(context.TODO(), dispatchID)
	if err != nil {
		ctx.Log().WithError(err).Errorf("Failed to delete DispatchID %s from DB", dispatchID)
	}
	// On Slurm resource query there may be no Dispatch in the DB, so only log as trace.
	ctx.Log().Tracef("Deleted DispatchID %s from DB, count %d", dispatchID, count)
}

// Sends the manifest to the launcher.
func (m *dispatcherResourceManager) sendManifestToDispatcher(
	ctx *actor.Context,
	manifest *launcher.Manifest,
	impersonatedUser string,
) (string, error) {
	/*
	 * "LaunchAsync()" does not wait for the "launcher" to move the job to the "RUNNING"
	 * state and returns right away while the job is still in the "PENDING" state. If it
	 * becomes necessary to wait for the job to be in the "RUNNING" state, we can switch
	 * to using "Launch()".
	 *
	 * The "manifest" describes the job to be launched and includes any environment
	 * variables, mount points, etc., that are needed by the job.
	 *
	 * The "impersonatedUser" is the user that we want to run the job as on the cluster.
	 * Of course, that user must be known to the cluster as either a local Linux user
	 * (e.g. "/etc/passwd"), LDAP, or some other authentication mechanism.
	 */
	dispatchInfo, response, err := m.apiClient.LaunchApi.
		LaunchAsync(m.authContext(ctx)).
		Manifest(*manifest).
		Impersonate(impersonatedUser).
		Execute() //nolint:bodyclose
	if err != nil {
		httpStatus := ""
		if response != nil {
			// So we can show the HTTP status code, if available.
			httpStatus = fmt.Sprintf("(HTTP status %d)", response.StatusCode)
		}
		return "", errors.Wrapf(err, "LaunchApi.LaunchAsync() returned an error %s, response: {%v}. "+
			"Verify that the launcher service is up and reachable. Try a restart the "+
			"launcher service followed by a restart of the determined-master service.", httpStatus, response)
	}
	return dispatchInfo.GetDispatchId(), nil
}

func (m *dispatcherResourceManager) addTask(ctx *actor.Context, msg sproto.AllocateRequest) {
	actors.NotifyOnStop(ctx, msg.AllocationRef, sproto.ResourcesReleased{
		AllocationRef: msg.AllocationRef,
	})

	if len(msg.AllocationID) == 0 {
		msg.AllocationID = model.AllocationID(uuid.New().String())
	}
	if msg.Group == nil {
		msg.Group = msg.AllocationRef
	}
	m.getOrCreateGroup(ctx, msg.Group)
	if len(msg.Name) == 0 {
		msg.Name = "Unnamed-Launcher-Job"
	}

	ctx.Log().Infof(
		"resources are requested by %s (Allocation ID: %s)",
		msg.AllocationRef.Address(), msg.AllocationID,
	)
	m.reqList.AddTask(&msg)
}

func (m *dispatcherResourceManager) jobQInfo(rp string) map[model.JobID]*sproto.RMJobInfo {
	var reqs []*sproto.AllocateRequest
	for it := m.reqList.Iterator(); it.Next(); {
		if it.Value().ResourcePool == rp {
			reqs = append(reqs, it.Value())
		}
	}
	return tasklist.ReduceToJobQInfo(reqs)
}

func (m *dispatcherResourceManager) receiveSetTaskName(
	ctx *actor.Context, msg sproto.SetAllocationName,
) {
	if task, found := m.reqList.TaskByHandler(msg.AllocationRef); found {
		task.Name = msg.Name
	}
}

func (m *dispatcherResourceManager) assignResources(
	ctx *actor.Context, req *sproto.AllocateRequest,
) {
	var dispatchID string
	var impersonatedUser string
	var rID sproto.ResourcesID

	if req.Restore {
		// Find the Dispatch IDs associated with the allocation ID. We'll need the
		// Dispatch ID to reconnect with the existing allocation.
		dispatches, err := db.ListDispatchesByAllocationID(context.TODO(), req.AllocationID)
		if err != nil {
			ctx.Log().WithError(err).Errorf(
				"Failed to retrieve the DispatchIDs associated with AllocationID %s",
				req.AllocationID)
			return
		}

		ctx.Log().Debugf("Restore: Found %d jobs associated with AllocationID %s",
			len(dispatches), req.AllocationID)

		for _, dispatch := range dispatches {
			dispatchID = dispatch.DispatchID
			impersonatedUser = dispatch.ImpersonatedUser
			rID = dispatch.ResourceID
			break
		}
	}

	if len(rID) == 0 {
		rID = sproto.ResourcesID(uuid.NewString())
	}
	allocations := sproto.ResourceList{
		rID: &DispatcherResources{
			id:                     rID,
			req:                    req,
			rm:                     ctx.Self(),
			group:                  m.groups[req.Group],
			defaultRendezvousIface: m.rmConfig.ResolveRendezvousNetworkInterface(req.ResourcePool),
			defaultProxyIface:      m.rmConfig.ResolveProxyNetworkInterface(req.ResourcePool),
		},
	}

	assigned := sproto.ResourcesAllocated{ID: req.AllocationID, Resources: allocations}
	m.reqList.AddAllocationRaw(req.AllocationRef, &assigned)
	req.AllocationRef.System().Tell(req.AllocationRef, assigned)

	if req.Restore {
		if len(dispatchID) == 0 {
			ctx.Log().Infof("Restore request with no active DispatchID found.  Fail the allocation request.")
			failed := sproto.NewResourcesFailure(sproto.ResourcesAborted,
				"Unable to locate HPC job on restart.", nil)
			stopped := sproto.ResourcesStopped{}
			stopped.Failure = failed
			ctx.Tell(req.AllocationRef, sproto.ResourcesStateChanged{
				ResourcesID:      rID,
				ResourcesState:   sproto.Terminated,
				ResourcesStopped: &stopped,
			})
		} else {
			// Simulate portions of Start() which will not be called on restore.
			ctx.Log().Infof("Reconnecting ResourceID %s, DispatchID %s, ImpersontatedUser: %s",
				rID, dispatchID, impersonatedUser)

			m.addDispatchIDToAllocationMap(dispatchID, req.AllocationID)

			m.jobWatcher.monitorJob(impersonatedUser, dispatchID, "")
		}
	} else {
		ctx.Log().
			WithField("allocation-id", req.AllocationID).
			WithField("task-handler", req.AllocationRef.Address()).
			Infof("resources assigned")
	}
}

func (m *dispatcherResourceManager) resourcesReleased(ctx *actor.Context, handler *actor.Ref) {
	ctx.Log().Infof("resources are released for %s", handler.Address())
	m.reqList.RemoveTaskByHandler(handler)
}

// Used on DEBUG startup, to queue a terminate and delete all dispatches in the DB
// that are no-longer associated with an active experiment/task.
// All active tasks, will get reconnected via AllocationRequest{Restore:true}
// events.   This path is currently used only for debug mode where we defer
// cleanup of dispatches until the restart to enable debugging of job
// logs -- TODO:  Currently it terminates all dispatches as we do
// not yet know how to identify active allocations during startup.
func (m *dispatcherResourceManager) debugKillAllInactiveDispatches(
	ctx *actor.Context, handler *actor.Ref,
) {
	if ctx.Log().Logger.Level < logrus.DebugLevel {
		return
	}

	ctx.Log().Infof("Releasing all resources due to master restart in DEBUG mode. " +
		"Jobs will not be restored, but may restart.")

	// Find the Dispatch IDs associated with the allocation ID. We'll need the
	// Dispatch ID to cancel the job on the launcher side.
	dispatches, err := db.ListAllDispatches(context.TODO())
	if err != nil {
		ctx.Log().WithError(err).Errorf("Failed to retrieve all Dispatches")
		return
	}
	ctx.Log().Debugf("Found %d Dispatches to release", len(dispatches))
	for _, dispatch := range dispatches {
		ctx.Log().Debugf("Queuing cleanup of AllocationID %s, DispatchID %s",
			dispatch.AllocationID, dispatch.DispatchID)
		ctx.Tell(handler, KillDispatcherResources{
			ResourcesID:  dispatch.ResourceID,
			AllocationID: dispatch.AllocationID,
		})
	}
}

func (m *dispatcherResourceManager) getOrCreateGroup(
	ctx *actor.Context,
	handler *actor.Ref,
) *tasklist.Group {
	if g, ok := m.groups[handler]; ok {
		return g
	}
	priority := config.KubernetesDefaultPriority
	g := &tasklist.Group{Handler: handler, Weight: 1, Priority: &priority}
	m.groups[handler] = g

	if ctx != nil && handler != nil { // ctx is nil only for testing purposes.
		actors.NotifyOnStop(ctx, handler, tasklist.GroupActorStopped{})
	}
	return g
}

func (m *dispatcherResourceManager) schedulePendingTasks(ctx *actor.Context) {
	for it := m.reqList.Iterator(); it.Next(); {
		req := it.Value()
		assigned := m.reqList.Allocation(req.AllocationRef)
		if !tasklist.AssignmentIsScheduled(assigned) {
			m.assignResources(ctx, req)
		}
	}
}

// When the launcher returns an error from "terminateDispatcherJob()", this
// function is called to schedule another retry after "waitTime" seconds and
// wil do the retries for up to "numRetries".
func performFailedJobTerminationRetries(
	ctx *actor.Context,
	dispatchID string,
	allocationID model.AllocationID,
	waitTime int,
	numRetries int,
	jobTerminationFunc func(ctx *actor.Context, allocationID model.AllocationID),
) {
	retriesExhausted := false

	val, ok := failedJobTerminations.Load(dispatchID)

	if ok {
		// The dispatch ID exists in the map. If we have not exhausted all
		// retries, then increment the retry count.
		if val.(int) < numRetries {
			failedJobTerminations.Store(dispatchID, val.(int)+1)
		} else {
			// All retries have been exhausted.
			retriesExhausted = true
		}
	} else {
		// The dispatch ID doesn't exist in the map, so add it with a count of
		// 1 retry.
		failedJobTerminations.Store(dispatchID, 1)
	}

	if !retriesExhausted {
		ctx.Log().WithField("allocation-id", allocationID).
			WithField("dispatch-id", dispatchID).
			Infof("Waiting %d seconds before resending KillDispatcherResources message",
				waitTime)

		waitAndSendAnotherJobTerminationMessage(ctx, allocationID, waitTime, jobTerminationFunc)
	} else {
		ctx.Log().WithField("allocation-id", allocationID).
			WithField("dispatch-id", dispatchID).
			Warn("Exhausted all job termination retries")

		// Remove the dispatch ID from the list, since we're not going to do
		// anymore retries.
		failedJobTerminations.Delete(dispatchID)
	}
}

// Sleeps for "waitTime" seconds, then sends another KillDispatcherResources
// message, which will invoke another call "terminateDispatcherJob()".
func waitAndSendAnotherJobTerminationMessage(
	ctx *actor.Context,
	allocationID model.AllocationID,
	waitTime int,
	jobTerminationFunc func(ctx *actor.Context, allocationID model.AllocationID),
) {
	time.Sleep(time.Duration(waitTime) * time.Second)

	// Send the KillDispatcherResources message.
	jobTerminationFunc(ctx, allocationID)
}

type (
	// DispatcherResources information.
	DispatcherResources struct {
		id    sproto.ResourcesID
		req   *sproto.AllocateRequest
		rm    *actor.Ref
		group *tasklist.Group

		defaultRendezvousIface string
		defaultProxyIface      string
	}

	// StartDispatcherResources comment to keep "golint" from complaining.
	StartDispatcherResources struct {
		AllocationID model.AllocationID
		ResourcesID  sproto.ResourcesID
		TaskActor    *actor.Ref
		Spec         tasks.TaskSpec
	}

	// KillDispatcherResources tells the dispatcher RM to clean up the resources with the given
	// resources ID.
	KillDispatcherResources struct {
		ResourcesID  sproto.ResourcesID
		AllocationID model.AllocationID
	}

	// DispatchStateChange notifies the dispatcher that the give dispatch has changed state.
	DispatchStateChange struct {
		DispatchID     string
		State          launcher.DispatchState
		IsPullingImage bool
		HPCJobID       string
	}

	// DispatchExited notifies the dispatcher that the give dispatch exited.
	DispatchExited struct {
		DispatchID string
		ExitCode   exitCode
		Message    string
	}
)

// Summary summarizes a container allocation.
func (r DispatcherResources) Summary() sproto.ResourcesSummary {
	return sproto.ResourcesSummary{
		ResourcesID:   r.id,
		ResourcesType: sproto.ResourcesTypeSlurmJob,
		AllocationID:  r.req.AllocationID,
		AgentDevices:  map[aproto.ID][]device.Device{},
		ContainerID:   nil,
	}
}

// Start notifies the pods actor that it should launch a pod for the provided task spec.
func (r DispatcherResources) Start(
	ctx *actor.Context, _ logger.Context, spec tasks.TaskSpec, rri sproto.ResourcesRuntimeInfo,
) error {
	spec.ResourcesID = string(r.id)
	spec.AllocationID = string(r.req.AllocationID)
	spec.AllocationSessionToken = rri.Token
	spec.TaskID = string(r.req.TaskID)
	spec.UseHostMode = rri.IsMultiAgent
	spec.ResourcesConfig.SetPriority(r.group.Priority)
	if spec.LoggingFields == nil {
		spec.LoggingFields = map[string]string{}
	}
	spec.LoggingFields["allocation_id"] = spec.AllocationID
	spec.LoggingFields["task_id"] = spec.TaskID
	spec.ExtraEnvVars[sproto.ResourcesTypeEnvVar] = string(sproto.ResourcesTypeSlurmJob)
	spec.ExtraEnvVars[sproto.SlurmRendezvousIfaceEnvVar] = r.defaultRendezvousIface
	spec.ExtraEnvVars[sproto.SlurmProxyIfaceEnvVar] = r.defaultProxyIface
	ctx.Tell(r.rm, StartDispatcherResources{
		AllocationID: r.req.AllocationID,
		ResourcesID:  r.id,
		TaskActor:    r.req.AllocationRef,
		Spec:         spec,
	})
	return nil
}

// Kill notifies the pods actor that it should stop the pod.
func (r DispatcherResources) Kill(ctx *actor.Context, _ logger.Context) {
	ctx.Tell(r.rm, KillDispatcherResources{ResourcesID: r.id, AllocationID: r.req.AllocationID})
}

// CreateSlurmResourcesManifest creates a Manifest for SlurmResources Carrier.
// This Manifest is used to retrieve information about resources available on the HPC system.
func createSlurmResourcesManifest() *launcher.Manifest {
	payload := launcher.NewPayloadWithDefaults()
	payload.SetName("DAI-HPC-Resources")
	payload.SetId("com.cray.analytics.capsules.hpc.resources")
	payload.SetVersion("latest")
	payload.SetCarriers([]string{slurmResourcesCarrier, pbsResourcesCarrier})

	// Create payload launch parameters
	launchParameters := launcher.NewLaunchParameters()
	launchParameters.SetMode("interactive")
	payload.SetLaunchParameters(*launchParameters)

	clientMetadata := launcher.NewClientMetadataWithDefaults()
	clientMetadata.SetName("DAI-HPC-Resources")

	// Create & populate the manifest
	manifest := *launcher.NewManifest("v1", *clientMetadata)
	manifest.SetPayloads([]launcher.Payload{*payload})

	return &manifest
}

// If an auth_file was specified, load the content and return it to enable authorization
//
//	with the launcher.  If the auth_file is configured, but does not exist we panic.
func loadAuthToken(config *config.DispatcherResourceManagerConfig) string {
	if len(config.LauncherAuthFile) > 0 {
		authToken, err := os.ReadFile(config.LauncherAuthFile)
		if err != nil {
			panic("Configuration resource_manager.auth_file not readable: " + config.LauncherAuthFile)
		}
		return string(authToken)
	}
	return ""
}

// schedulingStateFromDispatchState returns SchedulingState from DispatchState representation.
func schedulingStateFromDispatchState(state launcher.DispatchState) sproto.SchedulingState {
	switch state {
	case launcher.PENDING:
		return sproto.SchedulingStateQueued
	default:
		return sproto.SchedulingStateScheduled
	}
}

// resourcesStateFromDispatchState returns ResourcesState from DispatchState representation.
func resourcesStateFromDispatchState(
	isPullingImage bool,
	state launcher.DispatchState,
) sproto.ResourcesState {
	// The launcher has no state to indicate the image is being pulled, so we
	// have to test for that separately.
	if isPullingImage {
		return sproto.Pulling
	}

	switch state {
	case launcher.PENDING:
		return sproto.Assigned
	case launcher.RUNNING:
		return sproto.Running
	case launcher.TERMINATING:
		return sproto.Running
	case launcher.COMPLETED:
		return sproto.Terminated
	case launcher.FAILED:
		return sproto.Terminated
	default:
		return sproto.Unknown
	}
}

// NotifyContainerRunning receives a notification from the container to let
// the master know that the container is running.
func (d DispatcherResourceManager) NotifyContainerRunning(
	ctx actor.Messenger,
	msg sproto.NotifyContainerRunning,
) error {
	return d.Ask(ctx, msg, nil)
}

type taskContainerDefaults struct {
	fallbackDefault model.TaskContainerDefaultsConfig
	resourcePool    string
}

// TaskContainerDefaults returns TaskContainerDefaults for the specified pool.
func (d DispatcherResourceManager) TaskContainerDefaults(
	ctx actor.Messenger,
	resourcePoolName string,
	defaultConfig model.TaskContainerDefaultsConfig,
) (result model.TaskContainerDefaultsConfig, err error) {
	request := taskContainerDefaults{fallbackDefault: defaultConfig, resourcePool: resourcePoolName}
	return result, d.Ask(ctx, request, &result)
}
