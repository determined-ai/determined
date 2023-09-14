package agentrm

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/rm/actorrm"
	"github.com/determined-ai/determined/master/internal/rm/rmerrors"
	"github.com/determined-ai/determined/master/internal/rm/rmutils"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/aproto"
	"github.com/determined-ai/determined/master/pkg/command"
	"github.com/determined-ai/determined/master/pkg/device"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/jobv1"
	"github.com/determined-ai/determined/proto/pkg/resourcepoolv1"
)

const (
	best  = "best"
	worst = "worst"
)

// ResourceManager is a resource manager for Determined-managed resources.
type ResourceManager struct {
	*actorrm.ResourceManager
}

// New returns a new ResourceManager, which manages communicating with
// and scheduling on Determined agents.
func New(
	system *actor.System,
	db *db.PgDB,
	echo *echo.Echo,
	config *config.ResourceConfig,
	opts *aproto.MasterSetAgentOptions,
	cert *tls.Certificate,
) ResourceManager {
	ref, _ := system.ActorOf(
		sproto.AgentRMAddr,
		newAgentResourceManager(db, config, cert),
	)
	system.Ask(ref, actor.Ping{}).Get()
	rm := ResourceManager{ResourceManager: actorrm.Wrap(ref)}
	initializeAgents(system, rm, echo, opts)
	return rm
}

// GetResourcePoolRef gets an actor ref to a resource pool by name.
func (a ResourceManager) GetResourcePoolRef(
	ctx actor.Messenger,
	name string,
) (*actor.Ref, error) {
	rp := a.Ref().Child(name)
	if rp == nil {
		return nil, fmt.Errorf("cannot find resource pool: %s", name)
	}
	return rp, nil
}

// ValidateResourcePool validates existence of a resource pool.
func (a ResourceManager) ValidateResourcePool(ctx actor.Messenger, name string) error {
	_, err := a.GetResourcePoolRef(ctx, name)
	return err
}

// CheckMaxSlotsExceeded checks if the job exceeded the maximum number of slots.
func (a ResourceManager) CheckMaxSlotsExceeded(
	ctx actor.Messenger, name string, slots int,
) (bool, error) {
	ref, err := a.GetResourcePoolRef(ctx, name)
	if err != nil {
		return false, err
	}
	resp := ref.System().Ask(ref, sproto.CapacityCheck{
		Slots: slots,
	})
	if resp.Error() != nil {
		return false, resp.Error()
	}
	return resp.Get().(sproto.CapacityCheckResponse).CapacityExceeded, nil
}

// ResolveResourcePool fully resolves the resource pool name.
func (a ResourceManager) ResolveResourcePool(
	actorCtx actor.Messenger, name string, workspaceID, slots int,
) (string, error) {
	ctx := context.TODO()
	defaultComputePool, defaultAuxPool, err := db.GetDefaultPoolsForWorkspace(ctx, workspaceID)
	if err != nil {
		return "", err
	}
	// If the resource pool isn't set, fill in the default at creation time.
	if name == "" && slots == 0 {
		if defaultAuxPool == "" {
			req := sproto.GetDefaultAuxResourcePoolRequest{}
			resp, err := a.GetDefaultAuxResourcePool(actorCtx, req)
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
			resp, err := a.GetDefaultComputeResourcePool(actorCtx, req)
			if err != nil {
				return "", fmt.Errorf("defaulting to compute pool: %w", err)
			}
			return resp.PoolName, nil
		}
		name = defaultComputePool
	}

	resp, err := a.GetResourcePools(actorCtx, &apiv1.GetResourcePoolsRequest{})
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

	if err := a.ValidateResourcePool(actorCtx, name); err != nil {
		return "", fmt.Errorf("validating pool: %w", err)
	}
	return name, nil
}

