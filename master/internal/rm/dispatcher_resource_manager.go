package rm

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

	"github.com/ghodss/yaml"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	launcher "github.hpe.com/hpe/hpc-ard-launcher-go/launcher"
	"golang.org/x/exp/maps"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/db"
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
	slurmSchedulerType    = "slurm"
	pbsSchedulerType      = "pbs"
	slurmResourcesCarrier = "com.cray.analytics.capsules.carriers.hpc.slurm.SlurmResources"
	pbsResourcesCarrier   = "com.cray.analytics.capsules.carriers.hpc.pbs.PbsResources"
)

// hpcResources is a data type describing the HPC resources available
// to Slurm on on the Launcher node.
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
	Partitions                  []hpcPartitionDetails `json:"partitions,flow"`
	Nodes                       []hpcNodeDetails      `json:"nodes,flow"`
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
		HasResourcePool bool
	}
)

// DispatcherResourceManager is a resource manager for managing slurm resources.
type DispatcherResourceManager struct {
	*ActorResourceManager
}

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
	ctx actor.Messenger, name string, slots int, command bool,
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

	if err := d.ValidateResourcePool(ctx, name); err != nil {
		return "", fmt.Errorf("validating resource pool: %w", err)
	}
	return name, nil
}

// ValidateResourcePool validates that the given resource pool exists.
func (d *DispatcherResourceManager) ValidateResourcePool(ctx actor.Messenger, name string) error {
	var resp hasSlurmPartitionResponse
	switch err := d.ask(ctx, hasSlurmPartitionRequest{PoolName: name}, &resp); {
	case err != nil:
		return fmt.Errorf("requesting slurm partition: %w", err)
	case !resp.HasResourcePool:
		return fmt.Errorf("slurm partition not found: %s", name)
	default:
		return nil
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

// NewDispatcherResourceManager returns a new dispatcher resource manager.
func NewDispatcherResourceManager(
	system *actor.System,
	db *db.PgDB,
	echo *echo.Echo,
	config *config.ResourceConfig,
	opts *aproto.MasterSetAgentOptions,
	cert *tls.Certificate,
) *DispatcherResourceManager {
	tlsConfig, err := makeTLSConfig(cert)
	if err != nil {
		panic(errors.Wrap(err, "failed to set up TLS config"))
	}

	var rm *dispatcherResourceManager
	if config.ResourceManager.DispatcherRM != nil {
		// slurm type is configured
		rm = newDispatcherResourceManager(
			config.ResourceManager.DispatcherRM, tlsConfig, opts.LoggingOptions,
		)
	} else {
		// pbs type is configured
		rm = newDispatcherResourceManager(
			config.ResourceManager.PbsRM, tlsConfig, opts.LoggingOptions,
		)
	}
	ref, _ := system.ActorOf(
		sproto.DispatcherRMAddr,
		rm,
	)
	system.Ask(ref, actor.Ping{}).Get()
	return &DispatcherResourceManager{ActorResourceManager: WrapRMActor(ref)}
}

// dispatcherResourceProvider manages the lifecycle of dispatcher resources.
type dispatcherResourceManager struct {
	config *config.DispatcherResourceManagerConfig

	apiClient                *launcher.APIClient
	hpcResourcesManifest     *launcher.Manifest
	reqList                  *taskList
	groups                   map[*actor.Ref]*group
	dispatchIDToAllocationID map[string]model.AllocationID
	slotsUsedPerGroup        map[*group]int
	masterTLSConfig          model.TLSClientConfig
	loggingConfig            model.LoggingConfig
	jobWatcher               *launcherMonitor
	authToken                string
	resourceDetails          hpcResourceDetailsCache
	wlmType                  string
}

func newDispatcherResourceManager(
	config *config.DispatcherResourceManagerConfig,
	masterTLSConfig model.TLSClientConfig,
	loggingConfig model.LoggingConfig,
) *dispatcherResourceManager {
	// Set up the host address and IP address of the "launcher".
	clientConfiguration := launcher.NewConfiguration()

	// Host, port, and protocol are configured in the "resource_manager" section
	// of the "tools/devcluster.yaml" file. The host address and port refer to the
	// system where the "launcher" is running.
	clientConfiguration.Host = fmt.Sprintf("%s:%d", config.LauncherHost, config.LauncherPort)
	clientConfiguration.Scheme = config.LauncherProtocol // "http" or "https"
	if config.Security != nil {
		logrus.Debugf("Launcher communications InsecureSkipVerify: %t", config.Security.TLS.SkipVerify)
		transCfg := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: config.Security.TLS.SkipVerify}, //nolint:gosec
		}
		clientConfiguration.HTTPClient = &http.Client{Transport: transCfg}
	}

	apiClient := launcher.NewAPIClient(clientConfiguration)

	// One time activity to create a manifest using SlurmResources carrier.
	// This manifest is used on demand to retrieve details regarding HPC resources
	// e.g., nodes, GPUs etc
	hpcResourcesManifest := createSlurmResourcesManifest()

	// Authentication token that gets passed to the "launcher" REST API.
	authToken := loadAuthToken(config)

	return &dispatcherResourceManager{
		config: config,

		apiClient:                apiClient,
		hpcResourcesManifest:     hpcResourcesManifest,
		reqList:                  newTaskList(),
		groups:                   make(map[*actor.Ref]*group),
		dispatchIDToAllocationID: make(map[string]model.AllocationID),
		slotsUsedPerGroup:        make(map[*group]int),

		masterTLSConfig: masterTLSConfig,
		loggingConfig:   loggingConfig,
		jobWatcher:      newDispatchWatcher(apiClient, authToken, config.LauncherAuthFile),
		authToken:       authToken,
	}
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
		sproto.ResourcesReleased,
		groupActorStopped:
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
		ctx.Respond(getTaskHandler(m.reqList, msg.ID))

	case sproto.GetAllocationSummary:
		if resp := getTaskSummary(m.reqList, msg.ID, m.groups, slurmSchedulerType); resp != nil {
			ctx.Respond(*resp)
		}

	case sproto.GetAllocationSummaries:
		ctx.Respond(getTaskSummaries(m.reqList, m.groups, slurmSchedulerType))

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
		result := false
		if err == nil {
			for _, p := range hpcDetails.Partitions {
				if p.PartitionName == msg.PoolName {
					result = true
					break
				}
			}
		}
		ctx.Respond(hasSlurmPartitionResponse{HasResourcePool: result})

	case sproto.ValidateCommandResourcesRequest:
		// TODO(HAL-2862): Use inferred value here if possible.
		// fulfillable := m.config.MaxSlotsPerContainer >= msg.Slots
		ctx.Respond(sproto.ValidateCommandResourcesResponse{Fulfillable: true})

	case schedulerTick:
		m.schedulePendingTasks(ctx)
		actors.NotifyAfter(ctx, actionCoolDown, schedulerTick{})

	case *apiv1.GetAgentsRequest:
		ctx.Respond(m.generateGetAgentsResponse(ctx))

	default:
		ctx.Log().Errorf("unexpected message %T", msg)
		return actor.ErrUnexpectedMessage(ctx)
	}

	return nil
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
		response.Agents = append(response.Agents, &agent)
		if node.GpuCount == 0 {
			addSlotToAgent(
				&agent, devicev1.Type_TYPE_CPU, node, node.CPUCount, node.Allocated) // One CPU slot/device
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

// computeSlotType computes an agent GPU slot type from the configuration data available.
// For nodes that are members of multiple partitions, take the first configured slot type found,
// falling back to CUDA if nothing found.
func computeSlotType(node hpcNodeDetails, m *dispatcherResourceManager) devicev1.Type {
	for _, partition := range node.Partitions {
		slotType := m.config.ResolveSlotType(partition)
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
		req := m.reqList.taskByHandler[msg.TaskActor]

		slotType, err := m.resolveSlotType(ctx, req.ResourcePool)
		if err != nil {
			sendResourceStateChangedErrorResponse(ctx, err, msg,
				"unable to access resource pool configuration")
			return nil
		}

		// Make sure we explicitly choose a partition.  Use default if unspecified.
		partition := req.ResourcePool
		if partition == "" {
			partition = m.selectDefaultPartition(slotType)
		}

		tresSupported := m.config.TresSupported
		gresSupported := m.config.GresSupported
		if m.config.TresSupported && !m.config.GresSupported {
			ctx.Log().Warnf("tres_supported: true cannot be used when " +
				"gres_supported: false is specified. Use tres_supported: false instead.")
			tresSupported = false
		}

		// Create the manifest that will be ultimately sent to the launcher.
		manifest, impersonatedUser, payloadName, err := msg.Spec.ToDispatcherManifest(
			m.config.MasterHost, m.config.MasterPort, m.masterTLSConfig.CertificateName,
			req.SlotsNeeded, slotType, partition, tresSupported, gresSupported,
			m.config.LauncherContainerRunType, m.wlmType == pbsSchedulerType)
		if err != nil {
			sendResourceStateChangedErrorResponse(ctx, err, msg,
				"unable to create the Slurm launcher manifest")
			return nil
		}

		if impersonatedUser == "root" {
			sendResourceStateChangedErrorResponse(ctx,
				fmt.Errorf(
					"You are logged in as Determined user '%s', however the user ID on the "+
						"target HPC cluster for this user has either not been configured, or has "+
						"been set to the "+
						"disallowed value of 'root'. In either case, as a determined administrator, "+
						"use the command 'det user link-with-agent-user' to specify how jobs for "+
						"Determined user '%s' are to be launched on your HPC cluster.",
					msg.Spec.Owner.Username, msg.Spec.Owner.Username),
				msg, "")
			return nil
		}

		dispatchID, err := m.sendManifestToDispatcher(ctx, manifest, impersonatedUser)
		if err != nil {
			sendResourceStateChangedErrorResponse(ctx, err, msg,
				"unable to create Slurm job")
			return nil
		}

		ctx.Log().Info(fmt.Sprintf("DispatchID is %s", dispatchID))
		m.dispatchIDToAllocationID[dispatchID] = req.AllocationID
		if err := db.InsertDispatch(context.TODO(), &db.Dispatch{
			DispatchID:       dispatchID,
			ResourceID:       msg.ResourcesID,
			AllocationID:     req.AllocationID,
			ImpersonatedUser: impersonatedUser,
		}); err != nil {
			ctx.Log().WithError(err).Errorf("failed to persist dispatch: %v", dispatchID)
		}
		m.jobWatcher.monitorJob(impersonatedUser, dispatchID, payloadName)
		return nil

	case sproto.PendingPreemption:
		ctx.Log().Info(fmt.Sprintf("PendingPreemption of %s.  Terminating.", msg.AllocationID))
		allocReq, ok := m.reqList.GetTaskByID(msg.AllocationID)
		if ok {
			ctx.Tell(allocReq.AllocationRef, sproto.ReleaseResources{ForcePreemption: true})
		} else {
			ctx.Log().Error(fmt.Sprintf("unable to find Allocation actor for AllocationID %s",
				msg.AllocationID))
		}

	case KillDispatcherResources:

		ctx.Log().Debug(fmt.Sprintf("Received request to terminate jobs associated with AllocationID %s",
			msg.AllocationID))

		// Find the Dispatch IDs associated with the allocation ID. We'll need the
		// Dispatch ID to cancel the job on the launcher side.
		dispatches, err := db.ListDispatchesByAllocationID(context.TODO(), msg.AllocationID)
		if err != nil {
			ctx.Log().WithError(err).Errorf(
				"Failed to retrieve the DispatchIDs associated with AllocationID %s",
				msg.AllocationID)
			return nil
		}

		ctx.Log().Debug(fmt.Sprintf("Found %d jobs associated with AllocationID %s",
			len(dispatches), msg.AllocationID))

		for _, dispatch := range dispatches {
			dispatchID := dispatch.DispatchID
			impersonatedUser := dispatch.ImpersonatedUser

			ctx.Log().Info(fmt.Sprintf("Terminating job with DispatchID %s initiated by %s",
				dispatchID, impersonatedUser))

			// Terminate and cleanup, on failure leave Dispatch in DB for later retry
			if m.terminateDispatcherJob(ctx, dispatchID, impersonatedUser) {
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
					ctx.Log().Debug(
						fmt.Sprintf(
							"Not removing dispatch environment for dispatchID '%s' because job is being monitored",
							dispatchID))
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
			}
		}

	case DispatchStateChange:
		log := ctx.Log().WithField("dispatch-id", msg.DispatchID)
		allocationID, ok := m.dispatchIDToAllocationID[msg.DispatchID]
		if !ok {
			log.Warnf("received DispatchStateChange for unknown dispatch %s", msg.DispatchID)
			return nil
		}

		task, ok := m.reqList.GetTaskByID(allocationID)
		if !ok {
			log.Warnf("received DispatchStateChange for dispatch unknown to task list: %s", allocationID)
			return nil
		}

		alloc := m.reqList.GetAllocations(task.AllocationRef)
		if len(alloc.Resources) != 1 {
			log.Warnf("allocation has malformed resources: %v", alloc)
			return nil
		}
		r := maps.Values(alloc.Resources)[0]
		rID := r.Summary().ResourcesID

		task.State = schedulingStateFromDispatchState(msg.State)
		ctx.Tell(task.AllocationRef, sproto.ResourcesStateChanged{
			ResourcesID:      rID,
			ResourcesState:   resourcesStateFromDispatchState(msg.State),
			ResourcesStarted: &sproto.ResourcesStarted{},
		})

	case DispatchExited:
		log := ctx.Log().WithField("dispatch-id", msg.DispatchID)
		allocationID, ok := m.dispatchIDToAllocationID[msg.DispatchID]
		if !ok {
			log.Warnf("received DispatchExited for unknown dispatch %s", msg.DispatchID)
			return nil
		}

		task, ok := m.reqList.GetTaskByID(allocationID)
		if !ok {
			log.Warnf("received DispatchExited for dispatch unknown to task list: %s", allocationID)
			return nil
		}

		alloc := m.reqList.GetAllocations(task.AllocationRef)
		if len(alloc.Resources) != 1 {
			log.Warnf("allocation has malformed resources: %v", alloc)
			return nil
		}
		r := maps.Values(alloc.Resources)[0]
		rID := r.Summary().ResourcesID

		stopped := sproto.ResourcesStopped{}
		if msg.ExitCode > 0 {
			stopped.Failure = sproto.NewResourcesFailure(
				sproto.TaskError,
				msg.Message,
				ptrs.Ptr(sproto.ExitCode(msg.ExitCode)),
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
		ctx.Log().Debug(fmt.Sprintf("Found %d jobs associated with AllocationID %s",
			len(dispatches), allocationID))

		// Cleanup all the dispatcher environments associated with current allocation
		for _, dispatch := range dispatches {
			dispatchID := dispatch.DispatchID
			impersonatedUser := dispatch.ImpersonatedUser

			if ctx.Log().Logger.Level < logrus.DebugLevel {
				ctx.Log().Info(fmt.Sprintf(
					"Deleting dispatcher environment for job with DispatchID %s initiated by %s",
					dispatchID, impersonatedUser))

				// Cleanup the dispatcher environment
				m.removeDispatchEnvironment(ctx, impersonatedUser, dispatchID)
			}
		}

		// Remove the dispatch from mapping tables.
		delete(m.dispatchIDToAllocationID, msg.DispatchID)

	case sproto.SetGroupMaxSlots:
		m.getOrCreateGroup(ctx, msg.Handler).maxSlots = msg.MaxSlots

	case groupActorStopped:
		delete(m.slotsUsedPerGroup, m.groups[msg.Ref])
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
	impersonatedUser string, dispatchID string) {
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
				Stats:        jobStatsByPool(m.reqList, resourcePool),
				ResourcePool: resourcePool,
			})
		}
		ctx.Respond(resp)

	case sproto.GetJobQStats:
		ctx.Log().Debugf("GetJobQStats for resource pool %s", msg.ResourcePool)
		// TODO(HAL-2863): Fill this in for the given pool as discerned from the slurm resources
		// info job.
		ctx.Respond(jobStats(m.reqList))

	case sproto.SetGroupWeight, sproto.SetGroupPriority, sproto.MoveJob:
		// TODO(HAL-2863): We may not be able to support these specific actions, but how we
		// let people interact with the job queue in dispatcher/slurm world.
		// ctx.Respond(fmt.Errorf("modifying job positions is not yet supported in slurm"))
		if ctx.ExpectingResponse() {
			ctx.Respond(ErrUnsupported(fmt.Sprintf("%T unsupported in the dispatcher RM", msg)))
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
		result = append(result, &pool)
	}
	return result, nil
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
	if slotType := m.config.ResolveSlotType(partition); slotType != nil {
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

// fetchHpcResourceDetails retrieves the details about HPC Resources.
// This function uses HPC Resources manifest to retrieve the required details.
// This function performs the following steps:
// 	1. Launch the manifest.
// 	2. Read the log file with details on HPC resources.
// 	3. Parse and load the details into a predefined struct - HpcResourceDetails
// 	4. Terminate the manifest.
// Returns struct with HPC resource details - HpcResourceDetails.
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
		Execute()
	if err != nil {
		if r != nil && (r.StatusCode == http.StatusUnauthorized ||
			r.StatusCode == http.StatusForbidden) {
			ctx.Log().Errorf("Failed to communicate with launcher due to error: "+
				"{%v}. Reloaded the auth token file {%s}. If this error persists, restart "+
				"the launcher service followed by a restart of the determined-master service.",
				err, m.config.LauncherAuthFile)
			m.authToken = loadAuthToken(m.config)
			m.jobWatcher.ReloadAuthToken()
		} else {
			ctx.Log().Errorf("Failed to communicate with launcher due to error: "+
				"{%v}, response: {%v}. Verify that the launcher service is up and reachable."+
				" Try a restart the launcher service followed by a restart of the "+
				"determined-master service. ", err, r)
		}
		return
	}
	ctx.Log().Debug(fmt.Sprintf("Launched Manifest with DispatchID %s", dispatchInfo.GetDispatchId()))

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
		Execute()
	if err != nil {
		ctx.Log().WithError(err).Errorf("failed to retrieve HPC Resource details")
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
	ctx.Log().Infof("default resource pools are '%s', '%s'",
		m.resourceDetails.lastSample.DefaultComputePoolPartition,
		m.resourceDetails.lastSample.DefaultAuxPoolPartition)
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
	if m.terminateDispatcherJob(ctx, dispatchID, owner) {
		m.removeDispatchEnvironment(ctx, owner, dispatchID)
	}
}

// terminateDispatcherJob terminates the dispatcher job with the given ID.
// Return true to indicate if the DB dispatch should additionally be deleted.
func (m *dispatcherResourceManager) terminateDispatcherJob(ctx *actor.Context,
	dispatchID string, owner string,
) bool {
	if dispatchID == "" {
		ctx.Log().Warn("Missing dispatchID, so no environment clean-up")
		return false
	}
	var err error
	var response *http.Response
	if _, response, err = m.apiClient.RunningApi.TerminateRunning(m.authContext(ctx),
		owner, dispatchID).Force(true).Execute(); err != nil {
		if response == nil || response.StatusCode != 404 {
			ctx.Log().WithError(err).Errorf("Failed to terminate job with Dispatch ID %s",
				dispatchID)
			// We failed to delete, and not 404/notfound so leave in DB.
			return false
		}
	}
	ctx.Log().Debug(fmt.Sprintf("Terminated manifest with DispatchID %s", dispatchID))
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
	if response, err := m.apiClient.MonitoringApi.DeleteEnvironment(m.authContext(ctx),
		owner, dispatchID).Execute(); err != nil {
		if response == nil || response.StatusCode != 404 {
			ctx.Log().WithError(err).Errorf("Failed to remove environment for Dispatch ID %s",
				dispatchID)
			// We failed to delete, and not 404/notfound so leave in DB for later retry
			return
		}
	} else {
		ctx.Log().Debug(fmt.Sprintf("Deleted environment with DispatchID %s", dispatchID))
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
		Execute()
	if err != nil {
		httpStatus := ""
		if response != nil {
			// So we can show the HTTP status code, if available.
			httpStatus = fmt.Sprintf("(HTTP status %d)", response.StatusCode)
		}
		return "", errors.Wrapf(err, "LaunchApi.LaunchAsync() returned an error %s. "+
			"Verify that the launcher service is up and reachable. Try a restart the "+
			"launcher service followed by a restart of the determined-master service.", httpStatus)
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
		msg.Name = "Unnamed-Slurm-Job"
	}

	ctx.Log().Infof(
		"resources are requested by %s (Allocation ID: %s)",
		msg.AllocationRef.Address(), msg.AllocationID,
	)
	m.reqList.AddTask(&msg)
}

func (m *dispatcherResourceManager) jobQInfo(rp string) map[model.JobID]*sproto.RMJobInfo {
	var reqs []*sproto.AllocateRequest
	for it := m.reqList.iterator(); it.next(); {
		if it.value().ResourcePool == rp {
			reqs = append(reqs, it.value())
		}
	}
	return reduceToJobQInfo(reqs)
}

func (m *dispatcherResourceManager) receiveSetTaskName(
	ctx *actor.Context, msg sproto.SetAllocationName,
) {
	if task, found := m.reqList.GetAllocationByHandler(msg.AllocationRef); found {
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

		ctx.Log().Debug(fmt.Sprintf("Restore: Found %d jobs associated with AllocationID %s",
			len(dispatches), req.AllocationID))

		for _, dispatch := range dispatches {
			dispatchID = dispatch.DispatchID
			impersonatedUser = dispatch.ImpersonatedUser
			rID = dispatch.ResourceID
			break
		}
	}

	m.slotsUsedPerGroup[m.groups[req.Group]] += req.SlotsNeeded

	if len(rID) == 0 {
		rID = sproto.ResourcesID(uuid.NewString())
	}
	allocations := sproto.ResourceList{
		rID: &DispatcherResources{
			id:                     rID,
			req:                    req,
			rm:                     ctx.Self(),
			group:                  m.groups[req.Group],
			defaultRendezvousIface: m.config.ResolveRendezvousNetworkInterface(req.ResourcePool),
			defaultProxyIface:      m.config.ResolveProxyNetworkInterface(req.ResourcePool),
		},
	}

	assigned := sproto.ResourcesAllocated{ID: req.AllocationID, Resources: allocations}
	m.reqList.SetAllocationsRaw(req.AllocationRef, &assigned)
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

			m.dispatchIDToAllocationID[dispatchID] = req.AllocationID
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

	if req, ok := m.reqList.GetAllocationByHandler(handler); ok {
		if group := m.groups[handler]; group != nil {
			m.slotsUsedPerGroup[group] -= req.SlotsNeeded
		}
	}
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
	ctx.Log().Debug(fmt.Sprintf("Found %d Dispatches to release", len(dispatches)))
	for _, dispatch := range dispatches {
		ctx.Log().Debug(fmt.Sprintf("Queuing cleanup of AllocationID %s, DispatchID %s",
			dispatch.AllocationID, dispatch.DispatchID))
		ctx.Tell(handler, KillDispatcherResources{
			ResourcesID:  dispatch.ResourceID,
			AllocationID: dispatch.AllocationID,
		})
	}
}

func (m *dispatcherResourceManager) getOrCreateGroup(
	ctx *actor.Context,
	handler *actor.Ref,
) *group {
	if g, ok := m.groups[handler]; ok {
		return g
	}
	priority := config.KubernetesDefaultPriority
	g := &group{handler: handler, weight: 1, priority: &priority}
	m.groups[handler] = g
	m.slotsUsedPerGroup[g] = 0

	if ctx != nil && handler != nil { // ctx is nil only for testing purposes.
		actors.NotifyOnStop(ctx, handler, groupActorStopped{})
	}
	return g
}

func (m *dispatcherResourceManager) schedulePendingTasks(ctx *actor.Context) {
	for it := m.reqList.iterator(); it.next(); {
		req := it.value()
		group := m.groups[req.Group]
		assigned := m.reqList.GetAllocations(req.AllocationRef)
		if !assignmentIsScheduled(assigned) {
			if maxSlots := group.maxSlots; maxSlots != nil {
				if m.slotsUsedPerGroup[group]+req.SlotsNeeded > *maxSlots {
					continue
				}
			}
			m.assignResources(ctx, req)
		}
	}
}

type (
	// DispatcherResources information.
	DispatcherResources struct {
		id    sproto.ResourcesID
		req   *sproto.AllocateRequest
		rm    *actor.Ref
		group *group

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
		DispatchID string
		State      launcher.DispatchState
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
	spec.ResourcesConfig.SetPriority(r.group.priority)
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
	clientMetadata.SetName("DAI-Slurm-Resources")

	// Create & populate the manifest
	manifest := *launcher.NewManifest("v1", *clientMetadata)
	manifest.SetPayloads([]launcher.Payload{*payload})

	return &manifest
}

// If an auth_file was specified, load the content and return it to enable authorization
//  with the launcher.  If the auth_file is configured, but does not exist we panic.
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
func resourcesStateFromDispatchState(state launcher.DispatchState) sproto.ResourcesState {
	switch state {
	case launcher.PENDING:
		return sproto.Starting
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
