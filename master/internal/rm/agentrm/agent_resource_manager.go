package agentrm

import (
	"context"
	"crypto/tls"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/rm/rmerrors"
	"github.com/determined-ai/determined/master/internal/rm/rmevents"
	"github.com/determined-ai/determined/master/internal/rm/rmutils"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/aproto"
	"github.com/determined-ai/determined/master/pkg/command"
	"github.com/determined-ai/determined/master/pkg/device"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/syncx/queue"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/jobv1"
	"github.com/determined-ai/determined/proto/pkg/resourcepoolv1"
)

// New returns a new ResourceManager, which manages communicating with
// and scheduling on Determined agents.
func New(
	db *db.PgDB,
	e *echo.Echo,
	config *config.ResourceConfig,
	opts *aproto.MasterSetAgentOptions,
	cert *tls.Certificate,
) *ResourceManager {
	agentService, agentUpdates := newAgentService(config.ResourcePools, opts)

	e.GET("/agents", func(c echo.Context) error {
		if !c.IsWebSocket() {
			return echo.ErrBadRequest
		}
		return agentService.HandleWebsocketConnection(webSocketRequest{echoCtx: c})
	})

	return newAgentResourceManager(db, config, cert, agentService, agentUpdates)
}

// A ResourceManager manages many resource pools and routing requests for resources to them.
type ResourceManager struct {
	syslog *logrus.Entry

	config      *config.AgentResourceManagerConfig
	poolsConfig []config.ResourcePoolConfig
	cert        *tls.Certificate
	db          *db.PgDB

	agentService *agents
	agentUpdates *queue.Queue[agentUpdatedEvent]
	pools        map[string]*resourcePool // immutable. cannot be made mutable without significant change.
}

func newAgentResourceManager(
	db *db.PgDB, config *config.ResourceConfig, cert *tls.Certificate, agentService *agents,
	agentUpdates *queue.Queue[agentUpdatedEvent],
) *ResourceManager {
	a := &ResourceManager{
		syslog: logrus.WithField("component", "agentrm"),

		config:       config.ResourceManager.AgentRM,
		poolsConfig:  config.ResourcePools,
		cert:         cert,
		db:           db,
		agentService: agentService,
		agentUpdates: agentUpdates,
		pools:        make(map[string]*resourcePool),
	}

	for ix, config := range a.poolsConfig {
		rp, err := a.createResourcePool(a.db, a.poolsConfig[ix], a.cert)
		if err != nil {
			// TODO(DET-9975): Don't panic.
			a.syslog.WithError(err).Errorf("failed to create resource pool: %s", a.poolsConfig[ix].PoolName)
			panic(err)
		}
		a.pools[config.PoolName] = rp
	}
	go func() {
		for {
			update := a.agentUpdates.Get()
			pool, ok := a.pools[update.resourcePool]
			if !ok {
				a.syslog.Warn("ignoring agent update for unknown pool: %w", update.resourcePool)
				continue
			}
			pool.NotifyAgentUpdated()
		}
	}()

	return a
}

// Allocate implements rm.ResourceManager.
func (a *ResourceManager) Allocate(msg sproto.AllocateRequest) (*sproto.ResourcesSubscription, error) {
	// this code exists to handle the case where an experiment does not have
	// an explicit resource pool specified in the config. This should never happen
	// for newly created/forked experiments as the default pool is filled in to the
	// config at creation time. However, old experiments which were created prior to
	// the introduction of resource pools could have no resource pool associated with
	// them and so we need to handle that case gracefully.
	if len(msg.ResourcePool) == 0 {
		if msg.SlotsNeeded == 0 {
			msg.ResourcePool = a.config.DefaultAuxResourcePool
		} else {
			msg.ResourcePool = a.config.DefaultComputeResourcePool
		}
	}
	pool, err := a.poolByName(msg.ResourcePool)
	if err != nil {
		a.syslog.WithError(err).Error("handling an allocate request")
		return nil, err
	}

	sub := rmevents.Subscribe(msg.AllocationID)
	pool.Allocate(msg)
	return sub, nil
}

// DeleteJob implements rm.ResourceManager.
func (*ResourceManager) DeleteJob(sproto.DeleteJob) (sproto.DeleteJobResponse, error) {
	return sproto.EmptyDeleteJobResponse(), nil
}