// ValidateResources ensures enough resources are available for a command.
func (a ResourceManager) ValidateResources(
	ctx actor.Messenger, name string, slots int, command bool,
) error {
	// TODO: Replace this function usage with ValidateCommandResources
	if slots > 0 && command {
		switch resp, err := a.ValidateCommandResources(ctx,
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

// ValidateResourcePoolAvailability is a default implementation to satisfy the interface.
func (a ResourceManager) ValidateResourcePoolAvailability(ctx actor.Messenger,
	name string, slots int) (
	[]command.LaunchWarning,
	error,
) {
	if slots == 0 {
		return nil, nil
	}

	switch exceeded, err := a.CheckMaxSlotsExceeded(ctx, name, slots); {
	case err != nil:
		return nil, fmt.Errorf("validating request for (%s, %d): %w", name, slots, err)
	case exceeded:
		return []command.LaunchWarning{command.CurrentSlotsExceeded}, nil
	default:
		return nil, nil
	}
}

// GetAgents gets the state of connected agents. Go around the RM and directly to the agents actor
// to avoid blocking asks through it.
func (a ResourceManager) GetAgents(
	ctx actor.Messenger,
	msg *apiv1.GetAgentsRequest,
) (resp *apiv1.GetAgentsResponse, err error) {
	return resp, actorrm.AskAt(a.Ref().System(), sproto.AgentsAddr, msg, &resp)
}

type agentResourceManager struct {
	config      *config.AgentResourceManagerConfig
	poolsConfig []config.ResourcePoolConfig
	cert        *tls.Certificate
	db          *db.PgDB

	pools map[string]*actor.Ref
}

func newAgentResourceManager(db *db.PgDB, config *config.ResourceConfig,
	cert *tls.Certificate,
) *agentResourceManager {
	return &agentResourceManager{
		config:      config.ResourceManager.AgentRM,
		poolsConfig: config.ResourcePools,
		cert:        cert,
		db:          db,
		pools:       make(map[string]*actor.Ref),
	}
}

func (a *agentResourceManager) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart:
		for ix, config := range a.poolsConfig {
			rpRef := a.createResourcePool(ctx, a.db, a.poolsConfig[ix], a.cert)
			if rpRef != nil {
				a.pools[config.PoolName] = rpRef
			}
		}

	case sproto.AllocateRequest:
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
		a.forwardToPool(ctx, msg.ResourcePool, msg)

	case sproto.ResourcesReleased:
		a.forwardToAllPools(ctx, msg)

	case sproto.SetGroupMaxSlots, sproto.SetGroupWeight, sproto.SetGroupPriority,
		sproto.MoveJob:
		a.forwardToAllPools(ctx, msg)

	case sproto.PendingPreemption:
		ctx.Respond(actor.ErrUnexpectedMessage(ctx))
		return nil

	case sproto.DeleteJob:
		// For now, there is nothing to cleanup in determined-agents world.
		ctx.Respond(sproto.EmptyDeleteJobResponse())

	case sproto.RecoverJobPosition:
		a.forwardToPool(ctx, msg.ResourcePool, msg)

	case sproto.GetAllocationSummary:
		if summary := a.aggregateTaskSummary(a.forwardToAllPools(ctx, msg)); summary != nil {
			ctx.Respond(summary)
		}

	case sproto.GetAllocationSummaries:
		ctx.Respond(a.aggregateTaskSummaries(a.forwardToAllPools(ctx, msg)))

	case sproto.SetAllocationName:
		a.forwardToAllPools(ctx, msg)

	case sproto.GetDefaultComputeResourcePoolRequest:
		if a.config.DefaultComputeResourcePool == "" {
			ctx.Respond(rmerrors.ErrNoDefaultResourcePool)
		} else {
			ctx.Respond(sproto.GetDefaultComputeResourcePoolResponse{
				PoolName: a.config.DefaultComputeResourcePool,
			})
		}

	case sproto.GetDefaultAuxResourcePoolRequest:
		if a.config.DefaultAuxResourcePool == "" {
			ctx.Respond(rmerrors.ErrNoDefaultResourcePool)
		} else {
			ctx.Respond(sproto.GetDefaultAuxResourcePoolResponse{PoolName: a.config.DefaultAuxResourcePool})
		}

	case sproto.ValidateCommandResourcesRequest:
		a.forwardToPool(ctx, msg.ResourcePool, msg)

	case *apiv1.GetResourcePoolsRequest:
		summaries := make([]*resourcepoolv1.ResourcePool, 0, len(a.poolsConfig))
		for _, pool := range a.poolsConfig {
			summary, err := a.createResourcePoolSummary(ctx, pool.PoolName)
			if err != nil {
				// Should only raise an error if the resource pool doesn't exist and that can't happen.
				// But best to handle it anyway in case the implementation changes in the future.
				ctx.Log().WithError(err).Error("")
				ctx.Respond(err)
			}

			jobStats, err := a.getPoolJobStats(ctx, pool)
			if err != nil {
				ctx.Respond(err)
			}

			summary.Stats = jobStats
			summaries = append(summaries, summary)
		}
		resp := &apiv1.GetResourcePoolsResponse{ResourcePools: summaries}
		ctx.Respond(resp)
	case *apiv1.GetAgentsRequest:
		response := ctx.Self().System().AskAt(sproto.AgentsAddr, msg)
		ctx.Respond(response.Get())

	case sproto.GetJobQ:
		if msg.ResourcePool == "" {
			msg.ResourcePool = a.config.DefaultComputeResourcePool
		}

		rpRef := ctx.Child(msg.ResourcePool)
		if rpRef == nil {
			ctx.Respond(errors.Errorf("resource pool %s not found", msg.ResourcePool))
			return nil
		}
		resp := ctx.Ask(rpRef, msg).Get()
		ctx.Respond(resp)

	case *apiv1.GetJobQueueStatsRequest:
		resp := &apiv1.GetJobQueueStatsResponse{
			Results: make([]*apiv1.RPQueueStat, 0),
		}
		rpRefs := make([]*actor.Ref, 0)
		if len(msg.ResourcePools) == 0 {
			rpRefs = append(rpRefs, ctx.Children()...)
		} else {
			for _, rp := range msg.ResourcePools {
				rpRefs = append(rpRefs, ctx.Child(rp))
			}
		}

		actorResps := ctx.AskAll(sproto.GetJobQStats{}, rpRefs...).GetAll()
		for _, rpRef := range rpRefs {
			poolName := rpRef.Address().Local()
			qStats := apiv1.RPQueueStat{ResourcePool: poolName}
			aResp := actorResps[rpRef]
			switch aMsg := aResp.(type) {
			case error:
				ctx.Log().WithError(aMsg).Error("")
				ctx.Respond(aMsg)
				return nil
			case *jobv1.QueueStats:
				qStats.Stats = aMsg
				aggregates, err := a.fetchAvgQueuedTime(poolName)
				if err != nil {
					return fmt.Errorf("fetch average queued time: %s", err)
				}
				qStats.Aggregates = aggregates
				resp.Results = append(resp.Results, &qStats)
			default:
				return fmt.Errorf("unexpected response type: %T", aMsg)
			}
		}
		ctx.Respond(resp)
		return nil

	case sproto.GetJobQStats:
		resp := ctx.Ask(ctx.Child(msg.ResourcePool), msg).Get()
		ctx.Respond(resp)
	case taskContainerDefaults:
		ctx.Respond(a.getTaskContainerDefaults(msg))

	case sproto.GetExternalJobs:
		ctx.Respond(rmerrors.ErrNotSupported)

	default:
		return actor.ErrUnexpectedMessage(ctx)
	}
	return nil
}

func (a *agentResourceManager) getTaskContainerDefaults(
	msg taskContainerDefaults,
) model.TaskContainerDefaultsConfig {
	result := msg.fallbackDefault
	// Iterate through configured pools looking for a TaskContainerDefaults setting.
	for _, pool := range a.poolsConfig {
		if msg.resourcePool == pool.PoolName {
			if pool.TaskContainerDefaults == nil {
				break
			}
			result = *pool.TaskContainerDefaults
		}
	}
	return result
}

func (a *agentResourceManager) createResourcePool(
	ctx *actor.Context, db db.DB, config config.ResourcePoolConfig, cert *tls.Certificate,
) *actor.Ref {
	ctx.Log().Infof("creating resource pool: %s", config.PoolName)

	// We pass the config here in by value so that in the case where we replace
	// the scheduler config with the global scheduler config (when the pool does
	// not define one for itself) we do not modify the original data structures.
	if config.Scheduler != nil {
		ctx.Log().Infof("pool %s using local scheduling config", config.PoolName)
	} else {
		config.Scheduler = a.config.Scheduler
		ctx.Log().Infof("pool %s using global scheduling config", config.PoolName)
	}

	rp := newResourcePool(
		&config,
		db,
		cert,
		MakeScheduler(config.Scheduler),
		MakeFitFunction(config.Scheduler.FittingPolicy),
	)
	ref, ok := ctx.ActorOf(config.PoolName, rp)
	if !ok {
		ctx.Log().Errorf("cannot create resource pool actor: %s", config.PoolName)
		return nil
	}
	return ref
}

func (a *agentResourceManager) forwardToPool(
	ctx *actor.Context, resourcePool string, msg actor.Message,
) {
	if a.pools[resourcePool] == nil {
		sender := "unknown"
		if ctx.Sender() != nil {
			sender = ctx.Sender().Address().String()
		}
		err := errors.Errorf("cannot find resource pool %s for message %T from actor %s",
			resourcePool, ctx.Message(), sender)
		ctx.Log().WithError(err).Error("")
		if ctx.ExpectingResponse() {
			ctx.Respond(err)
		}
		return
	}
	if ctx.ExpectingResponse() {
		response := ctx.Ask(a.pools[resourcePool], msg)
		ctx.Respond(response.Get())
	} else {
		ctx.Tell(a.pools[resourcePool], msg)
	}
}

func (a *agentResourceManager) forwardToAllPools(
	ctx *actor.Context, msg actor.Message,
) map[*actor.Ref]actor.Message {
	if ctx.ExpectingResponse() {
		return ctx.AskAll(msg, ctx.Children()...).GetAll()
	}
	ctx.TellAll(msg, ctx.Children()...)
	return nil
}

func (a *agentResourceManager) getPoolJobStats(
	ctx *actor.Context, pool config.ResourcePoolConfig,
) (*jobv1.QueueStats, error) {
	jobStatsResp := ctx.Ask(a.pools[pool.PoolName], sproto.GetJobQStats{})
	if err := jobStatsResp.Error(); err != nil {
		return nil, fmt.Errorf("unexpected response type from jobStats: %s", err)
	}
	jobStats, ok := jobStatsResp.Get().(*jobv1.QueueStats)
	if !ok {
		return nil, fmt.Errorf("unexpected response type from jobStats")
	}
	return jobStats, nil
}

func (a *agentResourceManager) aggregateTaskSummary(
	resps map[*actor.Ref]actor.Message,
) *sproto.AllocationSummary {
	for _, resp := range resps {
		if resp != nil {
			typed := resp.(sproto.AllocationSummary)
			return &typed
		}
	}
	return nil
}

func (a *agentResourceManager) aggregateTaskSummaries(
	resps map[*actor.Ref]actor.Message,
) map[model.AllocationID]sproto.AllocationSummary {
	summaries := make(map[model.AllocationID]sproto.AllocationSummary)
	for _, resp := range resps {
		if resp != nil {
			typed := resp.(map[model.AllocationID]sproto.AllocationSummary)
			for id, summary := range typed {
				summaries[id] = summary
			}
		}
	}
	return summaries
}

func (a *agentResourceManager) getResourcePoolConfig(poolName string) (
	config.ResourcePoolConfig, error,
) {
	for i := range a.poolsConfig {
		if a.poolsConfig[i].PoolName == poolName {
			return a.poolsConfig[i], nil
		}
	}
	return config.ResourcePoolConfig{}, errors.Errorf("cannot find resource pool %s", poolName)
}

func (a *agentResourceManager) createResourcePoolSummary(
	ctx *actor.Context,
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
			ctx.Log().Errorf("scheduler is not present in config or in resource manager")
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
			ctx.Log().Errorf("unrecognized scheduler fitting policy")
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

	response := ctx.Ask(a.pools[poolName], getResourceSummary{})
	if response.Error() != nil {
		return &resourcepoolv1.ResourcePool{}, err
	}
	resourceSummary := response.Get().(resourceSummary)
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

func (a *agentResourceManager) fetchAvgQueuedTime(pool string) (
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

// NotifyContainerRunning receives a notification from the container to let
// the master know that the container is running.
func (a ResourceManager) NotifyContainerRunning(
	ctx actor.Messenger,
	msg sproto.NotifyContainerRunning,
) error {
	// Agent Resource Manager does not implement a handler for the
	// NotifyContainerRunning message, as it is only used on HPC
	// (High Performance Computing).
	return errors.New(
		"the NotifyContainerRunning message is unsupported for AgentResourceManager")
}

type taskContainerDefaults struct {
	fallbackDefault model.TaskContainerDefaultsConfig
	resourcePool    string
}

// TaskContainerDefaults returns TaskContainerDefaults for the specified pool.
func (a ResourceManager) TaskContainerDefaults(
	ctx actor.Messenger,
	pool string,
	fallbackConfig model.TaskContainerDefaultsConfig,
) (result model.TaskContainerDefaultsConfig, err error) {
	req := taskContainerDefaults{fallbackDefault: fallbackConfig, resourcePool: pool}
	return result, a.Ask(ctx, req, &result)
}
