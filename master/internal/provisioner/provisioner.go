package provisioner

import (
	"crypto/tls"
	"time"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/scheduler"
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
	actor.Actor

	provider     provider
	scaleDecider *scaleDecider
}

type provider interface {
	instanceType() instanceType
	maxInstanceNum() int
	list(ctx *actor.Context) ([]*Instance, error)
	launch(ctx *actor.Context, instanceType instanceType, instanceNum int)
	terminate(ctx *actor.Context, instanceIDs []string)
}

// New creates a new Provisioner.
func New(config *Config, cert *tls.Certificate) (*Provisioner, error) {
	if err := config.initMasterAddress(); err != nil {
		return nil, err
	}
	var cluster provider
	switch {
	case config.AWS != nil:
		var err error
		if cluster, err = newAWSCluster(config, cert); err != nil {
			return nil, errors.Wrap(err, "cannot create an EC2 cluster")
		}
	case config.GCP != nil:
		var err error
		if cluster, err = newGCPCluster(config, cert); err != nil {
			return nil, errors.Wrap(err, "cannot create a GCP cluster")
		}
	}

	return &Provisioner{
		provider: cluster,
		scaleDecider: newScaleDecider(
			time.Duration(config.MaxIdleAgentPeriod),
			time.Duration(config.MaxAgentStartingPeriod),
		),
	}, nil
}

// Receive implements the actor.Actor interface.
func (p *Provisioner) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart:
		actors.NotifyAfter(ctx, actionCooldown, provisionerTick{})

	case provisionerTick:
		p.provision(ctx)
		actors.NotifyAfter(ctx, actionCooldown, provisionerTick{})

	case scheduler.ViewSnapshot:
		p.scaleDecider.updateSchedulerSnapshot(&msg)

	default:
		return actor.ErrUnexpectedMessage(ctx)
	}
	return nil
}

// SlotsPerInstance returns the number of slots per instance the provisioner launches.
func (p *Provisioner) SlotsPerInstance() int {
	return p.provider.instanceType().slots()
}

func (p *Provisioner) provision(ctx *actor.Context) {
	instances, err := p.provider.list(ctx)
	if err != nil {
		ctx.Log().WithError(err).Error("cannot list instances")
		return
	}
	if p.scaleDecider.updateInstanceSnapshot(instances) {
		ctx.Log().Infof("found %d instances: %s", len(instances), fmtInstances(instances))
	}
	if !p.scaleDecider.needScale() {
		return
	}

	ctx.Log().Debug("scale happening")
	toTerminate := p.scaleDecider.findInstancesToTerminate(
		ctx, p.provider.maxInstanceNum(),
	)
	if len(toTerminate) > 0 {
		p.provider.terminate(ctx, toTerminate)
	}

	numToLaunch := p.scaleDecider.calculateNumInstancesToLaunch(
		p.provider.instanceType(),
		p.provider.maxInstanceNum(),
	)
	if numToLaunch > 0 {
		p.provider.launch(ctx, p.provider.instanceType(), numToLaunch)
	}
}
