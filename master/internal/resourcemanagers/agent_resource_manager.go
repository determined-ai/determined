package resourcemanagers

import (
	"crypto/tls"
	"strings"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"time"
)

type agentResourceManager struct {
	config      *AgentResourceManagerConfig
	poolsConfig *ResourcePoolsConfig
	cert        *tls.Certificate

	pools map[string]*actor.Ref
}

type GetResourcePoolSummary struct {}


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

	case GetResourcePoolSummary:
		// Send default information
		// Send ResourcePoolConfig



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

type ResourcePoolSummary struct {
	name string
	description string
	poolType string  // TODO: Maybe enum?
	//numAgents int
	//slotsAvailable
	//slotsUsed
	//cpuContainerCapacity
	//CpuContainersRunning
	defaultGpuPool bool
	defaultCpuPool bool
	preemptible bool
	minAgents int
	maxAgents int
	//AcceleratorsPerAgent  // TODO: Do we want to offer this?
	cpuContainerCapacityPerAgent int
	schedulerType string // TODO: Maybe enum?
	schedulerFittingPolicy string  // TODO: Maybe enum?
	location string
	imageId string
	instanceType string
	masterUrl string
	masterCertName string
	startupScript string
	containerStartupScript string
	agentDockerNetwork string
	agentDockerRuntime string
	agentDockerImage string
	agentFluentImage string
	maxIdleAgentPeriod string // TODO: Should this be a string?
	maxAgentStartingPeriod string // TODO: Should this be a string?
	//Details
}



func (a *agentResourceManager) getResourcePoolConfig(poolName string) (ResourcePoolConfig, error) {
	// TODO: Implement correctly
	return a.poolsConfig.ResourcePools[0], nil
}

func (a *agentResourceManager) createResourcePoolSummary(poolName string) (ResourcePoolSummary, error) {
	pool, err := a.getResourcePoolConfig(poolName)
	if err != nil {
		return ResourcePoolSummary{}, err
	}


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
		// TODO: Confirm that this looks good
	}




	resp := ResourcePoolSummary{
		name: pool.PoolName,
		description: pool.Description,
		poolType: poolType,
		defaultCpuPool: a.config.DefaultCPUResourcePool == poolName,
		defaultGpuPool: a.config.DefaultGPUResourcePool == poolName,
		preemptible:preemptible,
		minAgents: pool.Provider.MinInstances,  // TODO: Handle static pool explicitly?
		maxAgents: pool.Provider.MaxInstances,  // TODO: Handle static pool explicitly?
		cpuContainerCapacityPerAgent: pool.MaxCPUContainersPerAgent,
		schedulerType: schedulerType,
		schedulerFittingPolicy: pool.Scheduler.FittingPolicy,
		location:location,
		imageId:imageId,
		instanceType: instanceType,
		masterUrl: pool.Provider.MasterURL,
		masterCertName: pool.Provider.MasterCertName,
		startupScript: pool.Provider.StartupScript,
		containerStartupScript: pool.Provider.ContainerStartupScript,
		agentDockerNetwork: pool.Provider.AgentDockerNetwork,
		agentDockerRuntime: pool.Provider.AgentDockerRuntime,
		agentDockerImage: pool.Provider.AgentDockerImage,
		agentFluentImage: pool.Provider.AgentFluentImage,
		maxIdleAgentPeriod: time.Duration(pool.Provider.MaxIdleAgentPeriod).String(),
		maxAgentStartingPeriod: time.Duration(pool.Provider.MaxAgentStartingPeriod).String(),








	}
	return resp, nil
}