// DisableAgent implements rm.ResourceManager.
func (a *ResourceManager) DisableAgent(msg *apiv1.DisableAgentRequest) (*apiv1.DisableAgentResponse, error) {
	agent, ok := a.agentService.get(agentID(msg.AgentId))
	if !ok {
		return nil, api.NotFoundErrs("agent", msg.AgentId, true)
	}
	return agent.DisableAgent(msg)
}

// DisableSlot implements rm.ResourceManager.
func (a *ResourceManager) DisableSlot(req *apiv1.DisableSlotRequest) (*apiv1.DisableSlotResponse, error) {
	deviceIDStr, err := strconv.Atoi(req.SlotId)
	if err != nil {
		return nil, fmt.Errorf("invalid slot id: %s", req.SlotId)
	}
	deviceID := device.ID(deviceIDStr)

	enabled := false
	result, err := a.handlePatchSlotState(agentID(req.AgentId), patchSlotState{
		id:      deviceID,
		enabled: &enabled,
		drain:   &req.Drain,
	})
	if err != nil {
		return nil, err
	}
	return &apiv1.DisableSlotResponse{Slot: result.ToProto()}, nil
}

// EnableAgent implements rm.ResourceManager.
func (a *ResourceManager) EnableAgent(msg *apiv1.EnableAgentRequest) (*apiv1.EnableAgentResponse, error) {
	agent, ok := a.agentService.get(agentID(msg.AgentId))
	if !ok {
		return nil, api.NotFoundErrs("agent", msg.AgentId, true)
	}
	return agent.EnableAgent(msg)
}

// EnableSlot implements rm.ResourceManager.
func (a *ResourceManager) EnableSlot(req *apiv1.EnableSlotRequest) (*apiv1.EnableSlotResponse, error) {
	deviceIDStr, err := strconv.Atoi(req.SlotId)
	if err != nil {
		return nil, fmt.Errorf("invalid slot id: %s", req.SlotId)
	}
	deviceID := device.ID(deviceIDStr)

	enabled := true
	result, err := a.handlePatchSlotState(agentID(req.AgentId), patchSlotState{id: deviceID, enabled: &enabled})
	if err != nil {
		return nil, err
	}
	return &apiv1.EnableSlotResponse{Slot: result.ToProto()}, nil
}

func (a *ResourceManager) handlePatchSlotState(
	agentID agentID, msg patchSlotState,
) (*model.SlotSummary, error) {
	agent, ok := a.agentService.get(agentID)
	if !ok {
		return nil, api.NotFoundErrs("agent", string(agentID), true)
	}
	return agent.PatchSlotState(msg)
}

// CheckMaxSlotsExceeded checks if the job exceeded the maximum number of slots.
func (a *ResourceManager) CheckMaxSlotsExceeded(v *sproto.ValidateResourcePoolAvailabilityRequest) (bool, error) {
	pool, err := a.poolByName(v.Name)
	if err != nil {
		return false, err
	}

	resp, err := pool.CapacityCheck(sproto.CapacityCheck{
		Slots:  v.Slots,
		TaskID: v.TaskID,
	})
	if err != nil {
		return false, err
	}
	return resp.CapacityExceeded, nil
}

// ExternalPreemptionPending implements rm.ResourceManager.
func (*ResourceManager) ExternalPreemptionPending(sproto.PendingPreemption) error {
	return rmerrors.ErrNotSupported
}

// GetAgent implements rm.ResourceManager.
func (a *ResourceManager) GetAgent(msg *apiv1.GetAgentRequest) (*apiv1.GetAgentResponse, error) {
	agent, ok := a.agentService.get(agentID(msg.AgentId))
	if !ok {
		return nil, api.NotFoundErrs("agent", msg.AgentId, true)
	}
	return agent.GetAgent(msg), nil
}

// GetAgents implements rm.ResourceManager.
func (a *ResourceManager) GetAgents(msg *apiv1.GetAgentsRequest) (*apiv1.GetAgentsResponse, error) {
	return a.agentService.getAgents(msg), nil
}

// GetAllocationSummaries implements rm.ResourceManager.
func (a *ResourceManager) GetAllocationSummaries(
	msg sproto.GetAllocationSummaries,
) (map[model.AllocationID]sproto.AllocationSummary, error) {
	summaries := make(map[model.AllocationID]sproto.AllocationSummary)
	for _, pool := range a.pools {
		rpSummaries := pool.GetAllocationSummaries(msg)
		maps.Copy(summaries, rpSummaries)
	}
	return summaries, nil
}

