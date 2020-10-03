package resourcemanagers

import (
	"crypto/tls"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
)

type agentResourceManager struct {
	config      *AgentResourceManagerConfig
	poolsConfig *ResourcePoolsConfig
	cert        *tls.Certificate

	pools map[string]*actor.Ref
}

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
			rpRef := a.createResourcePool(ctx, &a.poolsConfig.ResourcePools[ix], a.cert)
			if rpRef != nil {
				a.pools[config.PoolName] = rpRef
			}
		}
		ctx.AskAll(actor.Ping{}, ctx.Children()...).GetAll()

	case AllocateRequest:
		a.setDefaultResourcePool(&msg)
		a.forward(ctx, msg.ResourcePool, msg)
	case ResourcesReleased:
		for name := range a.pools {
			a.forward(ctx, name, msg)
		}

	case sproto.SetGroupMaxSlots:
		a.forward(ctx, msg.ResourcePool, msg)
	case sproto.SetGroupWeight:
		a.forward(ctx, msg.ResourcePool, msg)
	case GetTaskSummary:
		if summary := a.aggregateTaskSummary(a.forwardAll(ctx, msg)); summary != nil {
			ctx.Respond(summary)
		}
	case GetTaskSummaries:
		ctx.Respond(a.aggregateTaskSummaries(a.forwardAll(ctx, msg)))
	case SetTaskName:
		a.forwardAll(ctx, msg)

	default:
		return actor.ErrUnexpectedMessage(ctx)
	}
	return nil
}

func (a *agentResourceManager) createResourcePool(
	ctx *actor.Context, config *ResourcePoolConfig, cert *tls.Certificate,
) *actor.Ref {
	ctx.Log().Infof("creating resource pool: %s", config.PoolName)
	rp := NewResourcePool(
		config,
		cert,
		MakeScheduler(a.config.SchedulingPolicy),
		MakeFitFunction(a.config.FittingPolicy),
	)
	ref, ok := ctx.ActorOf(config.PoolName, rp)
	if !ok {
		ctx.Log().Errorf("cannot create resource pool actor: %s", config.PoolName)
		return nil
	}
	return ref
}

func (a *agentResourceManager) setDefaultResourcePool(request *AllocateRequest) {
	if len(request.ResourcePool) == 0 {
		if request.SlotsNeeded == 0 {
			request.ResourcePool = a.config.DefaultCPUResourcePool
		} else {
			request.ResourcePool = a.config.DefaultGPUResourcePool
		}
	}
}

func (a *agentResourceManager) forward(
	ctx *actor.Context, resourcePool string, msg actor.Message,
) {
	if a.pools[resourcePool] == nil {
		err := errors.Errorf("cannot find resource pool: %s", resourcePool)
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

func (a *agentResourceManager) forwardAll(ctx *actor.Context, msg actor.Message) []actor.Message {
	respMap := ctx.AskAll(msg, ctx.Children()...).GetAll()
	resps := make([]actor.Message, 0)
	for _, resp := range respMap {
		resps = append(resps, resp)
	}
	return resps
}

func (a *agentResourceManager) aggregateTaskSummary(resps []actor.Message) *TaskSummary {
	for _, resp := range resps {
		if resp != nil {
			typed := resp.(TaskSummary)
			return &typed
		}
	}
	return nil
}

func (a *agentResourceManager) aggregateTaskSummaries(
	resps []actor.Message,
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
