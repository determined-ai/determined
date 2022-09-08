package provisioner

import (
	"sort"
	"time"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/mathx"
	"github.com/determined-ai/determined/master/pkg/model"
)

const (
	maxDisconnectPeriod = 10 * time.Minute
)

// scaleDecider makes decisions based on the following assumptions:
// 1. All pending tasks cannot fit into all agents when receiving the snapshots from
//    the scheduler, i.e. we need to launch new agents to fit the pending tasks.
// 2. All tasks, agents, and instances don't have empty identifiers.
// 3. All tasks, agents, and instances are not duplicated.
//
// scaleDecider ignores the agents that cannot be associated with any instances.
// scaleDecider considers the following two cases:
// 1. Instances that can be associated with agents.
// 2. Instances that cannot be associated with agents. There are several possible causes:
//    a. The provider is starting up the instances.
//    b. The instances are already running but agents on them are starting up.
//    c. The agents are disconnected to the master due to misconfiguration or some unknown reason.
type scaleDecider struct {
	maxIdlePeriod       time.Duration
	maxStartingPeriod   time.Duration
	maxDisconnectPeriod time.Duration
	minInstanceNum      int
	maxInstanceNum      int

	instanceSnapshot       map[string]*Instance
	connectedAgentSnapshot map[string]sproto.AgentSummary
	idleAgentSnapshot      map[string]sproto.AgentSummary
	desiredNewInstances    int

	instances        map[string]*Instance
	pending          map[string]bool
	recentlyLaunched map[string]bool
	stopped          map[string]bool
	disconnected     map[string]time.Time
	idle             map[string]time.Time
	longDisconnected map[string]bool
	longIdle         map[string]bool

	db           db.DB
	resourcePool string
}

func newScaleDecider(
	resourcePool string,
	maxIdlePeriod, maxStartingPeriod,
	maxDisconnectPeriod time.Duration,
	minInstanceNum int,
	maxInstanceNum int,
	db db.DB,
) *scaleDecider {
	return &scaleDecider{
		maxStartingPeriod:      maxStartingPeriod,
		maxIdlePeriod:          maxIdlePeriod,
		maxDisconnectPeriod:    maxDisconnectPeriod,
		minInstanceNum:         minInstanceNum,
		maxInstanceNum:         maxInstanceNum,
		instanceSnapshot:       make(map[string]*Instance),
		connectedAgentSnapshot: make(map[string]sproto.AgentSummary),
		idleAgentSnapshot:      make(map[string]sproto.AgentSummary),
		instances:              make(map[string]*Instance),
		pending:                make(map[string]bool),
		recentlyLaunched:       make(map[string]bool),
		stopped:                make(map[string]bool),
		disconnected:           make(map[string]time.Time),
		idle:                   make(map[string]time.Time),
		longDisconnected:       make(map[string]bool),
		longIdle:               make(map[string]bool),
		db:                     db,
		resourcePool:           resourcePool,
	}
}

func (s *scaleDecider) updateScalingInfo(info *sproto.ScalingInfo) {
	s.desiredNewInstances = info.DesiredNewInstances
	s.idleAgentSnapshot = make(map[string]sproto.AgentSummary)
	s.connectedAgentSnapshot = make(map[string]sproto.AgentSummary, len(info.Agents))
	for _, agent := range info.Agents {
		if agent.IsIdle {
			s.idleAgentSnapshot[agent.Name] = agent
		}
		s.connectedAgentSnapshot[agent.Name] = agent
	}
}

func (s *scaleDecider) updateInstanceSnapshot(instances []*Instance) bool {
	updateSnapshot := func() {
		s.instanceSnapshot = make(map[string]*Instance, len(instances))
		for _, inst := range instances {
			s.instanceSnapshot[inst.ID] = inst
		}
	}

	// Find if the instance snapshot has been changed.
	if s.instanceSnapshot == nil || len(s.instanceSnapshot) != len(instances) {
		updateSnapshot()
		return true
	}
	for _, inst := range instances {
		if other, ok := s.instanceSnapshot[inst.ID]; !ok || !inst.equals(*other) {
			updateSnapshot()
			return true
		}
	}
	return false
}