// GetAllocationSummary implements rm.ResourceManager.
func (a *ResourceManager) GetAllocationSummary(
	msg sproto.GetAllocationSummary,
) (*sproto.AllocationSummary, error) {
	for _, pool := range a.pools {
		summary, ok := pool.GetAllocationSummary(msg)
		if !ok {
			continue
		}
		return &summary, nil
	}
	return nil, errors.New("allocation not found")
}

// GetDefaultAuxResourcePool implements rm.ResourceManager.
func (a *ResourceManager) GetDefaultAuxResourcePool(
	sproto.GetDefaultAuxResourcePoolRequest,
) (sproto.GetDefaultAuxResourcePoolResponse, error) {
	if a.config.DefaultAuxResourcePool == "" {
		return sproto.GetDefaultAuxResourcePoolResponse{}, rmerrors.ErrNoDefaultResourcePool
	}
	return sproto.GetDefaultAuxResourcePoolResponse{PoolName: a.config.DefaultAuxResourcePool}, nil
}

// GetDefaultComputeResourcePool implements rm.ResourceManager.
func (a *ResourceManager) GetDefaultComputeResourcePool(
	sproto.GetDefaultComputeResourcePoolRequest,
) (sproto.GetDefaultComputeResourcePoolResponse, error) {
	if a.config.DefaultComputeResourcePool == "" {
		return sproto.GetDefaultComputeResourcePoolResponse{}, rmerrors.ErrNoDefaultResourcePool
	}
	return sproto.GetDefaultComputeResourcePoolResponse{
		PoolName: a.config.DefaultComputeResourcePool,
	}, nil
}

// GetExternalJobs implements rm.ResourceManager.
func (*ResourceManager) GetExternalJobs(sproto.GetExternalJobs) ([]*jobv1.Job, error) {
	return nil, rmerrors.ErrNotSupported
}

// GetJobQ implements rm.ResourceManager.
func (a *ResourceManager) GetJobQ(msg sproto.GetJobQ) (map[model.JobID]*sproto.RMJobInfo, error) {
	if msg.ResourcePool == "" {
		msg.ResourcePool = a.config.DefaultComputeResourcePool
	}

	pool, err := a.poolByName(msg.ResourcePool)
	if err != nil {
		return nil, err
	}
	return pool.GetJobQ(sproto.GetJobQ{ResourcePool: msg.ResourcePool}), nil
}

// GetJobQueueStatsRequest implements rm.ResourceManager.
func (a *ResourceManager) GetJobQueueStatsRequest(
	msg *apiv1.GetJobQueueStatsRequest,
) (*apiv1.GetJobQueueStatsResponse, error) {
	resp := &apiv1.GetJobQueueStatsResponse{
		Results: make([]*apiv1.RPQueueStat, 0),
	}

	for name, pool := range a.pools {
		if len(msg.ResourcePools) != 0 && !slices.Contains(msg.ResourcePools, name) {
			continue
		}

		stats := pool.GetJobQStats(sproto.GetJobQStats{})

		aggregates, err := a.fetchAvgQueuedTime(name)
		if err != nil {
			a.syslog.WithError(err).Error("fetch average queued time")
			continue
		}

		resp.Results = append(resp.Results, &apiv1.RPQueueStat{
			ResourcePool: name,
			Stats:        stats,
			Aggregates:   aggregates,
		})
	}

	return resp, nil
}

// GetResourcePools implements rm.ResourceManager.
func (a *ResourceManager) GetResourcePools(
	msg *apiv1.GetResourcePoolsRequest,
) (*apiv1.GetResourcePoolsResponse, error) {
	summaries := make([]*resourcepoolv1.ResourcePool, 0, len(a.poolsConfig))
	for _, pool := range a.poolsConfig {
		summary, err := a.createResourcePoolSummary(pool.PoolName)
		if err != nil {
			// Should only raise an error if the resource pool doesn't exist and that can't happen.
			// But best to handle it anyway in case the implementation changes in the future.
			a.syslog.WithError(err).Error("")
			return nil, err
		}

		jobStats, err := a.getPoolJobStats(pool)
		if err != nil {
			return nil, err
		}

		summary.Stats = jobStats
		summaries = append(summaries, summary)
	}
	return &apiv1.GetResourcePoolsResponse{ResourcePools: summaries}, nil
}

