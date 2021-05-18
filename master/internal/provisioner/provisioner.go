package provisioner

import (
	"crypto/tls"
	"time"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/actor/actors"
)

const (
	actionCooldown = 5 * time.Second
	secureScheme   = "https"
)

// provisionerTick periodically triggers the provisioner to act.
type provisionerTick struct{}

// Provisioner implements an actor to provision and terminate agent instances.
// It is composed of three parts: a provisioner actor, a scaling decision maker, and a provider.
// 1. The provisioner actor accepts actor messages with pending tasks and idle agents.
//    1.1. `Scheduler` pushes an immutable view of agents and tasks to `Provisioner`. `Provisioner`
//         pulls instance data from instance providers.
// 2. Based on the pending tasks, the scaleDecider chooses how many new instances to launch and
//    which instances to terminate.
//    2.1 It terminates instances if they stay idle for more than `maxIdleAgentPeriod` time.
//    2.2 It checks recently launched instances and avoids provisioning more than needed.
// 3. The instance providers take actions to launch/terminate instances.
type Provisioner struct {
	provider     provider
	scaleDecider *scaleDecider
}

type provider interface {
	instanceType() instanceType
	prestart(ctx *actor.Context)
	list(ctx *actor.Context) ([]*Instance, error)
	launch(ctx *actor.Context, instanceNum int)
	terminate(ctx *actor.Context, instanceIDs []string)
}

// New creates a new Provisioner.
func New(resourcePool string, config *Config, cert *tls.Certificate) (*Provisioner, error) {
	if err := config.initMasterAddress(); err != nil {
		return nil, err
	}
	var cluster provider
	switch {
	case config.AWS != nil:
		var err error
		if cluster, err = newAWSCluster(resourcePool, config, cert); err != nil {
			return nil, errors.Wrap(err, "cannot create an EC2 cluster")
		}
	case config.GCP != nil:
		var err error
		if cluster, err = newGCPCluster(resourcePool, config, cert); err != nil {
			return nil, errors.Wrap(err, "cannot create a GCP cluster")
		}
	}

	return &Provisioner{
		provider: cluster,
		scaleDecider: newScaleDecider(
			time.Duration(config.MaxIdleAgentPeriod),
			time.Duration(config.MaxAgentStartingPeriod),
			maxDisconnectPeriod,
			config.MinInstances,
			config.MaxInstances,
		),
	}, nil
}

// Receive implements the actor.Actor interface.
func (p *Provisioner) Receive(ctx *actor.Context) error {
	ctx.AddLabel("resource-pool", ctx.Self().Parent().Address().Local())

	switch msg := ctx.Message().(type) {
	case actor.PreStart:
		p.provider.prestart(ctx)
		actors.NotifyAfter(ctx, actionCooldown, provisionerTick{})

	case provisionerTick:
		p.provision(ctx)
		actors.NotifyAfter(ctx, actionCooldown, provisionerTick{})

	case sproto.ScalingInfo:
		p.scaleDecider.updateScalingInfo(&msg)

	default:
		return actor.ErrUnexpectedMessage(ctx)
	}
	return nil
}

// SlotsPerInstance returns the number of Slots per instance the provisioner launches.
func (p *Provisioner) SlotsPerInstance() int {
	return p.provider.instanceType().Slots()
}

func (p *Provisioner) provision(ctx *actor.Context) {
	instances, err := p.provider.list(ctx)
	if err != nil {
		ctx.Log().WithError(err).Error("cannot list instances")
		return
	}
	if p.scaleDecider.updateInstanceSnapshot(instances) {
		ctx.Log().Infof("found state changes in %d instances: %s",
			len(instances), fmtInstances(instances))
	}

	p.scaleDecider.calculateInstanceStates()

	if toTerminate := p.scaleDecider.findInstancesToTerminate(); len(toTerminate.InstanceIDs) > 0 {
		ctx.Log().Infof("decided to terminate %d instances: %s",
			len(toTerminate.InstanceIDs), toTerminate.String())
		p.provider.terminate(ctx, toTerminate.InstanceIDs)
	}

	if numToLaunch := p.scaleDecider.calculateNumInstancesToLaunch(); numToLaunch > 0 {
		ctx.Log().Infof("decided to launch %d instances (type %s)",
			numToLaunch, p.provider.instanceType().name())
		p.provider.launch(ctx, numToLaunch)
	}
}
