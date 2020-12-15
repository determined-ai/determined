package resourcemanagers

import (
	"crypto/tls"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"strings"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	resourcepoolv1 "github.com/determined-ai/determined/proto/pkg/resourcepoolv1"
	"time"
)

type agentResourceManager struct {
	config      *AgentResourceManagerConfig
	poolsConfig *ResourcePoolsConfig
	cert        *tls.Certificate

	pools map[string]*actor.Ref
}

type GetResourcePoolSummary struct{
	resourcePool string
}

type GetResourcePoolSummaries struct{}

func newAgentResourceManager(
	config *AgentResourceManagerConfig, poolsConfig *ResourcePoolsConfig, cert *tls.Certificate,
) *agentResourceManager {
	return &agentResourceManager{
		config:      config,
		poolsConfig: poolsConfig,
		cert:        cert,
		pools:       make(map[string]*actor.Ref),
	}
}

func (a *agentResourceManager) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart:
		for ix, config := range a.poolsConfig.ResourcePools {
			rpRef := a.createResourcePool(ctx, a.poolsConfig.ResourcePools[ix], a.cert)
			if rpRef != nil {
				a.pools[config.PoolName] = rpRef
			}
		}

	case AllocateRequest:
		if len(msg.ResourcePool) == 0 {
			msg.ResourcePool = a.getDefaultResourcePool(msg)
		}
		a.forwardToPool(ctx, msg.ResourcePool, msg)
	case ResourcesReleased:
		a.forwardToAllPools(ctx, msg)

	case sproto.SetGroupMaxSlots, sproto.SetGroupWeight, sproto.SetGroupPriority:
		a.forwardToAllPools(ctx, msg)

	case GetTaskSummary:
		if summary := a.aggregateTaskSummary(a.forwardToAllPools(ctx, msg)); summary != nil {
			ctx.Respond(summary)
		}
	case GetTaskSummaries:
		ctx.Respond(a.aggregateTaskSummaries(a.forwardToAllPools(ctx, msg)))
	case SetTaskName:
		a.forwardToAllPools(ctx, msg)

	case apiv1.GetResourcePoolRequest:

		if a.pools[msg.ResourcePoolId] == nil {
			err := errors.Errorf("cannot find resource pool %s to summarize", msg.ResourcePoolId)
			ctx.Log().WithError(err).Error("")
			ctx.Respond(err)
			break
		}

		resourcePoolSummary, err := a.createResourcePoolSummary(ctx, msg.ResourcePoolId)
		if err != nil {
			// TODO: handle this
		}
		ctx.Respond(resourcePoolSummary)

	case apiv1.GetResourcePoolsRequest:
		summaries := make([]*resourcepoolv1.ResourcePool, len(a.poolsConfig.ResourcePools))
		for _, pool := range a.poolsConfig.ResourcePools {
			summary, err := a.createResourcePoolSummary(ctx, pool.PoolName)
			if err != nil {
				// TODO: handle error
			}
			summaries = append(summaries, summary)
		}
		ctx.Respond(summaries)


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

func (a *agentResourceManager) getDefaultResourcePool(msg AllocateRequest) string {
	if msg.SlotsNeeded == 0 {
		return a.config.DefaultCPUResourcePool
	}
	return a.config.DefaultGPUResourcePool
}