// GetSlot implements rm.ResourceManager.
func (a *ResourceManager) GetSlot(req *apiv1.GetSlotRequest) (*apiv1.GetSlotResponse, error) {
	deviceIDStr, err := strconv.Atoi(req.SlotId)
	if err != nil {
		return nil, fmt.Errorf("invalid slot id: %s", req.SlotId)
	}
	deviceID := device.ID(deviceIDStr)

	result, err := a.handlePatchSlotState(agentID(req.AgentId), patchSlotState{id: deviceID})
	if err != nil {
		return nil, err
	}
	return &apiv1.GetSlotResponse{Slot: result.ToProto()}, nil
}

// GetSlots implements rm.ResourceManager.
func (a *ResourceManager) GetSlots(msg *apiv1.GetSlotsRequest) (*apiv1.GetSlotsResponse, error) {
	agent, ok := a.agentService.get(agentID(msg.AgentId))
	if !ok {
		return nil, api.NotFoundErrs("agent", msg.AgentId, true)
	}
	return agent.GetSlots(msg), nil
}

// IsReattachableOnlyAfterStarted implements rm.ResourceManager.
func (*ResourceManager) IsReattachableOnlyAfterStarted() bool {
	return true
}

// MoveJob implements rm.ResourceManager.
func (a *ResourceManager) MoveJob(msg sproto.MoveJob) error {
	pool, err := a.poolByName(msg.ResourcePool)
	if err != nil {
		return fmt.Errorf("move job found no resource pool with name %s: %w", msg.ResourcePool, err)
	}
	return pool.MoveJob(msg)
}

// NotifyContainerRunning implements rm.ResourceManager.
func (*ResourceManager) NotifyContainerRunning(sproto.NotifyContainerRunning) error {
	// Agent Resource Manager does not implement a handler for the
	// NotifyContainerRunning message, as it is only used on HPC
	// (High Performance Computing).
	return rmerrors.ErrNotSupported
}

// RecoverJobPosition implements rm.ResourceManager.
func (a *ResourceManager) RecoverJobPosition(msg sproto.RecoverJobPosition) {
	pool, err := a.poolByName(msg.ResourcePool)
	if err != nil {
		a.syslog.WithError(err).Error("recovering job position")
		return
	}
	pool.RecoverJobPosition(msg)
}

// Release implements rm.ResourceManager.
func (a *ResourceManager) Release(msg sproto.ResourcesReleased) {
	pool, err := a.poolByName(msg.ResourcePool)
	if err != nil {
		a.syslog.WithError(err).Warnf("release found no resource pool with name %s",
			msg.ResourcePool)
		return
	}
	pool.ResourcesReleased(msg)
}

// ResolveResourcePool implements rm.ResourceManager.
func (a *ResourceManager) ResolveResourcePool(name string, workspaceID int, slots int) (string, error) {
	ctx := context.TODO()
	defaultComputePool, defaultAuxPool, err := db.GetDefaultPoolsForWorkspace(ctx, workspaceID)
	if err != nil {
		return "", err
	}
	// If the resource pool isn't set, fill in the default at creation time.
	if name == "" && slots == 0 {
		if defaultAuxPool == "" {
			req := sproto.GetDefaultAuxResourcePoolRequest{}
			resp, err := a.GetDefaultAuxResourcePool(req)
			if err != nil {
				return "", fmt.Errorf("defaulting to aux pool: %w", err)
			}
			return resp.PoolName, nil
		}
		name = defaultAuxPool
	}

	if name == "" && slots >= 0 {
		if defaultComputePool == "" {
			req := sproto.GetDefaultComputeResourcePoolRequest{}
			resp, err := a.GetDefaultComputeResourcePool(req)
			if err != nil {
				return "", fmt.Errorf("defaulting to compute pool: %w", err)
			}
			return resp.PoolName, nil
		}
		name = defaultComputePool
	}

	resp, err := a.GetResourcePools(&apiv1.GetResourcePoolsRequest{})
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

	if err := a.ValidateResourcePool(name); err != nil {
		return "", fmt.Errorf("validating pool: %w", err)
	}
	return name, nil
}

