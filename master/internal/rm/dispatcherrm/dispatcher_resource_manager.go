package dispatcherrm

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/determined-ai/determined/master/internal/api/apiutils"
	"github.com/determined-ai/determined/master/internal/rm/rmevents"

	"github.com/google/uuid"
	echoV4 "github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	launcher "github.hpe.com/hpe/hpc-ard-launcher-go/launcher"
	"golang.org/x/exp/maps"
	"google.golang.org/protobuf/proto"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/rm/rmerrors"
	"github.com/determined-ai/determined/master/internal/rm/rmutils"
	"github.com/determined-ai/determined/master/internal/rm/tasklist"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/aproto"
	"github.com/determined-ai/determined/master/pkg/command"
	"github.com/determined-ai/determined/master/pkg/device"
	"github.com/determined-ai/determined/master/pkg/logger"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/set"
	"github.com/determined-ai/determined/master/pkg/syncx/mapx"
	"github.com/determined-ai/determined/master/pkg/syncx/orderedmapx"
	"github.com/determined-ai/determined/master/pkg/tasks"
	"github.com/determined-ai/determined/proto/pkg/agentv1"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/containerv1"
	"github.com/determined-ai/determined/proto/pkg/devicev1"
	"github.com/determined-ai/determined/proto/pkg/jobv1"
	"github.com/determined-ai/determined/proto/pkg/resourcepoolv1"
)

const (
	slurmSchedulerType    wlmType = "slurm"
	pbsSchedulerType      wlmType = "pbs"
	slurmResourcesCarrier         = "com.cray.analytics.capsules.carriers.hpc.slurm.SlurmResources"
	pbsResourcesCarrier           = "com.cray.analytics.capsules.carriers.hpc.pbs.PbsResources"
	root                          = "root"
	// How frequently to cleanup terminated dispatches when in debug mode.
	terminatedDispatchCleanupInterval = 18 * time.Hour
)

// The launcher can only run up to 8 concurrent async launch threads. It will
// queue anything after that.  Therefore, set the maximum number of job launch
// goroutines to 8 to match the number of concurrent async launch threads that
// the launcher is able to handle.  We don't want any launch requests sitting
// in the launcher's queue, because the request contains a lot of data that's
// needed for the experiment, so we don't want to needlessly consume a lot of
// memory.
const maxJobLaunchGoRoutines = 8

// Number of worker goroutines that monitor the job cancel queue for job
// cancelation requests.
const numJobCancelWorkers = 8

// Keeps track of how many times "schedulePendingTasks()" was called.
var numTimesScheduledPendingTasksCalled uint64

var errNotSupportedOnHpcCluster = fmt.Errorf("%w on HPC clusters", rmerrors.ErrNotSupported)

type wlmType string

// actionCoolDown is the rate limit for queue submission.
const actionCoolDown = 500 * time.Millisecond

// DispatcherResourceManager manages the lifecycle of dispatcher resources.
//
// "jobCancelQueue" is a FIFO queue where job cancelation requests are placed
// waiting for the "jobCancelQueueWorker()" to remove it from the queue and
// send it to "stopLauncherJob()" so that the job termination request can be
// sent to the launcher.
//
// "inflightCancelations" is a list of allocation IDs for job cancelations
// that are in progress. That is, "stopLauncherJob()" is running for that
// allocation ID. The "stopLauncherJob()" function will add the allocation ID
// to the list upon entry and remove it from the list upon exit.
type DispatcherResourceManager struct {
	// system dependencies
	syslog    *logrus.Entry
	db        *db.PgDB
	apiClient *launcherAPIClient

	// static configuration.
	wlmType         wlmType
	rmConfig        *config.DispatcherResourceManagerConfig
	poolConfig      []config.ResourcePoolConfig
	poolProviderMap map[string][]string
	masterTLSConfig model.TLSClientConfig
	loggingConfig   model.LoggingConfig

	// mutable state. access must occur under lock or it must be thread-safe already (and then
	// thinking about critical sections for logical consistency is still... critical).
	mu                   sync.Mutex
	reqList              *tasklist.TaskList
	groups               map[model.JobID]*tasklist.Group
	dispatchIDToHPCJobID *mapx.Map[string, string]
	scheduledLaunches    mapx.Map[model.AllocationID, struct{}]
	inflightCancelations mapx.Map[model.AllocationID, struct{}]
	jobCancelQueue       *orderedmapx.Map[string, KillDispatcherResources]

	// caches.
	hpcDetailsCache *hpcResourceDetailsCache

	// db state.
	dbState dispatcherState

	// subsystems.
	jobWatcher *launcherMonitor
}

// New returns a new dispatcher resource manager.
func New(
	db *db.PgDB,
	echo *echoV4.Echo,
	cfg *config.ResourceConfig,
	opts *aproto.MasterSetAgentOptions,
	cert *tls.Certificate,
) *DispatcherResourceManager {
	var wlm wlmType
	var rmCfg *config.DispatcherResourceManagerConfig
	if cfg.ResourceManager.DispatcherRM != nil {
		wlm = slurmSchedulerType
		rmCfg = cfg.ResourceManager.DispatcherRM
	} else {
		wlm = pbsSchedulerType
		rmCfg = cfg.ResourceManager.PbsRM
	}

	tlsConfig, err := model.MakeTLSConfig(cert)
	if err != nil {
		panic(errors.Wrap(err, "failed to set up TLS config"))
	}

	apiClient, err := newLauncherAPIClient(rmCfg)
	if err != nil {
		// TODO(Brad): Don't panic like this...
		panic(fmt.Errorf("building dispatcherrm: %w", err))
	}

	dispatchIDtoHPCJobID := mapx.New[string, string]()
	monitorEvents := make(chan launcherMonitorEvent, 64)
	watcher := newDispatchWatcher(apiClient, &dispatchIDtoHPCJobID, monitorEvents)

	dbState, err := getDispatcherState(context.TODO())
	if err != nil {
		panic(errors.Wrap(err, "failed to create state for dispatcher resource manager"))
	}
	m := &DispatcherResourceManager{
		syslog:    logrus.WithField("component", "dispatcherrm"),
		db:        db,
		apiClient: apiClient,

		wlmType:         wlm,
		rmConfig:        rmCfg,
		poolConfig:      cfg.ResourcePools,
		poolProviderMap: makeProvidedPoolsMap(cfg.ResourcePools),
		masterTLSConfig: tlsConfig,
		loggingConfig:   opts.LoggingOptions,

		reqList:              tasklist.New(),
		groups:               make(map[model.JobID]*tasklist.Group),
		dispatchIDToHPCJobID: &dispatchIDtoHPCJobID,
		scheduledLaunches:    mapx.New[model.AllocationID, struct{}](),
		inflightCancelations: mapx.New[model.AllocationID, struct{}](),
		jobCancelQueue:       orderedmapx.New[string, KillDispatcherResources](),

		hpcDetailsCache: newHpcResourceDetailsCache(rmCfg, apiClient),

		dbState: *dbState,

		jobWatcher: watcher,
	}

	m.syslog.Info("starting dispatcher resource manager")
	if err := checkVersionNow(context.TODO(), m.syslog, m.apiClient); err != nil {
		log.Fatal(err)
	}

	go m.killAllInactiveDispatches()
	go gcOrphanedDispatches(context.TODO(), m.syslog, m.apiClient)
	go m.jobWatcher.watch()
	go m.handleLauncherMonitorEvents(monitorEvents)

	m.startJobCancelWorkers(numJobCancelWorkers)

	m.hpcDetailsCache.wait()

	go m.periodicallySchedulePendingTasks()

	return m
}

// Allocate adds a task to the queue to be allocated.
func (m *DispatcherResourceManager) Allocate(
	msg sproto.AllocateRequest,
) (*sproto.ResourcesSubscription, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	sub := rmevents.Subscribe(msg.AllocationID)
	m.addTask(msg)
	return sub, nil
}

// DeleteJob delete resources associated with a job from the launcher.
// Note to developers: this function doesn't acquire a lock and, ideally, we won't make it.
func (m *DispatcherResourceManager) DeleteJob(
	msg sproto.DeleteJob,
) (sproto.DeleteJobResponse, error) {
	// Under normal conditions dispatches are removed on termination of the job
	// This path allows the cleanup of dispatches associated with a job under
	// exceptional conditions (debug mode, crashes, etc).
	m.syslog.WithField("job-id", msg.JobID).Info("delete job")

	dispatches, err := db.ListDispatchesByJobID(context.TODO(), string(msg.JobID))
	if err != nil {
		m.syslog.WithField("job-id", msg.JobID).WithError(err).Error(
			"failed to retrieve the dispatches associated with job")
		return sproto.DeleteJobResponseOf(err), nil
	}
	for _, dispatch := range dispatches {
		m.syslog.
			WithField("job-id", msg.JobID).
			WithField("dispatch-id", dispatch.DispatchID).
			Debug("found dispatch associated with job")
		go m.removeDispatchEnvironment(dispatch.ImpersonatedUser, dispatch.DispatchID)
	}
	m.syslog.WithField("job-id", msg.JobID).Debug("delete job successful")
	return sproto.EmptyDeleteJobResponse(), nil
}

// ExternalPreemptionPending notifies a task of a preemption from the underlying resource manager.
func (m *DispatcherResourceManager) ExternalPreemptionPending(
	msg sproto.PendingPreemption,
) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.syslog.WithField("allocation-id", msg.AllocationID).
		Info("pending preemption of allocation, terminating")
	allocReq, ok := m.reqList.TaskByID(msg.AllocationID)
	if ok {
		rmevents.Publish(allocReq.AllocationID, &sproto.ReleaseResources{
			Reason:          "preempted by the scheduler",
			ForcePreemption: true,
		})
	} else {
		m.syslog.WithField("allocation-id", msg.AllocationID).
			Errorf("unable to find allocation actor for allocation")
	}
	return nil
}