func (s *scaleDecider) recordInstanceStats(slots int) error {
	if s.db == nil {
		return nil
	}
	for _, inst := range s.instances {
		instID := inst.ID
		err := s.updateInstanceStartStats(s.resourcePool, instID, slots)
		if err != nil {
			continue
		}
	}
	for instID := range s.disconnected {
		err := s.updateInstanceEndStats(instID)
		if err != nil {
			return err
		}
	}
	for instID := range s.stopped {
		err := s.updateInstanceEndStats(instID)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *scaleDecider) updateInstanceStartStats(poolName string, instID string, slots int) error {
	return s.db.RecordInstanceStats(&model.InstanceStats{
		ResourcePool: poolName,
		InstanceID:   instID,
		Slots:        slots,
	})
}

func (s *scaleDecider) updateInstanceEndStats(instID string) error {
	return s.db.EndInstanceStats(&model.InstanceStats{
		InstanceID: instID,
	})
}

func (s *scaleDecider) updateInstancesEndStats(instIDs []string) error {
	if s.db == nil {
		return nil
	}
	var err error
	for _, instID := range instIDs {
		err = s.db.EndInstanceStats(&model.InstanceStats{
			InstanceID: instID,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *scaleDecider) calculateInstanceStates() {
	now := time.Now()
	pastDisconnected := s.disconnected
	pastIdle := s.idle
	s.instances = make(map[string]*Instance)
	s.pending = make(map[string]bool)
	s.recentlyLaunched = make(map[string]bool)
	s.stopped = make(map[string]bool)
	s.disconnected = make(map[string]time.Time)
	s.idle = make(map[string]time.Time)
	s.longDisconnected = make(map[string]bool)
	s.longIdle = make(map[string]bool)
	for _, inst := range s.instanceSnapshot {
		switch inst.State {
		case SpotRequestPendingAWS:
			s.instances[inst.ID] = inst
			s.pending[inst.ID] = true
			s.recentlyLaunched[inst.ID] = true

		case Starting, Running:
			s.instances[inst.ID] = inst

			// Connected agent instances.
			if _, connected := s.connectedAgentSnapshot[inst.AgentName]; connected {
				if _, ok := s.idleAgentSnapshot[inst.AgentName]; ok {
					// Connected idle agent instances.
					if t, ok := pastIdle[inst.ID]; ok {
						if now.After(t.Add(s.maxIdlePeriod)) {
							s.longIdle[inst.ID] = true
						}
						s.idle[inst.ID] = t
					} else {
						s.idle[inst.ID] = now
					}
				}
				continue
			}

			// Not connected and recently launched agent instances.
			if inst.LaunchTime.Add(s.maxStartingPeriod).After(now) {
				s.recentlyLaunched[inst.ID] = true
				continue
			}

			// Disconnected agent instances.
			if t, ok := pastDisconnected[inst.ID]; ok {
				if now.After(t.Add(s.maxDisconnectPeriod)) {
					s.longDisconnected[inst.ID] = true
				}
				s.disconnected[inst.ID] = t
			} else {
				s.disconnected[inst.ID] = now
			}
		case Stopped:
			s.stopped[inst.ID] = true
		}
	}
}

func (s *scaleDecider) findInstancesToTerminate() sproto.TerminateDecision {
	toTerminate := make(map[string]string)

	// Terminate stopped instances and find idle and disconnected instances.
	for id := range s.stopped {
		toTerminate[id] = sproto.TerminateStoppedInstances
		delete(s.stopped, id)
	}

	// Terminate instances that have not connected to the master for a long time.
	for id := range s.longDisconnected {
		toTerminate[id] = sproto.TerminateLongDisconnectedInstances
		delete(s.disconnected, id)
	}

	// Terminate instances that are idle for a long time.
	for id := range s.longIdle {
		if len(s.instances)-len(toTerminate) > s.minInstanceNum {
			toTerminate[id] = sproto.TerminateLongIdleInstances
			delete(s.idle, id)
		} else {
			break
		}
	}

	// Terminate instances to keep the number of instances less than than the desired size.
	// We start by terminating unfulfilled spot requests, then idle instances, then
	// disconnected instances, then the most recently provisioned instances
	for id := range s.pending {
		if len(s.instances)-len(toTerminate) > s.maxInstanceNum {
			toTerminate[id] = sproto.InstanceNumberExceedsMaximum
			delete(s.pending, id)
		} else {
			break
		}
	}
	for id := range s.idle {
		if len(s.instances)-len(toTerminate) > s.maxInstanceNum {
			toTerminate[id] = sproto.InstanceNumberExceedsMaximum
			delete(s.idle, id)
		} else {
			break
		}
	}
	for id := range s.disconnected {
		if len(s.instances)-len(toTerminate) > s.maxInstanceNum {
			toTerminate[id] = sproto.InstanceNumberExceedsMaximum
			delete(s.disconnected, id)
		} else {
			break
		}
	}
	instances := make([]*Instance, 0, len(s.instances))
	for _, inst := range s.instances {
		instances = append(instances, inst)
	}
	sort.Slice(instances, func(i, j int) bool {
		return instances[i].LaunchTime.After(instances[j].LaunchTime)
	})
	for i := 0; i < len(instances) && len(instances)-len(toTerminate) > s.maxInstanceNum; i++ {
		toTerminate[instances[i].ID] = sproto.InstanceNumberExceedsMaximum
	}

	res := sproto.TerminateDecision{}
	res.Reasons = toTerminate
	res.InstanceIDs = make([]string, 0, len(toTerminate))
	for inst := range toTerminate {
		res.InstanceIDs = append(res.InstanceIDs, inst)
	}
	return res
}

func (s *scaleDecider) calculateNumInstancesToLaunch() int {
	return mathx.Max(0, mathx.Clamp(
		s.minInstanceNum-len(s.instances),
		s.desiredNewInstances-len(s.recentlyLaunched),
		s.maxInstanceNum-len(s.instances),
	))
}