// SetAllocationName implements rm.ResourceManager.
func (a *ResourceManager) SetAllocationName(msg sproto.SetAllocationName) {
	pool, err := a.poolByName(msg.ResourcePool)
	if err != nil {
		a.syslog.WithError(err).Warnf("set allocation name found no resource pool with name %s",
			msg.ResourcePool)
		return
	}
	// In the actor system, this was a tell before, so the `go` is to keep the same structure.  I'm not changing it
	// out of principle during the refactor but removing it is very likely fine, just check for deadlocks.
	go pool.SetAllocationName(msg)
}

// SetGroupMaxSlots implements rm.ResourceManager.
func (a *ResourceManager) SetGroupMaxSlots(msg sproto.SetGroupMaxSlots) {
	pool, err := a.poolByName(msg.ResourcePool)
	if err != nil {
		a.syslog.WithError(err).Warnf("set group max slots found no resource pool with name %s",
			msg.ResourcePool)
		return
	}
	// In the actor system, this was a tell before, so the `go` is to keep the same structure.  I'm not changing it
	// out of principle during the refactor but removing it is very likely fine, just check for deadlocks.
	go pool.SetGroupMaxSlots(msg)
}

// SetGroupPriority implements rm.ResourceManager.
func (a *ResourceManager) SetGroupPriority(msg sproto.SetGroupPriority) error {
	pool, err := a.poolByName(msg.ResourcePool)
	if err != nil {
		return fmt.Errorf("set group priority found no resource pool with name %s: %w",
			msg.ResourcePool, err)
	}
	return pool.SetGroupPriority(msg)
}

// SetGroupWeight implements rm.ResourceManager.
func (a *ResourceManager) SetGroupWeight(msg sproto.SetGroupWeight) error {
	pool, err := a.poolByName(msg.ResourcePool)
	if err != nil {
		return fmt.Errorf("set group weight found no resource pool with name %s: %w",
			msg.ResourcePool, err)
	}
	pool.SetGroupWeight(msg)
	return nil
}

// TaskContainerDefaults implements rm.ResourceManager.
func (a *ResourceManager) TaskContainerDefaults(
	resourcePoolName string,
	fallbackConfig model.TaskContainerDefaultsConfig,
) (model.TaskContainerDefaultsConfig, error) {
	result := fallbackConfig
	// Iterate through configured pools looking for a TaskContainerDefaults setting.
	for _, pool := range a.poolsConfig {
		if resourcePoolName == pool.PoolName {
			if pool.TaskContainerDefaults == nil {
				break
			}
			result = *pool.TaskContainerDefaults
		}
	}
	return result, nil
}

// ValidateCommandResources implements rm.ResourceManager.
func (a *ResourceManager) ValidateCommandResources(
	msg sproto.ValidateCommandResourcesRequest,
) (sproto.ValidateCommandResourcesResponse, error) {
	pool, err := a.poolByName(msg.ResourcePool)
	if err != nil {
		a.syslog.WithError(err).Error("recovering job position")
		return sproto.ValidateCommandResourcesResponse{}, err
	}
	return pool.ValidateCommandResources(msg), nil
}

// ValidateResourcePool implements rm.ResourceManager.
func (a *ResourceManager) ValidateResourcePool(name string) error {
	_, err := a.poolByName(name)
	if err != nil {
		return err
	}
	return nil
}

// ValidateResourcePoolAvailability implements rm.ResourceManager.
func (a *ResourceManager) ValidateResourcePoolAvailability(
	v *sproto.ValidateResourcePoolAvailabilityRequest,
) ([]command.LaunchWarning, error) {
	if v.Slots == 0 {
		return nil, nil
	}

	switch exceeded, err := a.CheckMaxSlotsExceeded(v); {
	case err != nil:
		return nil, fmt.Errorf("validating request for (%s, %d): %w", v.Name, v.Slots, err)
	case exceeded:
		return []command.LaunchWarning{command.CurrentSlotsExceeded}, nil
	default:
		return nil, nil
	}
}

// ValidateResources implements rm.ResourceManager.
func (a *ResourceManager) ValidateResources(name string, slots int, command bool) error {
	// TODO: Replace this function usage with ValidateCommandResources
	if slots > 0 && command {
		switch resp, err := a.ValidateCommandResources(
			sproto.ValidateCommandResourcesRequest{
				ResourcePool: name,
				Slots:        slots,
			}); {
		case err != nil:
			return fmt.Errorf("validating request for (%s, %d): %w", name, slots, err)
		case !resp.Fulfillable:
			return errors.New("request unfulfillable, please try requesting less slots")
		}
	}
	return nil
}

