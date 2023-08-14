package provisioner

import (
	"crypto/tls"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/internal/config/provconfig"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/rm/agentrm/provisioner/agentsetup"
	"github.com/determined-ai/determined/master/internal/rm/agentrm/provisioner/aws"
	"github.com/determined-ai/determined/master/internal/rm/agentrm/provisioner/gcp"
	"github.com/determined-ai/determined/master/internal/rm/agentrm/provisioner/scaledecider"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/internal/telemetry"
	"github.com/determined-ai/determined/master/pkg/actor"
	errInfo "github.com/determined-ai/determined/master/pkg/errors"
	"github.com/determined-ai/determined/master/pkg/model"
)

const (
	actionCooldown      = 5 * time.Second
	telemetryCooldown   = 90 * time.Second
	maxDisconnectPeriod = 10 * time.Minute
)

// Provisioner provisions and terminates agent instances.
// It is composed of four parts: a provisioner, a scaling decision maker, a provider,
// and a rate limiter.
//
//  1. The provisioner is capable of reporting provider information and updating scaling info.
//     1.1. `Scheduler` pushes an immutable view of agents and tasks to `Provisioner`. `Provisioner`
//     pulls instance data from instance providers.
//  2. Based on the pending tasks, the scaleDecider chooses how many new instances to launch and
//     which instances to terminate.
//     2.1 It terminates instances if they stay idle for more than `maxIdleAgentPeriod` time.
//     2.2 It checks recently launched instances and avoids provisioning more than needed.
//  3. The instance providers take actions to launch/terminate instances.
//  4. The rate limiter ensures telemetry does not get sent more frequently than every 90sec.
type Provisioner struct {
	mu sync.Mutex

	provider         agentsetup.Provider
	scaleDecider     *scaledecider.ScaleDecider
	telemetryLimiter *rate.Limiter
	launchErr        *errInfo.StickyError

	syslog *logrus.Entry

	system *actor.System
}

// New creates a new Provisioner.
func New(
	system *actor.System,
	resourcePool string, config *provconfig.Config, cert *tls.Certificate, db db.DB,
) (*Provisioner, error) {
	if err := config.InitMasterAddress(); err != nil {
		return nil, err
	}
	var cluster agentsetup.Provider
	switch {
	case config.AWS != nil:
		var err error
		if cluster, err = aws.New(resourcePool, config, cert); err != nil {
			return nil, errors.Wrap(err, "cannot create an EC2 cluster")
		}
	case config.GCP != nil:
		var err error
		if cluster, err = gcp.New(resourcePool, config, cert); err != nil {
			return nil, errors.Wrap(err, "cannot create a GCP cluster")
		}
	}

	var launchErrorTimeout time.Duration
	if config != nil && config.LaunchErrorTimeout != nil {
		launchErrorTimeout = time.Duration(*config.LaunchErrorTimeout)
	}

	return &Provisioner{
		provider: cluster,
		scaleDecider: scaledecider.New(
			resourcePool,
			time.Duration(config.MaxIdleAgentPeriod),
			time.Duration(config.MaxAgentStartingPeriod),
			maxDisconnectPeriod,
			config.MinInstances,
			config.MaxInstances,
			db,
		),
		telemetryLimiter: rate.NewLimiter(rate.Every(telemetryCooldown), 1),
		launchErr:        errInfo.NewStickyError(launchErrorTimeout, config.LaunchErrorRetries),

		syslog: logrus.WithField("component", "provisioner").
			WithField("resource-pool", resourcePool),
		system: system,
	}, nil
}

// Run starts the provisioner loop.
func (p *Provisioner) Run() {
	for {
		// Cooldown period before the provisioner starts.
		time.Sleep(actionCooldown)
		p.Provision()
	}
}

// UpdateScalingInfo updates the scaling info for the provisioner.
func (p *Provisioner) UpdateScalingInfo(info *sproto.ScalingInfo) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.scaleDecider.UpdateScalingInfo(info)
}

// SlotsPerInstance returns the number of Slots per instance the provisioner launches.
func (p *Provisioner) SlotsPerInstance() int {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.provider.SlotsPerInstance()
}

// CurrentSlotCount returns the number of Slots available in the cluster.
func (p *Provisioner) CurrentSlotCount() (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	nodes, err := p.provider.List()
	if err != nil {
		p.syslog.WithError(err).Error("cannot List instances for current slot count")
		return 0, err
	}
	return p.provider.SlotsPerInstance() * len(nodes), nil
}

// InstanceType returns the instance type of the provider for the provisioner.
func (p *Provisioner) InstanceType() string {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.provider.InstanceType().Name()
}

// Provision runs a single provisioning iteration.
func (p *Provisioner) Provision() {
	p.mu.Lock()
	defer p.mu.Unlock()

	instances, err := p.provider.List()
	if err != nil {
		p.syslog.WithError(err).Error("cannot List instances for provisioning")
		return
	}
	updated := p.scaleDecider.UpdateInstanceSnapshot(instances)
	if updated {
		p.syslog.Infof("found state changes in %d instances: %s",
			len(instances), model.FmtInstances(instances))
	}

	p.scaleDecider.CalculateInstanceStates()

	if updated {
		err = p.scaleDecider.RecordInstanceStats(p.provider.SlotsPerInstance())
		if err != nil {
			p.syslog.WithError(err).Error("cannot record instance stats")
		}
	}

	if toTerminate := p.scaleDecider.FindInstancesToTerminate(); len(toTerminate.InstanceIDs) > 0 {
		p.syslog.Infof("decided to terminate %d instances: %s",
			len(toTerminate.InstanceIDs), toTerminate.String())
		p.provider.Terminate(toTerminate.InstanceIDs)
		err = p.scaleDecider.UpdateInstancesEndStats(toTerminate.InstanceIDs)
		if err != nil {
			p.syslog.WithError(err).Error("cannot update end stats for terminated instance")
		}
	}

	if numToLaunch := p.scaleDecider.CalculateNumInstancesToLaunch(); numToLaunch > 0 {
		p.syslog.Infof("decided to launch %d instances (type %s)",
			numToLaunch, p.provider.InstanceType().Name())
		if err := p.launch(numToLaunch); err != nil {
			p.syslog.WithError(err).Error("failure launching instances")
		}
	}

	if p.telemetryLimiter.Allow() {
		telemetry.ReportProvisionerTick(instances, p.provider.InstanceType().Name())
	}
}

func (p *Provisioner) launch(numToLaunch int) error {
	if err := p.launchErr.Error(); err != nil {
		return err
	}
	return p.launchErr.SetError(p.provider.Launch(numToLaunch))
}

// LaunchError returns the current launch error sent from the provider.
func (p *Provisioner) LaunchError() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.launchErr.Error()
}
