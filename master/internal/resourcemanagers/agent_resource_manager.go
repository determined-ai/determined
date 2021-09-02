package resourcemanagers

import (
	"crypto/tls"
	"fmt"
	"time"

	"github.com/determined-ai/determined/master/pkg/model"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/device"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/resourcepoolv1"
)

type agentResourceManager struct {
	config      *AgentResourceManagerConfig
	poolsConfig []ResourcePoolConfig
	cert        *tls.Certificate

	pools map[string]*actor.Ref
}

func newAgentResourceManager(config *ResourceConfig, cert *tls.Certificate) *agentResourceManager {
	return &agentResourceManager{
		config:      config.ResourceManager.AgentRM,
		poolsConfig: config.ResourcePools,
		cert:        cert,
		pools:       make(map[string]*actor.Ref),
	}
}

func (a *agentResourceManager) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart:
		for ix, config := range a.poolsConfig {
			rpRef := a.createResourcePool(ctx, a.poolsConfig[ix], a.cert)
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

	case sproto.SetGroupMaxSlots, sproto.SetGroupWeight, sproto.SetGroupPriority:
		a.forwardToAllPools(ctx, msg)

	case sproto.GetTaskHandler:
		if handler, err := a.aggregateTaskHandler(a.forwardToAllPools(ctx, msg)); err != nil {
			ctx.Respond(err)
		} else {
			ctx.Respond(handler)
		}

	case sproto.GetTaskSummary:
		if summary := a.aggregateTaskSummary(a.forwardToAllPools(ctx, msg)); summary != nil {
			ctx.Respond(summary)
		}

	case sproto.GetTaskSummaries:
		ctx.Respond(a.aggregateTaskSummaries(a.forwardToAllPools(ctx, msg)))

	case sproto.SetTaskName:
		a.forwardToAllPools(ctx, msg)

	case sproto.GetDefaultComputeResourcePoolRequest:
		ctx.Respond(sproto.GetDefaultComputeResourcePoolResponse{
			PoolName: a.config.DefaultComputeResourcePool,
		})

	case sproto.GetDefaultAuxResourcePoolRequest:
		ctx.Respond(sproto.GetDefaultAuxResourcePoolResponse{PoolName: a.config.DefaultAuxResourcePool})

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
			summaries = append(summaries, summary)
		}
		resp := &apiv1.GetResourcePoolsResponse{ResourcePools: summaries}
		ctx.Respond(resp)

	default:
		return actor.ErrUnexpectedMessage(ctx)
	}
	return nil
}

func (a *agentResourceManager) createResourcePool(
	ctx *actor.Context, config ResourcePoolConfig, cert *tls.Certificate,
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

	rp := NewResourcePool(
		&config,
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

func (a *agentResourceManager) aggregateTaskHandler(
	resps map[*actor.Ref]actor.Message,
) (*actor.Ref, error) {
	for _, resp := range resps {
		if typed, ok := resp.(*actor.Ref); ok && typed != nil {
			return typed, nil
		}
	}
	return nil, errors.New("task handler not found on any resource pool")
}

func (a *agentResourceManager) aggregateTaskSummary(
	resps map[*actor.Ref]actor.Message,
) *TaskSummary {
	for _, resp := range resps {
		if resp != nil {
			typed := resp.(TaskSummary)
			return &typed
		}
	}
	return nil
}

func (a *agentResourceManager) aggregateTaskSummaries(
	resps map[*actor.Ref]actor.Message,
) map[model.AllocationID]TaskSummary {
	summaries := make(map[model.AllocationID]TaskSummary)
	for _, resp := range resps {
		if resp != nil {
			typed := resp.(map[model.AllocationID]TaskSummary)
			for id, summary := range typed {
				summaries[id] = summary
			}
		}
	}
	return summaries
}

func (a *agentResourceManager) getResourcePoolConfig(poolName string) (ResourcePoolConfig, error) {
	for i := range a.poolsConfig {
		if a.poolsConfig[i].PoolName == poolName {
			return a.poolsConfig[i], nil
		}
	}
	return ResourcePoolConfig{}, errors.Errorf("cannot find resource pool %s", poolName)
}

func (a *agentResourceManager) createResourcePoolSummary(
	ctx *actor.Context,
	poolName string,
) (*resourcepoolv1.ResourcePool, error) {
	pool, err := a.getResourcePoolConfig(poolName)
	if err != nil {
		return &resourcepoolv1.ResourcePool{}, err
	}

	// Static Pool defaults
	poolType := resourcepoolv1.ResourcePoolType_RESOURCE_POOL_TYPE_STATIC
	preemptible := false
	location := "on-prem"
	imageID := ""
	instanceType := ""
	slotsPerAgent := -1
	slotType := device.ZeroSlot

	if pool.Provider != nil {
		if pool.Provider.AWS != nil {
			poolType = resourcepoolv1.ResourcePoolType_RESOURCE_POOL_TYPE_AWS
			preemptible = pool.Provider.AWS.SpotEnabled
			location = pool.Provider.AWS.Region
			imageID = pool.Provider.AWS.ImageID
			instanceType = string(pool.Provider.AWS.InstanceType)
			slotsPerAgent = pool.Provider.AWS.SlotsPerInstance()
			slotType = pool.Provider.AWS.SlotType()
		}
		if pool.Provider.GCP != nil {
			poolType = resourcepoolv1.ResourcePoolType_RESOURCE_POOL_TYPE_GCP
			preemptible = pool.Provider.GCP.InstanceType.Preemptible
			location = pool.Provider.GCP.Zone
			imageID = pool.Provider.GCP.BootDiskSourceImage
			slotsPerAgent = pool.Provider.GCP.SlotsPerInstance()
			slotType = pool.Provider.GCP.SlotType()
			if pool.Provider.GCP.InstanceType.GPUNum == 0 {
				instanceType = pool.Provider.GCP.InstanceType.MachineType
			} else {
				instanceType = fmt.Sprintf("%s, %d x %s",
					pool.Provider.GCP.InstanceType.MachineType,
					pool.Provider.GCP.InstanceType.GPUNum,
					pool.Provider.GCP.InstanceType.GPUType,
				)
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
		resp.AgentFluentImage = pool.Provider.AgentFluentImage
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

	resourceSummary := ctx.Ask(a.pools[poolName], GetResourceSummary{}).Get().(ResourceSummary)
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