func (a *ResourceManager) createResourcePool(
	db db.DB, config config.ResourcePoolConfig, cert *tls.Certificate,
) (*resourcePool, error) {
	a.syslog.Infof("creating resource pool: %s", config.PoolName)

	// We pass the config here in by value so that in the case where we replace
	// the scheduler config with the global scheduler config (when the pool does
	// not define one for itself) we do not modify the original data structures.
	if config.Scheduler != nil {
		a.syslog.Infof("pool %s using local scheduling config", config.PoolName)
	} else {
		config.Scheduler = a.config.Scheduler
		a.syslog.Infof("pool %s using global scheduling config", config.PoolName)
	}

	return newResourcePool(
		&config,
		db,
		cert,
		MakeScheduler(config.Scheduler),
		MakeFitFunction(config.Scheduler.FittingPolicy),
		a.agentService,
	)
}

func (a *ResourceManager) poolByName(name string) (*resourcePool, error) {
	if name == "" {
		return nil, errors.New("invalid call: cannot get a resource pool with no name")
	}
	pool, ok := a.pools[name]
	if !ok {
		return nil, fmt.Errorf("cannot find resource pool %s", name)
	}
	return pool, nil
}

func (a *ResourceManager) getPoolJobStats(poolConfig config.ResourcePoolConfig) (*jobv1.QueueStats, error) {
	pool, err := a.poolByName(poolConfig.PoolName)
	if err != nil {
		return nil, err
	}
	return pool.GetJobQStats(sproto.GetJobQStats{}), nil
}

func (a *ResourceManager) getResourcePoolConfig(poolName string) (
	config.ResourcePoolConfig, error,
) {
	for i := range a.poolsConfig {
		if a.poolsConfig[i].PoolName == poolName {
			return a.poolsConfig[i], nil
		}
	}
	return config.ResourcePoolConfig{}, errors.Errorf("cannot find resource pool %s", poolName)
}

