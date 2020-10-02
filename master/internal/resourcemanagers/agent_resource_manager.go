package resourcemanagers

import (
	"crypto/tls"

	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
)

type agentResourceManager struct {
	config      *AgentResourceManagerConfig
	poolsConfig *ResourcePoolsConfig
	cert        *tls.Certificate

	// onlyPool hosts the reference to the only resource pool
	// since we currently support only one resource pool.
	onlyPool *actor.Ref
}

func newAgentResourceManager(
	config *AgentResourceManagerConfig, poolsConfig *ResourcePoolsConfig, cert *tls.Certificate,
) *agentResourceManager {
	return &agentResourceManager{
		config:      config,
		poolsConfig: poolsConfig,
		cert:        cert,
	}
}

func (a *agentResourceManager) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart:
		for ix := range a.poolsConfig.ResourcePools {
			rpRef := a.createResourcePool(ctx, &a.poolsConfig.ResourcePools[ix], a.cert)
			if rpRef != nil {
				a.onlyPool = rpRef
				return nil
			}
		}

	case
		AllocateRequest, ResourcesReleased,
		sproto.SetGroupMaxSlots, sproto.SetGroupWeight,
		GetTaskSummary, GetTaskSummaries, SetTaskName,
		sproto.ConfigureEndpoints:
		a.forward(ctx, msg)

	default:
		return actor.ErrUnexpectedMessage(ctx)
	}
	return nil
}

func (a *agentResourceManager) forward(ctx *actor.Context, msg actor.Message) {
	if ctx.ExpectingResponse() {
		response := ctx.Ask(a.onlyPool, msg)
		ctx.Respond(response.Get())
	} else {
		ctx.Tell(a.onlyPool, msg)
	}
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
