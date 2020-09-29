package resourcemanagers

import (
	"crypto/tls"
	"fmt"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/provisioner"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
)

// determinedResourceManager manages resources using Determined.
type determinedResourceManager struct {
	config      *DeterminedResourceManagerConfig
	poolsConfig *ResourcePoolsConfig
	cert        *tls.Certificate

	onlyPool *actor.Ref
}

func newDeterminedResourceManager(
	config *DeterminedResourceManagerConfig, poolsConfig *ResourcePoolsConfig, cert *tls.Certificate,
) *determinedResourceManager {
	return &determinedResourceManager{
		config:      config,
		poolsConfig: poolsConfig,
		cert:        cert,
	}
}

func (d *determinedResourceManager) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart:
		for ix := range d.poolsConfig.ResourcePools {
			config := d.poolsConfig.ResourcePools[ix]
			if ref := ctx.Child(config.PoolName); ref != nil {
				panic("cannot have duplicate resource pool names")
			}
			if rpRef, _ := d.createResourcePool(ctx, &config, d.cert); rpRef != nil {
				d.onlyPool = rpRef
			}
		}

	case
		AllocateRequest, ResourcesReleased,
		sproto.SetGroupMaxSlots, sproto.SetGroupWeight,
		GetTaskSummary, GetTaskSummaries,
		sproto.ConfigureEndpoints, sproto.GetEndpointActorAddress:
		d.forward(ctx, msg)

	default:
		return actor.ErrUnexpectedMessage(ctx)
	}
	return nil
}

func (d *determinedResourceManager) forward(ctx *actor.Context, msg actor.Message) {
	if ctx.ExpectingResponse() {
		response := ctx.Ask(d.onlyPool, msg)
		ctx.Respond(response.Get())
	} else {
		ctx.Tell(d.onlyPool, msg)
	}
}

func (d *determinedResourceManager) createResourcePool(
	ctx *actor.Context, config *ResourcePoolConfig, cert *tls.Certificate,
) (*actor.Ref, error) {
	ctx.Log().Infof("creating resource pool: %s", config.PoolName)
	var rp *ResourcePool
	if config.Provider == nil {
		ctx.Log().Infof("disabling provisioner for resource pool: %s", config.PoolName)
		rp = NewResourcePool(
			MakeScheduler(d.config.SchedulingPolicy),
			MakeFitFunction(d.config.FittingPolicy),
			nil,
			0,
		)
	} else {
		p, pRef, err := provisioner.Setup(ctx, config.Provider, cert)
		if err != nil {
			return nil, errors.Wrap(err, "cannot create resource pool")
		}
		rp = NewResourcePool(
			MakeScheduler(d.config.SchedulingPolicy),
			MakeFitFunction(d.config.FittingPolicy),
			pRef,
			p.SlotsPerInstance(),
		)
	}
	ref, ok := ctx.ActorOf(config.PoolName, rp)
	if !ok {
		panic(fmt.Sprintf("cannot create resource pool actor: %s", config.PoolName))
	}
	return ref, nil
}