func (a *ResourceManager) createResourcePoolSummary(
	poolName string,
) (*resourcepoolv1.ResourcePool, error) {
	pool, err := a.getResourcePoolConfig(poolName)
	if err != nil {
		return &resourcepoolv1.ResourcePool{}, err
	}

	// Hide secrets.
	pool = pool.Printable()

	// Static Pool defaults
	poolType := resourcepoolv1.ResourcePoolType_RESOURCE_POOL_TYPE_STATIC
	preemptible := false
	location := "on-prem"
	imageID := ""
	instanceType := ""
	slotsPerAgent := -1
	slotType := device.ZeroSlot
	accelerator := ""

	if pool.Provider != nil {
		if pool.Provider.AWS != nil {
			poolType = resourcepoolv1.ResourcePoolType_RESOURCE_POOL_TYPE_AWS
			preemptible = pool.Provider.AWS.SpotEnabled
			location = pool.Provider.AWS.Region
			imageID = pool.Provider.AWS.ImageID
			instanceType = string(pool.Provider.AWS.InstanceType)
			slotsPerAgent = pool.Provider.AWS.SlotsPerInstance()
			slotType = pool.Provider.AWS.SlotType()
			accelerator = pool.Provider.AWS.Accelerator()
		}
		if pool.Provider.GCP != nil {
			poolType = resourcepoolv1.ResourcePoolType_RESOURCE_POOL_TYPE_GCP
			preemptible = pool.Provider.GCP.InstanceType.Preemptible
			location = pool.Provider.GCP.Zone
			imageID = pool.Provider.GCP.BootDiskSourceImage
			slotsPerAgent = pool.Provider.GCP.SlotsPerInstance()
			slotType = pool.Provider.GCP.SlotType()
			instanceType = pool.Provider.GCP.InstanceType.MachineType
			if pool.Provider.GCP.InstanceType.GPUNum > 0 {
				accelerator = pool.Provider.GCP.Accelerator()
			}
		}
	}

	var schedulerType resourcepoolv1.SchedulerType
	if pool.Scheduler == nil {
		// This means the scheduler setting should be inherited from the resource manager
		pool.Scheduler = a.config.Scheduler
		if a.config.Scheduler == nil {
			a.syslog.Errorf("scheduler is not present in config or in resource manager")
			return &resourcepoolv1.ResourcePool{}, err
		}
	}

	if pool.Scheduler.FairShare != nil {
		schedulerType = resourcepoolv1.SchedulerType_SCHEDULER_TYPE_FAIR_SHARE
	}
	if pool.Scheduler.Priority != nil {
		schedulerType = resourcepoolv1.SchedulerType_SCHEDULER_TYPE_PRIORITY
	}
	if pool.Scheduler.RoundRobin != nil {
		schedulerType = resourcepoolv1.SchedulerType_SCHEDULER_TYPE_ROUND_ROBIN
	}

	resp := &resourcepoolv1.ResourcePool{
		Name:                         pool.PoolName,
		Description:                  pool.Description,
		Type:                         poolType,
		DefaultAuxPool:               a.config.DefaultAuxResourcePool == poolName,
		DefaultComputePool:           a.config.DefaultComputeResourcePool == poolName,
		Preemptible:                  preemptible,
		SlotsPerAgent:                int32(slotsPerAgent),
		AuxContainerCapacityPerAgent: int32(pool.MaxAuxContainersPerAgent),
		SchedulerType:                schedulerType,
		Location:                     location,
		ImageId:                      imageID,
		InstanceType:                 instanceType,
		Details:                      &resourcepoolv1.ResourcePoolDetail{},
		SlotType:                     slotType.Proto(),
		Accelerator:                  accelerator,
	}
	if pool.Provider != nil {
		resp.MinAgents = int32(pool.Provider.MinInstances)
		resp.MaxAgents = int32(pool.Provider.MaxInstances)
		resp.MasterUrl = pool.Provider.MasterURL
		resp.MasterCertName = pool.Provider.MasterCertName
		resp.StartupScript = pool.Provider.StartupScript
		resp.ContainerStartupScript = pool.Provider.ContainerStartupScript
		resp.AgentDockerNetwork = pool.Provider.AgentDockerNetwork
		resp.AgentDockerRuntime = pool.Provider.AgentDockerRuntime
		resp.AgentDockerImage = pool.Provider.AgentDockerImage
		resp.MaxIdleAgentPeriod = float32(time.Duration(pool.Provider.MaxIdleAgentPeriod).Seconds())
		startingPeriodSecs := time.Duration(pool.Provider.MaxAgentStartingPeriod).Seconds()
		resp.MaxAgentStartingPeriod = float32(startingPeriodSecs)
	}
	if pool.Scheduler != nil {
		if pool.Scheduler.FittingPolicy == best {
			resp.SchedulerFittingPolicy = resourcepoolv1.FittingPolicy_FITTING_POLICY_BEST
		}
		if pool.Scheduler.FittingPolicy == worst {
			resp.SchedulerFittingPolicy = resourcepoolv1.FittingPolicy_FITTING_POLICY_WORST
		}

		if pool.Scheduler.FittingPolicy != best && pool.Scheduler.FittingPolicy != worst {
			a.syslog.Errorf("unrecognized scheduler fitting policy")
			return &resourcepoolv1.ResourcePool{}, err
		}
	}
	if poolType == resourcepoolv1.ResourcePoolType_RESOURCE_POOL_TYPE_AWS {
		aws := pool.Provider.AWS
		resp.Details.Aws = &resourcepoolv1.ResourcePoolAwsDetail{
			Region:                aws.Region,
			RootVolumeSize:        int32(aws.RootVolumeSize),
			ImageId:               aws.ImageID,
			TagKey:                aws.TagKey,
			TagValue:              aws.TagValue,
			InstanceName:          aws.InstanceName,
			SshKeyName:            aws.SSHKeyName,
			PublicIp:              aws.NetworkInterface.PublicIP,
			SubnetId:              aws.NetworkInterface.SubnetID,
			SecurityGroupId:       aws.NetworkInterface.SecurityGroupID,
			IamInstanceProfileArn: aws.IamInstanceProfileArn,
			InstanceType:          string(aws.InstanceType),
			LogGroup:              aws.LogGroup,
			LogStream:             aws.LogStream,
			SpotEnabled:           aws.SpotEnabled,
			SpotMaxPrice:          aws.SpotMaxPrice,
		}
		customTags := make([]*resourcepoolv1.AwsCustomTag, len(aws.CustomTags))
		for i, tagInfo := range aws.CustomTags {
			customTags[i] = &resourcepoolv1.AwsCustomTag{
				Key:   tagInfo.Key,
				Value: tagInfo.Value,
			}
		}
		resp.Details.Aws.CustomTags = customTags
	}
	if poolType == resourcepoolv1.ResourcePoolType_RESOURCE_POOL_TYPE_GCP {
		gcp := pool.Provider.GCP
		resp.Details.Gcp = &resourcepoolv1.ResourcePoolGcpDetail{
			Project:                gcp.Project,
			Zone:                   gcp.Zone,
			BootDiskSize:           int32(gcp.BootDiskSize),
			BootDiskSourceImage:    gcp.BootDiskSourceImage,
			LabelKey:               gcp.LabelKey,
			LabelValue:             gcp.LabelValue,
			NamePrefix:             gcp.NamePrefix,
			Network:                gcp.NetworkInterface.Network,
			Subnetwork:             gcp.NetworkInterface.Subnetwork,
			ExternalIp:             gcp.NetworkInterface.ExternalIP,
			NetworkTags:            gcp.NetworkTags,
			ServiceAccountEmail:    gcp.ServiceAccount.Email,
			ServiceAccountScopes:   gcp.ServiceAccount.Scopes,
			MachineType:            gcp.InstanceType.MachineType,
			GpuType:                gcp.InstanceType.GPUType,
			GpuNum:                 int32(gcp.InstanceType.GPUNum),
			Preemptible:            gcp.InstanceType.Preemptible,
			OperationTimeoutPeriod: float32(time.Duration(gcp.OperationTimeoutPeriod).Seconds()),
		}
	}

	if schedulerType == resourcepoolv1.SchedulerType_SCHEDULER_TYPE_PRIORITY {
		resp.Details.PriorityScheduler = &resourcepoolv1.ResourcePoolPrioritySchedulerDetail{
			Preemption:      pool.Scheduler.Priority.Preemption,
			DefaultPriority: int32(*pool.Scheduler.Priority.DefaultPriority),
		}
	}

	rp, err := a.poolByName(poolName)
	if err != nil {
		return nil, err
	}
	resourceSummary := rp.GetResourceSummary()

	resp.NumAgents = int32(resourceSummary.numAgents)
	resp.SlotsAvailable = int32(resourceSummary.numTotalSlots)
	resp.SlotsUsed = int32(resourceSummary.numActiveSlots)
	resp.AuxContainerCapacity = int32(resourceSummary.maxNumAuxContainers)
	resp.AuxContainersRunning = int32(resourceSummary.numActiveAuxContainers)
	if pool.Provider == nil && resp.NumAgents > 0 {
		resp.SlotType = resourceSummary.slotType.Proto()
	}

	return resp, nil
}