// GetAgents implements rm.ResourceManager.
// Note to developers: this function must not acquire locks, since it is polled to saturate
// the UI.
func (m *DispatcherResourceManager) GetAgents(
	msg *apiv1.GetAgentsRequest,
) (*apiv1.GetAgentsResponse, error) {
	hpcDetails, err := m.hpcDetailsCache.load()
	if err != nil {
		return nil, err
	}

	var resp apiv1.GetAgentsResponse
	for _, node := range hpcDetails.Nodes {
		resp.Agents = append(resp.Agents, m.hpcNodeToAgent(node))
	}
	return &resp, nil
}

// GetAllocationSummaries implements rm.ResourceManager.
func (m *DispatcherResourceManager) GetAllocationSummaries(
	msg sproto.GetAllocationSummaries,
) (map[model.AllocationID]sproto.AllocationSummary, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.reqList.TaskSummaries(m.groups, string(m.wlmType)), nil
}

// GetAllocationSummary implements rm.ResourceManager.
func (m *DispatcherResourceManager) GetAllocationSummary(
	msg sproto.GetAllocationSummary,
) (*sproto.AllocationSummary, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.reqList.TaskSummary(msg.ID, m.groups, string(m.wlmType)), nil
}

// GetDefaultAuxResourcePool implements rm.ResourceManager.
func (m *DispatcherResourceManager) GetDefaultAuxResourcePool(
	msg sproto.GetDefaultAuxResourcePoolRequest,
) (sproto.GetDefaultAuxResourcePoolResponse, error) {
	hpcDetails, err := m.hpcDetailsCache.load()
	if err != nil {
		return sproto.GetDefaultAuxResourcePoolResponse{}, err
	}
	return sproto.GetDefaultAuxResourcePoolResponse{
		PoolName: hpcDetails.DefaultAuxPoolPartition,
	}, nil
}

// GetDefaultComputeResourcePool implements rm.ResourceManager.
func (m *DispatcherResourceManager) GetDefaultComputeResourcePool(
	msg sproto.GetDefaultComputeResourcePoolRequest,
) (sproto.GetDefaultComputeResourcePoolResponse, error) {
	hpcDetails, err := m.hpcDetailsCache.load()
	if err != nil {
		return sproto.GetDefaultComputeResourcePoolResponse{}, err
	}
	return sproto.GetDefaultComputeResourcePoolResponse{
		PoolName: hpcDetails.DefaultComputePoolPartition,
	}, nil
}

// GetExternalJobs implements rm.ResourceManager.
func (m *DispatcherResourceManager) GetExternalJobs(
	msg sproto.GetExternalJobs,
) ([]*jobv1.Job, error) {
	return m.jobWatcher.fetchExternalJobs(msg.ResourcePool), nil
}

// GetJobQ implements rm.ResourceManager.
func (m *DispatcherResourceManager) GetJobQ(
	msg sproto.GetJobQ,
) (map[model.JobID]*sproto.RMJobInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(strings.TrimSpace(msg.ResourcePool)) == 0 {
		msg.ResourcePool = m.hpcDetailsCache.lastSample.Load().DefaultComputePoolPartition
		m.syslog.WithField("resource-pool", msg.ResourcePool).
			Trace("no resource pool name provided, selected the default compute pool")
	}
	var reqs []*sproto.AllocateRequest
	for it := m.reqList.Iterator(); it.Next(); {
		if it.Value().ResourcePool == msg.ResourcePool {
			reqs = append(reqs, it.Value())
		}
	}
	return tasklist.ReduceToJobQInfo(reqs), nil
}

// GetJobQueueStatsRequest implements rm.ResourceManager.
// This and other job queue saturation points should be refactored to not take locks.
func (m *DispatcherResourceManager) GetJobQueueStatsRequest(
	msg *apiv1.GetJobQueueStatsRequest,
) (*apiv1.GetJobQueueStatsResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.syslog.Tracef("GetJobQueueStatsRequest, pool count %d", len(msg.ResourcePools))

	var resp apiv1.GetJobQueueStatsResponse
	// If no list of resource pools has been specified, return data for all pools.
	if len(msg.ResourcePools) == 0 {
		resourcePools, err := m.GetResourcePools(&apiv1.GetResourcePoolsRequest{})
		if err != nil {
			return nil, err
		}

		for _, p := range resourcePools.ResourcePools {
			msg.ResourcePools = append(msg.ResourcePools, p.Name)
		}
	}
	// Compute RPQueueStat results for each resource pool
	for _, resourcePool := range msg.ResourcePools {
		resp.Results = append(resp.Results, &apiv1.RPQueueStat{
			Stats:        m.getCombinedJobStats(resourcePool),
			ResourcePool: resourcePool,
		})
	}
	return &resp, nil
}

// getCombinedJobStats returns job queue statistics for a given resource pool for both
// DeterminedAI and external jobs. If the resource pool is an empty string then job queue
// statistics for all resource pools is returned.
func (m *DispatcherResourceManager) getCombinedJobStats(resourcePool string) *jobv1.QueueStats {
	var determinedJobStats *jobv1.QueueStats
	if resourcePool != "" {
		determinedJobStats = tasklist.JobStatsByPool(m.reqList, resourcePool)
	} else {
		determinedJobStats = tasklist.JobStats(m.reqList)
	}
	externalJobStats := m.jobWatcher.getExternalJobQStats(resourcePool)
	combinedJobStats := &jobv1.QueueStats{}
	combinedJobStats.QueuedCount = determinedJobStats.QueuedCount + externalJobStats.QueuedCount
	combinedJobStats.ScheduledCount = determinedJobStats.ScheduledCount + externalJobStats.ScheduledCount
	return combinedJobStats
}

// GetResourcePools retrieves details regarding hpc resources of the underlying system.
// Note to developers: this function must not acquire locks, since it is polled to saturate
// the UI.
func (m *DispatcherResourceManager) GetResourcePools(
	msg *apiv1.GetResourcePoolsRequest,
) (*apiv1.GetResourcePoolsResponse, error) {
	hpcDetails, err := m.hpcDetailsCache.load()
	if err != nil {
		return nil, err
	}

	wlmName, schedulerType, fittingPolicy := m.getWlmResources()
	var result []*resourcepoolv1.ResourcePool
	poolNameMap := make(map[string]*resourcepoolv1.ResourcePool)

	for _, v := range hpcDetails.Partitions {
		slotType := m.resolveSlotType(hpcDetails, v.PartitionName)
		slotsAvailable := int32(v.TotalGpuSlots)
		slotsUsed := int32(v.TotalGpuSlots - v.TotalAvailableGpuSlots)
		if slotType == device.CPU {
			slotsAvailable = int32(v.TotalCPUSlots)
			slotsUsed = int32(v.TotalCPUSlots - v.TotalAvailableCPUSlots)
		}
		slotsPerAgent := 0

		// "TotalAvailableNodes" represents the nodes that are in service,
		// which may be equal or lesser than "TotalNodes". For example, with
		// the Slurm Workload Manager, nodes that are in service will have
		// a state of "idle", "mix", or "alloc". Any other state means that
		// no jobs will be scheduled on those nodes.  Therefore, since
		// "slotsAvailable" represents the combined number of slots for the
		// nodes that are in service, we need to divide by the number of
		// nodes that are in service to get an accurate number of slots per
		// agent.
		totalNodesInService := v.TotalAllocatedNodes + v.TotalAvailableNodes

		if totalNodesInService != 0 {
			slotsPerAgent = int(slotsAvailable) / totalNodesInService
		}

		description := wlmName + "-managed pool of resources"
		// Due to viper.MergeConfigMap, map keys in configurations lose case. We match case
		// insensitive here to handle partitions with upper case characters, at the cost of
		// incorrectly matching when names are only equal when comparing case-insensitive.
		if overrides, ok := m.rmConfig.PartitionOverrides[strings.ToLower(v.PartitionName)]; ok {
			description = overrides.Description
		}

		pool := resourcepoolv1.ResourcePool{
			Name:                         v.PartitionName,
			Description:                  description,
			Type:                         resourcepoolv1.ResourcePoolType_RESOURCE_POOL_TYPE_STATIC,
			NumAgents:                    int32(totalNodesInService),
			SlotType:                     slotType.Proto(),
			SlotsAvailable:               slotsAvailable,
			SlotsUsed:                    slotsUsed,
			AuxContainerCapacity:         int32(v.TotalCPUSlots),
			AuxContainersRunning:         int32(v.TotalCPUSlots - v.TotalAvailableCPUSlots),
			DefaultComputePool:           v.PartitionName == m.getDefaultPoolName(hpcDetails, false),
			DefaultAuxPool:               v.PartitionName == m.getDefaultPoolName(hpcDetails, true),
			Preemptible:                  true,
			MinAgents:                    int32(v.TotalNodes),
			MaxAgents:                    int32(v.TotalNodes),
			SlotsPerAgent:                int32(slotsPerAgent),
			AuxContainerCapacityPerAgent: 0,
			SchedulerType:                schedulerType,
			SchedulerFittingPolicy:       fittingPolicy,
			Location:                     "",
			ImageId:                      "",
			InstanceType:                 "",
			Details:                      &resourcepoolv1.ResourcePoolDetail{},
			Accelerator:                  v.Accelerator,
		}
		poolNameMap[pool.Name] = &pool
		result = append(result, &pool)
	}
	result = append(result, m.getLauncherProvidedPools(hpcDetails, poolNameMap)...)

	return &apiv1.GetResourcePoolsResponse{ResourcePools: result}, nil
}

