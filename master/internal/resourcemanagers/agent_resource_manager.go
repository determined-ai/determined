package resourcemanagers

import (
	"crypto/tls"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
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