func (a *ResourceManager) fetchAvgQueuedTime(pool string) (
	[]*jobv1.AggregateQueueStats, error,
) {
	aggregates := []model.ResourceAggregates{}
	err := db.Bun().NewSelect().Model(&aggregates).
		Where("aggregation_type = ?", "queued").
		Where("aggregation_key = ?", pool).
		Where("date >= CURRENT_TIMESTAMP - interval '30 days'").
		Order("date ASC").Scan(context.TODO())
	if err != nil {
		return nil, err
	}
	res := make([]*jobv1.AggregateQueueStats, 0)
	for _, record := range aggregates {
		res = append(res, &jobv1.AggregateQueueStats{
			PeriodStart: record.Date.Format("2006-01-02"),
			Seconds:     record.Seconds,
		})
	}
	today := float32(0)
	subq := db.Bun().NewSelect().TableExpr("allocations").Column("allocation_id").
		Where("resource_pool = ?", pool).
		Where("start_time >= CURRENT_DATE")
	err = db.Bun().NewSelect().TableExpr("task_stats").ColumnExpr(
		"avg(extract(epoch FROM end_time - start_time))",
	).Where("event_type = ?", "QUEUED").
		Where("end_time >= CURRENT_DATE AND allocation_id IN (?) ", subq).
		Scan(context.TODO(), &today)
	if err != nil {
		return nil, err
	}
	res = append(res, &jobv1.AggregateQueueStats{
		PeriodStart: time.Now().Format("2006-01-02"),
		Seconds:     today,
	})
	return res, nil
}

// mostly for tests.
func (a *ResourceManager) stop() {
	for _, pool := range a.pools {
		pool.stop()
	}
}