// getLauncherProvidedPools provides data for any launcher-provided resource pools
// from the master configuration.
// Note to the developer: this must not acquire a lock. Possibly changing this from a method to a
// function makes this more obvious.
func (m *DispatcherResourceManager) getLauncherProvidedPools(
	hpcDetails *hpcResources,
	poolNameMap map[string]*resourcepoolv1.ResourcePool,
) []*resourcepoolv1.ResourcePool {
	var result []*resourcepoolv1.ResourcePool
	for _, pool := range m.poolConfig {
		if isValidProvider(pool) {
			basePoolName := pool.Provider.HPC.Partition
			basePool, found := poolNameMap[basePoolName]
			if !found {
				m.syslog.Errorf("resource pool %s specifies provider.partition '%s' that does not exist",
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
			launcherPoolResult.DefaultComputePool = pool.PoolName == m.getDefaultPoolName(hpcDetails, false)
			launcherPoolResult.DefaultAuxPool = pool.PoolName == m.getDefaultPoolName(hpcDetails, true)
			result = append(result, launcherPoolResult)
		}
	}
	return result
}

// MoveJob implements rm.ResourceManager.
func (*DispatcherResourceManager) MoveJob(sproto.MoveJob) error {
	// TODO(HAL-2863): We may not be able to support these specific actions, but how we
	// let people interact with the job queue in dispatcher/slurm world.
	// ctx.Respond(fmt.Errorf("modifying job positions is not yet supported in slurm"))
	return rmerrors.UnsupportedError("move job unsupported in the dispatcher RM")
}

// RecoverJobPosition implements rm.ResourceManager.
func (m *DispatcherResourceManager) RecoverJobPosition(sproto.RecoverJobPosition) {
	m.syslog.Warn("move job unsupported in the dispatcher RM")
}

// Release implements rm.ResourceManager.
func (m *DispatcherResourceManager) Release(msg sproto.ResourcesReleased) {
	if msg.ResourcesID != nil {
		// This optimization does not apply to dispatcher RM, since slurm or pbs is the actual RM.
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Remove any scheduled launch for the job associated with the
	// allocation ID.  Typically, this would be called at the end of
	// "startLauncherJob()". However, if the experiment is canceled
	// immediately, it is possible for "resourcesReleased()" to be called
	// without ever getting a "StartDispatcherResources" message.
	// Therefore, we make sure we remove the allocation ID from the
	// "scheduledLaunches" map here, since the size of the map determines
	// if "schedulePendingTasks()" will assign resources to another job
	// (i.e., send another StartDispatcherResources message). It should
	// also be noted that "resourcesReleased()" may get called multiple
	// times, but there's no harm in calling "deleteScheduledLaunch()"
	// more than once.
	m.scheduledLaunches.Delete(msg.AllocationID)

	req := m.reqList.RemoveTaskByID(msg.AllocationID)
	if req == nil {
		m.syslog.
			WithField("allocation-id", msg.AllocationID).
			WithField("scheduled-launches", m.scheduledLaunches.Len()).
			Info("resources were already released")
	} else {
		m.syslog.
			WithField("name", req.Name).
			WithField("allocation-id", msg.AllocationID).
			WithField("scheduled-launches", m.scheduledLaunches.Len()).
			Info("resources are released")
	}
	rmevents.Publish(msg.AllocationID, sproto.ResourcesReleasedEvent{})
}

// SetAllocationName implements rm.ResourceManager.
func (m *DispatcherResourceManager) SetAllocationName(
	msg sproto.SetAllocationName,
) {
	m.mu.Lock()
	defer m.mu.Unlock()

	task, ok := m.reqList.TaskByID(msg.AllocationID)
	if !ok {
		return
	}
	task.Name = msg.Name
}

// SetGroupMaxSlots implements rm.ResourceManager.
func (m *DispatcherResourceManager) SetGroupMaxSlots(
	msg sproto.SetGroupMaxSlots,
) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.getOrCreateGroup(msg.JobID).MaxSlots = msg.MaxSlots
}

// SetGroupPriority implements rm.ResourceManager.
func (*DispatcherResourceManager) SetGroupPriority(sproto.SetGroupPriority) error {
	// TODO(HAL-2863)
	return rmerrors.UnsupportedError("set group priority unsupported in the dispatcher RM")
}

// SetGroupWeight implements rm.ResourceManager.
func (*DispatcherResourceManager) SetGroupWeight(sproto.SetGroupWeight) error {
	// TODO(HAL-2863)
	return rmerrors.UnsupportedError("set group weight unsupported in the dispatcher RM")
}

// ValidateCommandResources implements rm.ResourceManager.
func (*DispatcherResourceManager) ValidateCommandResources(
	sproto.ValidateCommandResourcesRequest,
) (sproto.ValidateCommandResourcesResponse, error) {
	// TODO(HAL-2862): Use inferred value here if possible.
	// fulfillable := m.config.MaxSlotsPerContainer >= msg.Slots
	return sproto.ValidateCommandResourcesResponse{Fulfillable: true}, nil
}

// ValidateResourcePoolAvailability implements rm.ResourceManager.
func (*DispatcherResourceManager) ValidateResourcePoolAvailability(
	*sproto.ValidateResourcePoolAvailabilityRequest,
) ([]command.LaunchWarning, error) {
	return nil, nil
}

// ValidateResources implements rm.ResourceManager.
func (*DispatcherResourceManager) ValidateResources(
	string, int, bool,
) error {
	return nil
}

// DisableAgent adds an agent to the exclude list when launching jobs.
// Note to developers: this function doesn't acquire a lock and, ideally, we won't make it.
func (m *DispatcherResourceManager) DisableAgent(
	msg *apiv1.DisableAgentRequest,
) (*apiv1.DisableAgentResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.wlmType == pbsSchedulerType {
		return nil, errors.New("disable agent is not supported for PBS")
	}

	agent, err := m.findAgent(msg.AgentId)
	if err != nil {
		return nil, err
	}
	if err := m.dbState.disableAgent(msg.AgentId); err != nil {
		return nil, err
	}
	agent.Enabled = false

	return &apiv1.DisableAgentResponse{Agent: agent}, nil
}

// EnableAgent removes an agent from the exclude list when launching jobs.
// Note to developers: this function doesn't acquire a lock and, ideally, we won't make it.
func (m *DispatcherResourceManager) EnableAgent(
	msg *apiv1.EnableAgentRequest,
) (*apiv1.EnableAgentResponse, error) {
	if m.wlmType == pbsSchedulerType {
		return nil, errors.New("enable agent is not supported for PBS")
	}

	agent, err := m.findAgent(msg.AgentId)
	if err != nil {
		return nil, err
	}

	if err := m.dbState.enableAgent(msg.AgentId); err != nil {
		return nil, err
	}
	agent.Enabled = true

	return &apiv1.EnableAgentResponse{Agent: agent}, nil
}

// GetAgent implements rm.ResourceManager.
// Note to developers: this function must not acquire locks, since it is called to saturate UIs.
func (m *DispatcherResourceManager) GetAgent(
	msg *apiv1.GetAgentRequest,
) (*apiv1.GetAgentResponse, error) {
	hpcDetails, err := m.hpcDetailsCache.load()
	if err != nil {
		return nil, err
	}

	for _, node := range hpcDetails.Nodes {
		if node.Name == msg.AgentId {
			return &apiv1.GetAgentResponse{Agent: m.hpcNodeToAgent(node)}, nil
		}
	}
	return nil, apiutils.ErrNotFound
}

// GetSlot is unsupported.
func (*DispatcherResourceManager) GetSlot(*apiv1.GetSlotRequest) (*apiv1.GetSlotResponse, error) {
	return nil, rmerrors.ErrNotSupported
}

// GetSlots is unsupported.
func (*DispatcherResourceManager) GetSlots(*apiv1.GetSlotsRequest) (*apiv1.GetSlotsResponse, error) {
	return nil, rmerrors.ErrNotSupported
}

// ResolveResourcePool returns the resolved slurm partition or an error if it doesn't exist or
// can't be resolved due to internal errors.
// Note to developers: this function doesn't acquire a lock and, ideally, we won't make it, since
// it is called a lot.
func (m *DispatcherResourceManager) ResolveResourcePool(
	name string, workspaceID, slots int,
) (string, error) {
	hpcDetails, err := m.hpcDetailsCache.load()
	if err != nil {
		return "", err
	}

	ctx := context.TODO()
	defaultComputePool, defaultAuxPool, err := db.GetDefaultPoolsForWorkspace(ctx, workspaceID)
	if err != nil {
		return "", err
	}

	// If the resource pool isn't set, fill in the default at creation time.
	if name == "" && slots == 0 {
		if defaultAuxPool == "" {
			name = hpcDetails.DefaultAuxPoolPartition
		} else {
			name = defaultAuxPool
		}
	}

	if name == "" && slots >= 0 {
		if defaultComputePool == "" {
			name = hpcDetails.DefaultComputePoolPartition
		} else {
			name = defaultComputePool
		}
	}

	resp, err := m.GetResourcePools(&apiv1.GetResourcePoolsRequest{})
	if err != nil {
		return "", err
	}

	poolNames, _, err := db.ReadRPsAvailableToWorkspace(
		ctx, int32(workspaceID), 0, -1, rmutils.ResourcePoolsToConfig(resp.ResourcePools))
	if err != nil {
		return "", err
	}
	found := false
	for _, poolName := range poolNames {
		if name == poolName {
			found = true
			break
		}
	}
	if !found {
		return "", fmt.Errorf(
			"resource pool %s does not exist or is not available to workspace id %d",
			name, workspaceID)
	}

	_, err = m.validateResourcePool(hpcDetails, name)
	if err != nil {
		return "", fmt.Errorf("validating resource pool: %w", err)
	}
	return name, nil
}

// ValidateResourcePool validates that the given resource pool exists.
// Note to developers: this function doesn't acquire a lock and, ideally, we won't make it, since
// it is called a lot.
func (m *DispatcherResourceManager) ValidateResourcePool(name string) error {
	hpcDetails, err := m.hpcDetailsCache.load()
	if err != nil {
		return err
	}

	_, err = m.validateResourcePool(hpcDetails, name)
	return err
}

func (m *DispatcherResourceManager) validateResourcePool(
	hpcDetails *hpcResources,
	name string,
) (string, error) {
	switch resp := m.hasSlurmPartition(hpcDetails, name); {
	case !resp.HasResourcePool && resp.ProvidingPartition != "":
		return "", fmt.Errorf(
			"resource pool %s is configured to use partition '%s' that does not exist "+
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
func (m *DispatcherResourceManager) IsReattachEnabled() bool {
	return true
}

// IsReattachableOnlyAfterStarted is always false for dispatcher-based job schedulers
// as the start_time is not set on our allocations.
func (m *DispatcherResourceManager) IsReattachableOnlyAfterStarted() bool {
	return false
}

// IsReattachEnabledForRP returns true for all resource pools.
func (m *DispatcherResourceManager) IsReattachEnabledForRP(rpName string) bool {
	return true
}

func (m *DispatcherResourceManager) handleLauncherMonitorEvents(evs <-chan launcherMonitorEvent) {
	for msg := range evs {
		switch msg := msg.(type) {
		case DispatchStateChange:
			m.DispatchStateChange(msg)
		case DispatchExited:
			m.handleDispatchExited(msg)
		case dispatchExpLogMessage:
			m.DispatchExpLogMessage(msg)
		}
	}
	m.syslog.Error("dispatcher monitor stopped unexpectedly")
}

func (m *DispatcherResourceManager) handleDispatchExited(msg DispatchExited) {
	// Perform any necessary accesses to the m.reqList directly in
	// the handler to avoid any synchronization issues.
	log := m.syslog.WithField("dispatch-id", msg.DispatchID)

	allocationID := m.getAllocationID(msg.DispatchID)

	// Job completed while it was sitting in the cancelation queue, so
	// remove it so that we don't send a request to the launcher to
	// terminate a job that already completed.
	if m.jobCancelQueue.Delete(string(allocationID)) {
		log.Info("job completed while still in cancelation queue, removed job from cancelation queue")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	task, ok := m.reqList.TaskByID(allocationID)
	if !ok {
		log.Warn("received DispatchExited for dispatch unknown to task list")
		return
	}

	alloc := m.reqList.Allocation(task.AllocationID)
	if len(alloc.Resources) != 1 {
		log.Warnf("allocation has malformed resources: %v", alloc)
		return
	}

	// Now preform the actual work asych to avoid blocking
	go m.dispatchExited(msg, task, alloc)
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

func (m *DispatcherResourceManager) getProvidingPartition(name string) string {
	for _, pool := range m.poolConfig {
		if isValidProvider(pool) && pool.PoolName == name {
			return pool.Provider.HPC.Partition
		}
	}
	return name
}

// jobCancelQueueWorker waits to be notified that a job cancelation request is
// in the queue, then calls "stopLauncherJob()" to cancel the job.
func (m *DispatcherResourceManager) jobCancelQueueWorker(workerID int) {
	// Loop forever.
	for {
		// Remove the next job cancelation request from the queue and send
		// it to the launcher. If the queue is empty, "GetAndDelete" will
		// wait for an element to be placed in the queue.
		msg, ok := m.jobCancelQueue.GetAndDelete()
		if ok {
			m.syslog.WithField("worker-id", workerID).
				WithField("allocation-id", msg.AllocationID).
				WithField("queue-size", m.jobCancelQueue.Length()).
				Debug("job cancel queue worker found request")
			m.stopLauncherJob(msg)
			continue
		}

		// Should never hit this case, but log a warning if we do.
		m.syslog.WithField("worker-id", workerID).
			Warn("job cancel queue worker did not find any requests")
	}
}

// startJobCancelWorkers starts up "numWorkers" goroutines which wait to be
// notified that a job cancelation request has been queued. The first worker
// to receive the notification will call "stopLauncherJob()" to cancel the
// job.
func (m *DispatcherResourceManager) startJobCancelWorkers(numWorkers int) {
	for i := 0; i < numWorkers; i++ {
		go m.jobCancelQueueWorker(i)
	}
}

// hasSlurmPartitionResponse is the response to HasResourcePoolRequest.
type hasSlurmPartitionResponse struct {
	HasResourcePool    bool
	ProvidingPartition string // Set for launcher-provided resource pools
	ValidationErrors   []error
}

// hasSlurmPartition computes a response to a resource pool validation request. The target may be
// either a HPC native partition/queue, or a launcher-provided pool. In the latter case we verify
// that the providing partition exists on the cluster.
func (m *DispatcherResourceManager) hasSlurmPartition(
	hpcDetails *hpcResources,
	poolName string,
) hasSlurmPartitionResponse {
	result := false
	providingPartition := ""
	var validationErrors []error
	result = partitionExists(poolName, hpcDetails.Partitions)
	if !result {
		for _, pool := range m.poolConfig {
			if pool.PoolName == poolName && isValidProvider(pool) {
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

// hpcNodeToAgent converts a hpcNodeDetails to an agentv1.Agent.
func (m *DispatcherResourceManager) hpcNodeToAgent(node hpcNodeDetails) *agentv1.Agent {
	agent := &agentv1.Agent{
		Id:             node.Name,
		RegisteredTime: nil,
		Slots:          map[string]*agentv1.Slot{},
		ResourcePools:  node.Partitions,
		Addresses:      node.Addresses,
		Enabled:        m.dbState.isAgentEnabled(node.Name),
		Draining:       node.Draining,
	}
	m.updateAgentWithAnyProvidedResourcePools(agent)
	if node.GpuCount == 0 {
		// Adds a slot ID (e.g., 0, 1, 2, ..., N) to the agent for every
		// CPU being used on the node. This is needed so that the
		// "Resource Pools" page on the Determined AI User Interface
		// correctly shows the "N/M CPU Slots Allocated".
		for i := 0; i < node.CPUCount; i++ {
			addSlotToAgent(
				agent, devicev1.Type_TYPE_CPU, node, i, i < node.CPUInUseCount)
		}
	} else {
		for i := 0; i < node.GpuCount; i++ {
			slotType := computeSlotType(node, m)
			addSlotToAgent(
				agent, slotType, node, i, i < node.GpuInUseCount) // [1:N] CUDA slots
		}
	}
	return agent
}

func (m *DispatcherResourceManager) updateAgentWithAnyProvidedResourcePools(
	agent *agentv1.Agent,
) {
	for _, poolName := range agent.ResourcePools {
		agent.ResourcePools = append(agent.ResourcePools, m.poolProviderMap[poolName]...)
	}
}

// computeSlotType computes an agent GPU slot type from the configuration data available.
// For nodes that are members of multiple partitions, take the first configured slot type found,
// falling back to CUDA if nothing found.
func computeSlotType(node hpcNodeDetails, m *DispatcherResourceManager) devicev1.Type {
	for _, partition := range node.Partitions {
		slotType := m.rmConfig.ResolveSlotTypeFromOverrides(partition)
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
		slot.Container = &containerv1.Container{Id: "dispatcherrm-inuse-slot-placeholder"}
		slot.Container.State = containerv1.State_STATE_RUNNING
	}
	agent.Slots[slotRef] = &slot
}

// StartDispatcherResources starts an async process to launch a task.
// Note to developers: If it acquires locks, it must be fast (no DB or API calls).
func (m *DispatcherResourceManager) StartDispatcherResources(msg StartDispatcherResources) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Perform any necessary actions on m.reqList before going async
	req, ok := m.reqList.TaskByID(msg.AllocationID)
	if !ok {
		m.sendResourceStateChangedErrorResponse(errors.New("no such task"), msg,
			"task not found in the task list")

		// no request to process, so bail
		return
	}

	// Start each launcher job in a goroutine to prevent incoming messages
	// from backing up, due to the main thread being busy handling one
	// message at a time. Adaptive searches may create many launcher jobs
	// for a single experiment, so we must allow the main thread to continue
	// handling incoming messages while the previous messages are still
	// being processed. The UI will become unresponsive if the messages
	// start backing up.
	go m.startLauncherJob(msg, req)
}

// KillDispatcherResources puts a kill request on the queue.
// Note to developers: If it acquires locks, it must be fast (no DB or API calls).
func (m *DispatcherResourceManager) KillDispatcherResources(msg KillDispatcherResources) {
	// Check if there is already a job cancelation inflight.
	if _, ok := m.inflightCancelations.Load(msg.AllocationID); ok {
		message := "Received request to cancel job, but job cancelation is already in progress"
		m.syslog.WithField("allocation-id", msg.AllocationID).Debug(message)
		m.DispatchExpLogMessage(dispatchExpLogMessage{
			DispatchID: string(msg.AllocationID),
			Message:    message,
		})
		return
	}

	// Put the job cancelation request in the queue. If there is already a
	// request queued, do not queue up a second one.  Simply log a message
	// both in the master log and the experiment log.
	if _, ok := m.jobCancelQueue.PutIfAbsent(string(msg.AllocationID), msg); !ok {
		message := "Received request to cancel job, but job cancelation request is already queued"
		m.syslog.WithField("allocation-id", msg.AllocationID).Debug(message)
		m.DispatchExpLogMessage(dispatchExpLogMessage{
			DispatchID: string(msg.AllocationID),
			Message:    message,
		})
		return
	}
}

// DispatchExpLogMessage publishes a log for the dispatch-associated task. It is called by the
// launcher monitor event handler.
func (m *DispatcherResourceManager) DispatchExpLogMessage(msg dispatchExpLogMessage) {
	m.mu.Lock()
	defer m.mu.Unlock()

	log := m.syslog.WithField("dispatch-id", msg.DispatchID)
	task := m.getAssociatedTask(log, msg.DispatchID)
	if task == nil {
		return
	}
	rmevents.Publish(task.AllocationID, &sproto.ContainerLog{AuxMessage: &msg.Message})
}

// DispatchStateChange records state changes and propagates them to allocations. It is called
// by the launcher monitor event handler.
// Note to developers: this function locks so don't make API or DB calls without optimization.
func (m *DispatcherResourceManager) DispatchStateChange(msg DispatchStateChange) {
	m.mu.Lock()
	defer m.mu.Unlock()

	log := m.syslog.WithField("dispatch-id", msg.DispatchID)
	task := m.getAssociatedTask(log, msg.DispatchID)
	if task == nil {
		return
	}

	alloc := m.reqList.Allocation(task.AllocationID)
	if len(alloc.Resources) != 1 {
		log.Warnf("allocation has malformed resources: %v", alloc)
		return
	}

	_, exist := m.dispatchIDToHPCJobID.Load(msg.DispatchID)
	if !exist && msg.HPCJobID != "" {
		hpcJobIDMsg := "HPC Job ID: " + msg.HPCJobID
		rmevents.Publish(task.AllocationID, &sproto.ContainerLog{AuxMessage: &hpcJobIDMsg})
		m.dispatchIDToHPCJobID.Store(msg.DispatchID, msg.HPCJobID)

		log.WithField("hpc-job-id", msg.HPCJobID).
			Debug("received HPC job ID for dispatch")
	}

	r := maps.Values(alloc.Resources)[0]
	rID := r.Summary().ResourcesID

	task.State = schedulingStateFromDispatchState(msg.State)
	rmevents.Publish(task.AllocationID, &sproto.ResourcesStateChanged{
		ResourcesID:      rID,
		ResourcesState:   resourcesStateFromDispatchState(msg.IsPullingImage, msg.State),
		ResourcesStarted: &sproto.ResourcesStarted{},
	})
}

// Utility method to convert a dispatchID to an allocationID
// Prior to 0.22.2 they were distinct values, so need to handle
// active dispatchIDs that started prior to 0.22.2 by looking up
// in the DB instead of just using the dispatchID as the allocationID.
func (m *DispatcherResourceManager) getAllocationID(
	dispatchID string,
) model.AllocationID {
	// For dispatches created before 0.22.2 the DispatchID may not
	// be the AllocationID, so look it up instead.
	allocationTask := m.getAssociatedTask(m.syslog, dispatchID)
	if allocationTask != nil {
		// We found the task so use the allocationID from it
		// in case it came from the DB.
		return allocationTask.AllocationID
	}
	return model.AllocationID(dispatchID)
}

func (m *DispatcherResourceManager) getAssociatedTask(
	log *logrus.Entry,
	dispatchID string,
) *sproto.AllocateRequest {
	allocationID := model.AllocationID(dispatchID)

	task, ok := m.reqList.TaskByID(allocationID)
	if !ok {
		// This is a corner-case due to the change to convert from using generated
		//  dispatchIDs, to re-using the AllocationID for that purpose.
		//  If there is an active dispatch across the upgrade we cannot look
		//  it up using the AllocationID.  Instead we have to use the DB
		//  table to map the dispatchID to AllocationID.   This code can
		//  be dropped when we no longer need to support upgrades from versions
		//  prior to 0.22.2-ee.
		dispatch, err := db.DispatchByID(context.TODO(), dispatchID)
		if err == nil {
			task, ok = m.reqList.TaskByID(dispatch.AllocationID)
			if ok {
				return task
			}
		}
		log.WithField("dispatch-id", dispatchID).Warn("received message for dispatch unknown to task list")
		return nil
	}
	return task
}

// Called only from DispatchExited event and always run via go routine.
func (m *DispatcherResourceManager) dispatchExited(
	msg DispatchExited,
	task *sproto.AllocateRequest,
	alloc *sproto.ResourcesAllocated,
) {
	log := m.syslog.WithField("dispatch-id", msg.DispatchID)
	r := maps.Values(alloc.Resources)[0]
	rID := r.Summary().ResourcesID

	if strings.TrimSpace(msg.Message) != "" {
		rmevents.Publish(task.AllocationID, &sproto.ContainerLog{
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

	log.Infof("dispatch exited with exit code %d", msg.ExitCode)

	rmevents.Publish(task.AllocationID, &sproto.ResourcesStateChanged{
		ResourcesID:      rID,
		ResourcesState:   sproto.Terminated,
		ResourcesStopped: &stopped,
	})

	allocationID := m.getAllocationID(msg.DispatchID)

	// Find the Dispatch IDs associated with the allocation ID. We'll need the
	// Dispatch ID to clean up the dispatcher environments for the job.
	dispatches, err := db.ListDispatchesByAllocationID(context.TODO(), allocationID)
	if err != nil {
		log.WithError(err).
			Error("failed to retrieve the dispatches")
		return
	}
	log.Debugf("found %d dispatches", len(dispatches))

	// Cleanup all the dispatcher environments associated with current allocation
	for _, dispatch := range dispatches {
		dispatchID := dispatch.DispatchID
		impersonatedUser := dispatch.ImpersonatedUser

		if m.syslog.Logger.Level < logrus.DebugLevel {
			log.WithField("impersonated-user", impersonatedUser).
				Infof("deleting dispatcher environment")

			// Cleanup the dispatcher environment
			m.removeDispatchEnvironment(impersonatedUser, dispatchID)
		}
	}

	// Remove the dispatch from mapping tables.
	m.dispatchIDToHPCJobID.Delete(msg.DispatchID)
}

// Common method for sending a terminate request, and appropriately clean up a dispatch.
// Called only from killAllInactiveDispatches which is always run via go routine.
// Note to developers: this function must not acquire locks, unless they careful avoid being
// held over the API and DB calls.
func (m *DispatcherResourceManager) terminateAndDeleteDispatch(
	dispatchID string,
	impersonatedUser string,
) {
	log := m.syslog.WithField("dispatch-id", dispatchID)

	log.WithField("impersonated-user", impersonatedUser).
		Info("terminating dispatch job initiated by user")

	if m.terminateDispatcherJob(dispatchID, impersonatedUser, false) {
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
			log.Debug("not removing environment for dispatch because job is being monitored")
		} else {
			// If we are here, then we are likely being called from
			// startup, as opposed to a user explicitly canceling
			// a job. It's OK to remove the environment in this case
			// because we aren't actively monitoring any jobs, but we need to wait
			// for the terminate request above to complete, before we can actually
			// do the delete of the environment to avoid a 500 error response.
			m.waitForDispatchTerminalState(impersonatedUser, dispatchID)
			m.removeDispatchEnvironment(impersonatedUser, dispatchID)
		}
	}
}

// Wait up to 2mins for the dispatch to be in a terminal state.
func (m *DispatcherResourceManager) waitForDispatchTerminalState(
	impersonatedUser string, dispatchID string,
) {
	log := m.syslog.WithField("dispatch-id", dispatchID)

	for i := 0; i < 20; i++ {
		if m.jobWatcher.isDispatchInProgress(impersonatedUser, dispatchID) {
			log.Debugf("dispatch still active, waiting for termination")
			time.Sleep(6 * time.Second)
		} else {
			return
		}
	}
	log.Warn("dispatch still active, but wait time exceeded, continuing...")
}

func (m *DispatcherResourceManager) startLauncherJob(
	msg StartDispatcherResources,
	req *sproto.AllocateRequest,
) {
	dispatchID := string(msg.AllocationID)

	// No longer a scheduled launch, since we've now actually launched the job.
	defer m.scheduledLaunches.Delete(msg.AllocationID)

	// Log at INFO level so that we know we got this far. We had an issue on the
	// Grenoble cluster where an attempt to delete completed experiments failed
	// because the CHECKPOINT_GC task never ran. There was nothing in the log
	// indicated that the launcher ever got the request. Therefore, going
	// forward, make sure that we record that we got the request in the log to
	// help us troubleshoot customer issues.
	m.syslog.WithField("allocation-id", msg.AllocationID).
		WithField("description", msg.Spec.Description).
		WithField("scheduled-launches", m.scheduledLaunches.Len()).
		Info("received request to launch job")

	hpcDetails, err := m.hpcDetailsCache.load()
	if err != nil {
		m.sendResourceStateChangedErrorResponse(err, msg,
			"unable to start jobs without HPC details cache written")
		return
	}

	// TODO: There is a 'which first?' issue with resolving slot type and partition that needs to be
	// unwound before it causes a bug.
	partition := m.getProvidingPartition(req.ResourcePool)

	slotType := device.CPU
	// Only resolve the slot type if the number of slots requested is non-zero.
	// Checkpoint GC tasks will always request zero slots and they should
	// remain with a slot type of "CPU".
	if req.SlotsNeeded > 0 {
		slotType = m.resolveSlotType(hpcDetails, partition)
	}

	// Make sure we explicitly choose a partition.  Use default if unspecified.
	if partition == "" {
		partition = m.getDefaultPoolName(hpcDetails, slotType == device.CPU)
	}

	tresSupported := m.rmConfig.TresSupported
	gresSupported := m.rmConfig.GresSupported
	if m.rmConfig.TresSupported && !m.rmConfig.GresSupported {
		m.syslog.Warn("tres_supported: true cannot be used when " +
			"gres_supported: false is specified. Use tres_supported: false instead.")
		tresSupported = false
	}

	disabledAgents := set.FromSlice(append(m.dbState.DisabledAgents, req.BlockedNodes...)).ToSlice()

	// Create the manifest that will be ultimately sent to the launcher.
	manifest, impersonatedUser, payloadName, err := msg.Spec.ToDispatcherManifest(
		m.syslog, string(req.AllocationID),
		m.masterTLSConfig.Enabled,
		m.rmConfig.MasterHost, m.rmConfig.MasterPort, m.masterTLSConfig.CertificateName,
		req.SlotsNeeded, slotType, partition, tresSupported, gresSupported,
		m.rmConfig.LauncherContainerRunType, m.wlmType == pbsSchedulerType,
		m.rmConfig.JobProjectSource, disabledAgents,
	)
	if err != nil {
		m.sendResourceStateChangedErrorResponse(err, msg,
			"unable to launch job")
		return
	}

	if impersonatedUser == root && m.rmConfig.UserName != root {
		m.sendResourceStateChangedErrorResponse(
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

	warning := msg.Spec.WarnUnsupportedOptions(
		msg.UserConfiguredPriority, m.rmConfig.LauncherContainerRunType)

	if len(warning) > 0 {
		rmevents.Publish(msg.AllocationID, &sproto.ContainerLog{
			AuxMessage: &warning,
			Level:      ptrs.Ptr("WARNING"),
		})
	}

	m.syslog.WithField("dispatch-id", dispatchID).
		WithField("description", msg.Spec.Description).
		Info("dispatch created")

	// Pre-register dispatchID (which is now the AllocationID) so we can
	// handle events from the launched job and insert the dispatch into
	// the DB so that we ensure that it is later cleaned-up
	// if the launch is successful.
	if err := db.InsertDispatch(context.TODO(), &db.Dispatch{
		DispatchID:       dispatchID,
		ResourceID:       msg.ResourcesID,
		AllocationID:     req.AllocationID,
		ImpersonatedUser: impersonatedUser,
	}); err != nil {
		m.syslog.WithField("dispatch-id", dispatchID).
			WithError(err).Errorf("failed to persist dispatch")
	}

	// Pre-register dispatchID (which is now the AllocationID) with the job
	// monitor such that notifyContainerRunning calls that might be delivered prior
	// to the synchronous launch returning will be handled properly.
	m.jobWatcher.monitorJob(impersonatedUser, dispatchID, payloadName, true)

	tempDispatchID, err := m.sendManifestToDispatcher(
		manifest, impersonatedUser, string(msg.AllocationID))

	// Failed launch, clear pre-registered dispatchID==AllocationID
	if err != nil {
		m.syslog.WithField("dispatch-id", dispatchID).
			WithField("description", msg.Spec.Description).
			Infof("remove dispatch from failed launch")

		_, dberr := db.DeleteDispatch(context.TODO(), dispatchID)
		if dberr != nil {
			m.syslog.WithField("dispatch-id", dispatchID).
				WithError(dberr).Errorf("failed to delete dispatch from DB")
		}

		m.jobWatcher.removeJob(dispatchID)

		m.sendResourceStateChangedErrorResponse(err, msg, "")
	} else {
		// Successful launch, clear launchInProgress status
		m.jobWatcher.notifyJobLaunched(dispatchID)

		if tempDispatchID != dispatchID {
			incompMsg := "HPC Launcher version is below the minimum required. " +
				"Update to version 3.3.1 or greater."
			m.syslog.
				WithField("dispatch-id", dispatchID).
				WithField("description", msg.Spec.Description).
				Errorf("launcher did not honor DispatchID assignment.  " +
					incompMsg)
			rmevents.Publish(req.AllocationID, &sproto.ContainerLog{
				AuxMessage: &incompMsg,
				Level:      ptrs.Ptr("ERROR"),
			})
		}
	}
}

// stopLauncherJob is called only via KillDispatcherResources and called via go routine.
// Note to developers: this function must not acquire locks, unless they careful avoid being
// held over the API and DB calls.
func (m *DispatcherResourceManager) stopLauncherJob(msg KillDispatcherResources) {
	// Log at INFO level to let us know that the dispatcher resource manager
	// actually received the request to delete the job.
	m.syslog.WithField("allocation-id", msg.AllocationID).
		Info("received request to terminate job")

	// Make a note that there is a cancelation inflight for this job, so that
	// if another cancelation request is received, we ignore it and don't queue
	// it.
	m.inflightCancelations.Store(msg.AllocationID, struct{}{})
	defer m.inflightCancelations.Delete(msg.AllocationID)

	// Find the Dispatch IDs associated with the allocation ID. We'll need the
	// Dispatch ID to cancel the job on the launcher side.
	dispatches, err := db.ListDispatchesByAllocationID(context.TODO(), msg.AllocationID)
	if err != nil {
		m.syslog.WithField("allocation-id", msg.AllocationID).WithError(err).Errorf(
			"failed to retrieve the dispatches")

		return
	}

	// The job cancelation message arrived before the launcher created the
	// dispatch ID. Since we can't cancel the job without the dispatch ID,
	// return and wait for Determined to call us again for a retry.
	if len(dispatches) == 0 {
		m.syslog.WithField("allocation-id", msg.AllocationID).
			Info("job termination handler found 0 jobs associated with allocation")

		return
	}

	m.syslog.WithField("allocation-id", msg.AllocationID).
		Debugf("job termination handler found %d jobs associated with allocation",
			len(dispatches))

	for _, dispatch := range dispatches {
		dispatchID := dispatch.DispatchID
		impersonatedUser := dispatch.ImpersonatedUser

		// Get the HPC job ID, if it's available, to include in the log message.
		hpcJobID, _ := m.dispatchIDToHPCJobID.Load(dispatchID)

		logger := m.syslog.WithField("dispatch-id", dispatchID).
			WithField("hpc-job-id", hpcJobID).
			WithField("impersonated-user", impersonatedUser)

		// When the job monitor's queue is large, it may take a while for the
		// job monitor to query the launcher for confirmation that the Workload
		// Manager (Slurm/PBS) has terminated the job. Therefore, don't keep
		// sending termination requests to the launcher if a termination request
		// has already been sent, but we're just waiting for the job monitor to
		// query the launcher for termination confirmation.  As a safeguard
		// against a termination request that was sent to the launcher, but the
		// launcher didn't act upon it, allow another termination request to be
		// sent again if it's been 300 seconds (5 minutes) since the job monitor
		// last sent a job termination request to the launcher.
		if m.jobWatcher.isJobMarkedAsTerminated(dispatchID) &&
			time.Since(m.jobWatcher.getLastJobTerminationRequestTime(dispatchID)).Seconds() < 300 {
			logger.WithField("canceled-job-position", m.jobWatcher.getJobListPosition(dispatchID)).
				WithField("current-job-position", m.jobWatcher.currentJobPosition.Load()).
				WithField("last-update", m.jobWatcher.getLastJobStatusCheckTime(dispatchID).Format("2006-01-02 15:04:05")).
				Info("termination request already sent, waiting for acknowledgement of termination")
			return
		}

		logger.Info("terminating job initiated by user")

		// Terminate and cleanup, on failure leave Dispatch in DB for later retry
		if m.terminateDispatcherJob(dispatchID, impersonatedUser, false) {
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
				m.syslog.WithField("dispatch-id", dispatchID).Debug(
					"not removing dispatch environment because job is being monitored")
			} else {
				// If we are here, then we are likely being called from
				// startup, as opposed to a user explicitly canceling
				// a job. It's OK to remove the environment in this case
				// because we aren't actively monitoring any jobs, but we need to wait
				// for the terminate request above to complete, before we can actually
				// do the delete of the environment to avoid a 500 error response.
				m.waitForDispatchTerminalState(impersonatedUser, dispatchID)
				m.removeDispatchEnvironment(impersonatedUser, dispatchID)

				// The job monitor usually takes care of notifying Determined
				// that the job terminated, but since the job is no longer
				// being monitored, we have to send the notification ourselves,
				// so that the job doesn't remain in the STOPPING_CANCELED
				// state.
				m.handleDispatchExited(DispatchExited{
					DispatchID: dispatchID,
					ExitCode:   -1,
					Message:    "Job was canceled",
				})
			}
		}
	}
}

// Log the failure, and send a ResourcesStateChanged describing the failure.
func (m *DispatcherResourceManager) sendResourceStateChangedErrorResponse(
	err error,
	msg StartDispatcherResources,
	errMessageStr string,
) {
	m.syslog.WithError(err).Error(errMessageStr)
	stopped := sproto.ResourcesStopped{}
	stopped.Failure = sproto.NewResourcesFailure(
		sproto.ResourcesFailed,
		errors.Wrapf(err, errMessageStr).Error(),
		nil,
	)
	rmevents.Publish(msg.AllocationID, &sproto.ResourcesStateChanged{
		ResourcesID: msg.ResourcesID,
		// Could be a better message("container failed with non-zero exit code")
		ResourcesState:   sproto.Terminated,
		ResourcesStopped: &stopped,
	})
}

// getDefaultPoolName returns the default aux pool if the arg is true,
// otherwise default compute pool.
// Note to the developer: this must not acquire a lock.
func (m *DispatcherResourceManager) getDefaultPoolName(
	hpcDetails *hpcResources, isCPU bool,
) string {
	if isCPU {
		return hpcDetails.DefaultAuxPoolPartition
	}
	return hpcDetails.DefaultComputePoolPartition
}

// isValidProvider returns true is a usable Provider definition has been provided.
func isValidProvider(pool config.ResourcePoolConfig) bool {
	return pool.Provider != nil && pool.Provider.HPC != nil
}

func duplicateResourcePool(basePool *resourcepoolv1.ResourcePool) *resourcepoolv1.ResourcePool {
	return proto.Clone(basePool).(*resourcepoolv1.ResourcePool)
}

// getWlmResources returns various WLM-dependent resources used in constructing a resource pool.
// Note to the developer: this must not acquire a lock.
func (m *DispatcherResourceManager) getWlmResources() (
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

// resolveSlotType resolves the correct slot type for a job targeting the given partition. If the
// slot type is specified in the master config, use that. Otherwise if the partition is specified
// and known, and has no GPUs select CPU as the processor type, else default to CUDA.
// Note to the developer: this must not acquire a lock.
func (m *DispatcherResourceManager) resolveSlotType(
	hpcDetails *hpcResources,
	partition string,
) device.Type {
	if slotType := m.rmConfig.ResolveSlotType(partition); slotType != nil {
		return *slotType
	}

	for _, v := range hpcDetails.Partitions {
		if v.PartitionName == partition && v.TotalGpuSlots == 0 {
			return device.CPU
		}
	}
	return device.CUDA
}

// ResourceQueryPostActions performs actions to clean up after any dispatch
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
// Called only from fetchHpcResourceDetails and always run via go routine
// except the one time during startup to retrieve initial cluster cache.
func (m *DispatcherResourceManager) ResourceQueryPostActions(
	dispatchID string, owner string,
) {
	if m.terminateDispatcherJob(dispatchID, owner, true) {
		m.removeDispatchEnvironment(owner, dispatchID)
	}
}

// terminateDispatcherJob terminates the dispatcher job with the given ID.
// Return true to indicate if the DB dispatch should additionally be deleted.
// Note to developers: this function must not acquire locks.
func (m *DispatcherResourceManager) terminateDispatcherJob(
	dispatchID string, owner string, slurmResourcesPolling bool,
) bool {
	if dispatchID == "" {
		m.syslog.Warn("missing dispatchID, so no environment clean-up")
		return false
	}

	// The logger we will pass to the API client, so that when the API client
	// logs a message, we know who called it.
	launcherAPILogger := m.syslog.WithField("caller", "terminateDispatcherJob")

	_, _, err := m.apiClient.terminateDispatch( //nolint:bodyclose
		owner,
		dispatchID,
		launcherAPILogger)
	if err != nil {
		m.syslog.WithField("dispatch-id", dispatchID).
			WithError(err).Errorf("failed to terminate dispatch job")
		return false
	}

	if slurmResourcesPolling {
		m.syslog.WithField("dispatch-id", dispatchID).Debug("terminated dispatch job")
	} else {
		m.syslog.WithField("dispatch-id", dispatchID).Info("terminated dispatch job")
	}

	// Let the job monitor know that the job was terminated, otherwise it
	// might get a 404 (Not Found) error from the launcher and not send
	// Determined notification that the job was terminated.
	m.jobWatcher.markJobAsTerminated(dispatchID)

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
// Note to developers: this function must not acquire locks.
func (m *DispatcherResourceManager) removeDispatchEnvironment(
	owner string, dispatchID string,
) {
	log := m.syslog.WithField("dispatch-id", dispatchID).WithField("owner", owner)

	// The logger we will pass to the API client, so that when the API client
	// logs a message, we know who called it.
	launcherAPILogger := m.syslog.WithField("caller", "removeDispatchEnvironment")

	_, err := m.apiClient.deleteDispatch(owner, dispatchID, launcherAPILogger) //nolint:bodyclose
	if err != nil {
		log.WithError(err).Error("failed to delete dispatch")
		return
	}

	count, err := db.DeleteDispatch(context.TODO(), dispatchID)
	if err != nil {
		log.WithError(err).Error("failed to delete dispatch from DB")
		return
	}
	// On Slurm resource query there may be no Dispatch in the DB, so only log as trace.
	log.Tracef("Deleted dispatch from DB, count %d", count)
}

// Sends the manifest to the launcher.
func (m *DispatcherResourceManager) sendManifestToDispatcher(
	manifest *launcher.Manifest,
	impersonatedUser string,
	allocationID string,
) (string, error) {
	// The logger we will pass to the API client, so that when the API client
	// logs a message, we know who called it.
	launcherAPILogger := m.syslog.WithField("caller", "sendManifestToDispatcher")

	//nolint:bodyclose
	dispatchInfo, response, err := m.apiClient.launchDispatcherJob(
		manifest,
		impersonatedUser,
		allocationID,
		launcherAPILogger)
	if err != nil {
		if response != nil {
			// If we have a real error body, return the details message
			if details := extractDetailsFromResponse(response, err); len(details) > 0 {
				return "", errors.New(details)
			}
			return "", errors.Wrapf(err, m.apiClient.handleLauncherError(
				response, "Job launch failed", err))
		}
		if strings.Contains(err.Error(), "EOF") {
			return "", errors.Wrapf(err, "Launcher rejected the job due to "+
				"excessive outstanding requests.  Normal operation will typically "+
				"resume once the outstanding requests have been processed.")
		}
		return "", errors.Wrapf(err, "Job launch failed. "+
			"Verify that the launcher service is up and reachable.")
	}
	return dispatchInfo.GetDispatchId(), nil
}

func (m *DispatcherResourceManager) addTask(msg sproto.AllocateRequest) {
	m.getOrCreateGroup(msg.JobID)
	if len(msg.Name) == 0 {
		msg.Name = "Unnamed-Launcher-Job"
	}

	m.syslog.WithField("name", msg.Name).
		WithField("allocation-id", msg.AllocationID).
		Info("resources are requested")
	m.reqList.AddTask(&msg)
}

func (m *DispatcherResourceManager) assignResources(req *sproto.AllocateRequest) {
	var dispatchID string
	var impersonatedUser string
	var rID sproto.ResourcesID

	if req.Restore {
		// Find the Dispatch IDs associated with the allocation ID. We'll need the
		// Dispatch ID to reconnect with the existing allocation.
		dispatches, err := db.ListDispatchesByAllocationID(context.TODO(), req.AllocationID)
		if err != nil {
			m.syslog.WithField("allocation-id", req.AllocationID).
				WithError(err).Errorf("failed to retrieve dispatches")
			return
		}

		m.syslog.WithField("allocation-id", req.AllocationID).
			Debugf("restore: found %d dispatches",
				len(dispatches))

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
			rm:                     m,
			group:                  m.groups[req.JobID],
			defaultRendezvousIface: m.rmConfig.ResolveRendezvousNetworkInterface(req.ResourcePool),
			defaultProxyIface:      m.rmConfig.ResolveProxyNetworkInterface(req.ResourcePool),
		},
	}

	assigned := sproto.ResourcesAllocated{ID: req.AllocationID, Resources: allocations}
	m.reqList.AddAllocationRaw(req.AllocationID, &assigned)
	rmevents.Publish(req.AllocationID, assigned.Clone())

	if req.Restore {
		if len(dispatchID) == 0 {
			m.syslog.Info("restore request with no active dispatch found, fail the allocation request")
			failed := sproto.NewResourcesFailure(sproto.ResourcesAborted,
				"Unable to locate HPC job on restart.", nil)
			stopped := sproto.ResourcesStopped{}
			stopped.Failure = failed
			rmevents.Publish(req.AllocationID, &sproto.ResourcesStateChanged{
				ResourcesID:      rID,
				ResourcesState:   sproto.Terminated,
				ResourcesStopped: &stopped,
			})
		} else {
			// Simulate portions of Start() which will not be called on restore.
			m.syslog.WithField("resource-id", rID).
				WithField("dispatch-id", dispatchID).
				WithField("impersonated-user", impersonatedUser).
				Info("reconnecting")
			m.jobWatcher.monitorJob(impersonatedUser, dispatchID, "", false)
		}
	} else {
		m.syslog.
			WithField("allocation-id", req.AllocationID).
			WithField("task-handler", req.Name).
			Infof("resources assigned")
	}
}

// Perform a terminate and delete all dispatches in the DB
// that are no-longer associated with an active experiment/task.
// All active tasks will get reconnected via AllocationRequest{Restore:true}
// events.  This case is to handle those that will not be restored.
// When in debug mode it continues to periodically executed to prune
// terminated dispatches for which we are deferring deletion.
func (m *DispatcherResourceManager) killAllInactiveDispatches() {
	// Ticker with an initial pass through without delay
	ticker := time.NewTicker(terminatedDispatchCleanupInterval)
	for ; true; <-ticker.C {
		m.syslog.Info("releasing all dispatches for terminated allocations")

		// Find the Dispatch IDs
		dispatches, err := db.ListAllDispatches(context.TODO())
		if err != nil {
			m.syslog.WithError(err).Error("failed to retrieve all dispatches")
			return
		}
		m.syslog.Debugf("found %d dispatches to check", len(dispatches))
		for _, dispatch := range dispatches {
			dispatchID := dispatch.DispatchID
			impersonatedUser := dispatch.ImpersonatedUser
			allocation, err := db.AllocationByID(context.TODO(), dispatch.AllocationID)
			if err != nil {
				m.syslog.WithField("dispatch-id", dispatchID).
					WithError(err).Errorf("unexpected DB lookup error")
				continue
			} else if allocation != nil && allocation.EndTime == nil {
				m.syslog.WithField("dispatch-id", dispatchID).
					Debug("not removing dispatch environment for dispatch because allocation is still active.")
				continue
			}

			m.terminateAndDeleteDispatch(dispatchID, impersonatedUser)
		}

		if m.syslog.Logger.Level < logrus.DebugLevel {
			// Do only one cleanup unless in debug mode
			return
		}
	}
}

func (m *DispatcherResourceManager) getOrCreateGroup(jobID model.JobID) *tasklist.Group {
	if g, ok := m.groups[jobID]; ok {
		return g
	}

	priority := config.KubernetesDefaultPriority
	g := &tasklist.Group{JobID: jobID, Weight: 1, Priority: &priority}
	m.groups[jobID] = g
	tasklist.GroupPriorityChangeRegistry.OnDelete(jobID, func() {
		m.mu.Lock()
		defer m.mu.Unlock()
		delete(m.groups, jobID)
	})
	return g
}

func (m *DispatcherResourceManager) periodicallySchedulePendingTasks() {
	t := time.NewTicker(actionCoolDown)
	defer t.Stop()
	for range t.C {
		m.SchedulePendingTasks()
	}
}

// SchedulePendingTasks is called periodically to respond to allocations with resources when we
// have capacity to launch.
// Note to developers: this function only locks over DB calls in the restore path. Let's keep it
// this way.
func (m *DispatcherResourceManager) SchedulePendingTasks() {
	m.mu.Lock()
	defer m.mu.Unlock()

	numTimesScheduledPendingTasksCalled++

	for it := m.reqList.Iterator(); it.Next(); {
		req := it.Value()
		if !m.reqList.IsScheduled(req.AllocationID) {
			// A restore means that the Determined master was restarted and
			// we're simply monitoring the jobs we previously launched. When
			// it's not a restore, we want to limit the number of launch
			// requests we send to the launcher, so that we don't overwhelm
			// the launcher with too many concurrent requests.
			if !req.Restore {
				count := m.scheduledLaunches.Len()
				if count >= maxJobLaunchGoRoutines {
					// To help us troubleshoot problems, log a message every 10
					// seconds when we've reached our goroutine limit. The
					// "schedulePendingTasks()" function gets called twice a
					// second, so we modulo divide by 20 to log the message
					// every 10 seconds.
					if numTimesScheduledPendingTasksCalled%20 == 0 {
						m.syslog.WithField("scheduled-launches", count).
							Info("delaying start of task because the concurrent launch limit has been reached")
					}
					return
				}

				// Add the allocation ID to the "scheduledLaunches" map. The
				// "startLauncherJob()" function will remove the allocation ID
				// from the map when it's launched the job. The
				// "resourcesReleased()" function will also remove the
				// allocation ID from the map, since jobs that are canceled
				// too quickly may never call "startLauncherJob()".
				m.scheduledLaunches.Store(req.AllocationID, struct{}{})
			}

			m.assignResources(req)
		}
	}
}

// Note to developers: this function doesn't acquire a lock and, ideally, we shouldn't make it.
func (m *DispatcherResourceManager) findAgent(agentID string) (*agentv1.Agent, error) {
	hpcDetails, err := m.hpcDetailsCache.load()
	if err != nil {
		return nil, err
	}

	for _, node := range hpcDetails.Nodes {
		if node.Name == agentID {
			return m.hpcNodeToAgent(node), nil
		}
	}
	return nil, fmt.Errorf("agent %s not found", agentID)
}

type (
	// DispatcherResources information.
	DispatcherResources struct {
		id    sproto.ResourcesID
		req   *sproto.AllocateRequest
		rm    *DispatcherResourceManager
		group *tasklist.Group

		defaultRendezvousIface string
		defaultProxyIface      string
	}

	// StartDispatcherResources comment to keep "golint" from complaining.
	StartDispatcherResources struct {
		AllocationID           model.AllocationID
		ResourcesID            sproto.ResourcesID
		Spec                   tasks.TaskSpec
		UserConfiguredPriority bool
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

	// dispatchExpLogMessage notifies the dispatcher of a message to be added to the exp log.
	dispatchExpLogMessage struct {
		DispatchID string
		Message    string
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
	_ logger.Context, spec tasks.TaskSpec, rri sproto.ResourcesRuntimeInfo,
) error {
	spec.ResourcesID = string(r.id)
	spec.AllocationID = string(r.req.AllocationID)
	spec.AllocationSessionToken = rri.Token
	spec.TaskID = string(r.req.TaskID)
	spec.UseHostMode = rri.IsMultiAgent

	// HPC launcher is setting a value for resources.priority. The user configured
	// value will be ignored. A warning message will be given if user configured
	// this option. To generate the warning, we need to record if this option is configured
	// before it is changed by the code below.
	userConfiguredPriority := false
	if spec.ResourcesConfig.Priority() != nil {
		userConfiguredPriority = true
	}
	spec.ResourcesConfig.SetPriority(r.group.Priority)

	if spec.LoggingFields == nil {
		spec.LoggingFields = map[string]string{}
	}
	spec.LoggingFields["allocation_id"] = spec.AllocationID
	spec.LoggingFields["task_id"] = spec.TaskID
	spec.ExtraEnvVars[sproto.ResourcesTypeEnvVar] = string(sproto.ResourcesTypeSlurmJob)
	spec.ExtraEnvVars[sproto.SlurmRendezvousIfaceEnvVar] = r.defaultRendezvousIface
	spec.ExtraEnvVars[sproto.SlurmProxyIfaceEnvVar] = r.defaultProxyIface
	r.rm.StartDispatcherResources(StartDispatcherResources{
		AllocationID:           r.req.AllocationID,
		ResourcesID:            r.id,
		Spec:                   spec,
		UserConfiguredPriority: userConfiguredPriority,
	})
	return nil
}

// Kill notifies the pods actor that it should stop the pod.
func (r DispatcherResources) Kill(_ logger.Context) {
	r.rm.KillDispatcherResources(KillDispatcherResources{
		ResourcesID:  r.id,
		AllocationID: r.req.AllocationID,
	})
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
// Note to developers: this function doesn't need to acquire a lock. Let's keep it that way.
func (m *DispatcherResourceManager) NotifyContainerRunning(msg sproto.NotifyContainerRunning) error {
	dispatches, err := db.ListDispatchesByAllocationID(context.TODO(), msg.AllocationID)
	if err != nil {
		m.syslog.WithField("allocation-id", msg.AllocationID).
			WithError(err).Errorf("Failed to retrieve the dispatch associated with allocation")
		return nil
	}

	foundMonitoredDispatch := false
	for _, dispatch := range dispatches {
		dispatchID := dispatch.DispatchID
		if m.jobWatcher.isJobBeingMonitored(dispatchID) {
			foundMonitoredDispatch = true
			m.jobWatcher.notifyContainerRunning(dispatchID, msg.Rank, msg.NumPeers, msg.NodeName)
		}
	}
	if !foundMonitoredDispatch {
		m.syslog.WithField("allocation-id", msg.AllocationID).Warnf(
			"NotifyContainerRunning did not find an active, monitored dispatch")
	}
	return nil
}

type taskContainerDefaults struct {
	fallbackDefault model.TaskContainerDefaultsConfig
	resourcePool    string
}

// TaskContainerDefaults returns TaskContainerDefaults for the specified pool.
// Note to developers: this function doesn't need to acquire a lock. Let's keep it that way.
func (m *DispatcherResourceManager) TaskContainerDefaults(
	resourcePoolName string,
	defaultConfig model.TaskContainerDefaultsConfig,
) (model.TaskContainerDefaultsConfig, error) {
	result := defaultConfig

	partition := m.getProvidingPartition(resourcePoolName)
	partitionOverrides := m.rmConfig.ResolveTaskContainerDefaults(partition)
	if partitionOverrides != nil {
		tmp, err := result.Merge(*partitionOverrides)
		if err != nil {
			return model.TaskContainerDefaultsConfig{}, err
		}
		result = tmp
	}

	var poolConfigOverrides *model.TaskContainerDefaultsConfig
	for _, pool := range m.poolConfig {
		if resourcePoolName == pool.PoolName {
			if pool.TaskContainerDefaults == nil {
				break
			}
			poolConfigOverrides = pool.TaskContainerDefaults
		}
	}
	if poolConfigOverrides != nil {
		tmp, err := result.Merge(*poolConfigOverrides)
		if err != nil {
			return model.TaskContainerDefaultsConfig{}, err
		}
		result = tmp
	}

	return result, nil
}

// EnableSlot implements 'det slot enable...' functionality.
func (m *DispatcherResourceManager) EnableSlot(
	req *apiv1.EnableSlotRequest,
) (resp *apiv1.EnableSlotResponse, err error) {
	return nil, errNotSupportedOnHpcCluster
}

// DisableSlot implements 'det slot disable...' functionality.
func (m *DispatcherResourceManager) DisableSlot(
	req *apiv1.DisableSlotRequest,
) (resp *apiv1.DisableSlotResponse, err error) {
	return nil, errNotSupportedOnHpcCluster
}