func (a *agentResourceManager) forwardToPool(
	ctx *actor.Context, resourcePool string, msg actor.Message,
) {
	if a.pools[resourcePool] == nil {
		err := errors.Errorf("cannot find resource pool %s for message %T from actor %s",
			resourcePool, ctx.Message(), ctx.Sender().Address().String())
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
) map[TaskID]TaskSummary {
	summaries := make(map[TaskID]TaskSummary)
	for _, resp := range resps {
		if resp != nil {
			typed := resp.(map[TaskID]TaskSummary)
			for id, summary := range typed {
				summaries[id] = summary
			}
		}
	}
	return summaries
}




func (a *agentResourceManager) getResourcePoolConfig(poolName string) (ResourcePoolConfig, error) {
	for i := range a.poolsConfig.ResourcePools {
		if a.poolsConfig.ResourcePools[i].PoolName == poolName {
			return a.poolsConfig.ResourcePools[i], nil
		}
	}
	return ResourcePoolConfig{},  errors.Errorf("cannot find resource pool %s", poolName)
}

func (a *agentResourceManager) createResourcePoolSummary(ctx *actor.Context, poolName string) (*resourcepoolv1.ResourcePool, error) {
	pool, err := a.getResourcePoolConfig(poolName)
	if err != nil {
		return &resourcepoolv1.ResourcePool{}, err
	}

	// TODO: Group the coallescing
	poolType := "static"
	if pool.Provider.AWS != nil {
		poolType = "aws"
	}
	if pool.Provider.GCP != nil {
		poolType = "gcp"
	}

	preemptible := false
	if poolType == "aws" {
		preemptible = pool.Provider.AWS.SpotEnabled
	}
	if poolType == "gcp" {
		preemptible = pool.Provider.GCP.InstanceType.Preemptible
	}

	var schedulerType string
	if pool.Scheduler.FairShare != nil {
		schedulerType = "fair share"
	}
	if pool.Scheduler.Priority != nil {
		schedulerType = "priority"
	}
	if pool.Scheduler.RoundRobin != nil {
		schedulerType = "round robin"
	}

	location := "on-prem"
	// TODO: Add GCP and AWS to location info?
	if poolType == "aws" {
		location = pool.Provider.AWS.Region
		// TODO: Would be nice to automatically detect the AZ that the subnet is in as well
	}
	if poolType == "gcp" {
		location = pool.Provider.GCP.Zone
	}

	imageId := "N/A"
	if poolType == "aws" {
		imageId = pool.Provider.AWS.ImageID
		// TODO: Would be nice to also have the description/name instead of just the ID
	}
	if poolType == "gcp" {
		imageId = pool.Provider.GCP.BootDiskSourceImage
	}

	instanceType := "N/A"
	if poolType == "aws" {
		instanceType = string(pool.Provider.AWS.InstanceType)
	}
	if poolType == "gcp" {
		instanceTypeStringBuilder := strings.Builder{}
		instanceTypeStringBuilder.WriteString(pool.Provider.GCP.InstanceType.MachineType)
		instanceTypeStringBuilder.WriteString(", ")
		instanceTypeStringBuilder.WriteString(string(pool.Provider.GCP.InstanceType.GPUNum))
		instanceTypeStringBuilder.WriteString("x")
		instanceTypeStringBuilder.WriteString(pool.Provider.GCP.InstanceType.GPUType)
		instanceType = instanceTypeStringBuilder.String()
		// TODO: Confirm that this looks good as output for
	}

	resp := &resourcepoolv1.ResourcePool{
		Id:                           pool.PoolName,
		Description:                  pool.Description,
		Type:                         poolType,
		DefaultCpuPool:               a.config.DefaultCPUResourcePool == poolName,
		DefaultGpuPool:               a.config.DefaultGPUResourcePool == poolName,
		Preemptible:                  preemptible,
		MinAgents:                    int32(pool.Provider.MinInstances), // TODO: Handle static pool explicitly?
		MaxAgents:                    int32(pool.Provider.MaxInstances), // TODO: Handle static pool explicitly?
		CpuContainerCapacityPerAgent: int32(pool.MaxCPUContainersPerAgent),
		SchedulerType:                schedulerType,
		SchedulerFittingPolicy:       pool.Scheduler.FittingPolicy,
		Location:                     location,
		ImageId:                      imageId,
		InstanceType:                 instanceType,
		MasterUrl:                    pool.Provider.MasterURL,
		MasterCertName:               pool.Provider.MasterCertName,
		StartupScript:                pool.Provider.StartupScript,
		ContainerStartupScript:       pool.Provider.ContainerStartupScript,
		AgentDockerNetwork:           pool.Provider.AgentDockerNetwork,
		AgentDockerRuntime:           pool.Provider.AgentDockerRuntime,
		AgentDockerImage:             pool.Provider.AgentDockerImage,
		AgentFluentImage:             pool.Provider.AgentFluentImage,
		MaxIdleAgentPeriod:           time.Duration(pool.Provider.MaxIdleAgentPeriod).String(),
		MaxAgentStartingPeriod:       time.Duration(pool.Provider.MaxAgentStartingPeriod).String(),
		Details:                      &resourcepoolv1.ResourcePoolDetail{},
	}
	if poolType == "aws" {
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
	if poolType == "gcp" {
		gcp := pool.Provider.GCP
		// Note: We do not return base image config because of how complex the structure is and because
		// we are completely reliant on the schema being what GCP provides us.
		// TODO: We should probably come up with a solution to this eventually
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
			OperationTimeoutPeriod: time.Duration(gcp.OperationTimeoutPeriod).String(),
		}
	}

	if schedulerType == "priority" {
		resp.Details.PriorityScheduler = &resourcepoolv1.ResourcePoolPrioritySchedulerDetail{
			Preemption:      pool.Scheduler.Priority.Preemption,
			DefaultPriority: int32(*pool.Scheduler.Priority.DefaultPriority),
		}
	}

	resourceSummary := ctx.Ask(a.pools[poolName], GetResourceSummary{}).Get().(ResourceSummary)
	resp.NumAgents = int32(resourceSummary.numAgents)
	resp.SlotsAvailable = int32(resourceSummary.numTotalSlots)
	resp.SlotsUsed = int32(resourceSummary.numActiveSlots)
	resp.CpuContainerCapacity = int32(resourceSummary.maxNumCpuContainers)
	resp.CpuContainersRunning = int32(resourceSummary.numActiveCpuContainers)

	return resp, nil
}
