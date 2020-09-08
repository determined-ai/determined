package provisioner

import (
	"sort"
	"time"

	"github.com/determined-ai/determined/master/pkg/actor"

	"github.com/determined-ai/determined/master/internal/scheduler"
)

const (
	minRetryInterval = 10 * time.Second
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
	maxIdlePeriod     time.Duration
	maxStartingPeriod time.Duration

	lastProvision          time.Time
	lastSchedulerUpdated   time.Time
	lastProviderUpdated    time.Time
	instanceSnapshot       map[string]*Instance
	idleAgentSnapshot      map[string]*scheduler.AgentSummary
	connectedAgentSnapshot map[string]*scheduler.AgentSummary
	taskSnapshot           []*scheduler.TaskSummary

	pastIdleInstances         map[string]time.Time
	pastDisconnectedInstances map[string]time.Time
}

func newScaleDecider(
	maxIdlePeriod time.Duration, maxStartingPeriod time.Duration,
) *scaleDecider {
	return &scaleDecider{
		maxStartingPeriod: maxStartingPeriod,
		maxIdlePeriod:     maxIdlePeriod,
	}
}

// needScale returns if a cluster is ready for rescaling.
// It returns true if one of the following situations is met:
// 1. The time has passed over the minimum retrying interval since last provision.
// 2. last provision < last update + the maximum agent starting period.
// 3. last provision < last update + the maximum agent idle period.
func (s *scaleDecider) needScale() bool {
	lastUpdated := s.lastProviderUpdated
	if lastUpdated.Before(s.lastSchedulerUpdated) {
		lastUpdated = s.lastSchedulerUpdated
	}

	now := time.Now()
	if now.After(s.lastProvision.Add(minRetryInterval)) ||
		s.lastProvision.Before(lastUpdated.Add(s.maxStartingPeriod)) ||
		s.lastProvision.Before(lastUpdated.Add(s.maxIdlePeriod)) {
		s.lastProvision = now
		return true
	}
	return false
}

func (s *scaleDecider) updateSchedulerSnapshot(snapshot *scheduler.ViewSnapshot) {
	s.idleAgentSnapshot = make(map[string]*scheduler.AgentSummary)
	for _, agent := range snapshot.IdleAgents {
		s.idleAgentSnapshot[agent.Name] = agent
	}

	s.connectedAgentSnapshot = make(map[string]*scheduler.AgentSummary)
	for _, agent := range snapshot.ConnectedAgents {
		s.connectedAgentSnapshot[agent.Name] = agent
	}

	s.taskSnapshot = snapshot.Tasks
	s.lastSchedulerUpdated = time.Now()
}

func (s *scaleDecider) updateInstanceSnapshot(instances []*Instance) bool {
	updated := func() {
		s.instanceSnapshot = make(map[string]*Instance)
		for _, inst := range instances {
			s.instanceSnapshot[inst.ID] = inst
		}
		s.lastProviderUpdated = time.Now()
	}
	if s.instanceSnapshot == nil || len(s.instanceSnapshot) != len(instances) {
		updated()
		return true
	}

	for _, inst := range instances {
		if other, ok := s.instanceSnapshot[inst.ID]; !ok || !inst.equals(*other) {
			updated()
			return true
		}
	}
	return false
}

func (s *scaleDecider) findInstancesToTerminate(
	ctx *actor.Context,
	maxInstanceNum int,
) []string {
	toTerminate := make(map[string]bool)
	idleInstances := make(map[string]bool)
	disconnectedInstances := make(map[string]bool)

	// Terminate stopped instances and find idle instances.
	for _, inst := range s.instanceSnapshot {
		switch inst.State {
		case Stopped:
			toTerminate[inst.ID] = true

		case Running:
			if _, ok := s.idleAgentSnapshot[inst.AgentName]; ok {
				idleInstances[inst.ID] = true
			}
		}
	}

	// Identify instances that are not currently connected to the master.
	now := time.Now()
	for _, inst := range s.instanceSnapshot {
		// If instance is connected no need to do anything here.
		if _, connected := s.connectedAgentSnapshot[inst.AgentName]; connected {
			continue
		}

		// If instance instance is still in the start-up period, do not terminate it for
		// being disconnected.
		if inst.LaunchTime.Add(s.maxStartingPeriod).After(now) {
			continue
		}

		// Don't terminate instances that are already stopped or are stopping.
		switch inst.State {
		case Stopping, Stopped:
			continue
		}

		disconnectedInstances[inst.ID] = true
	}

	// Terminate instances that have not connected to the master for a long time.
	var longUnconnected map[string]bool
	s.pastDisconnectedInstances, longUnconnected = findInstancesLongInSameState(
		s.pastDisconnectedInstances, disconnectedInstances, s.maxIdlePeriod)
	for id := range longUnconnected {
		toTerminate[id] = true
	}

	// Terminate instances that are idle for a long time.
	var longIdle map[string]bool
	s.pastIdleInstances, longIdle = findInstancesLongInSameState(
		s.pastIdleInstances, idleInstances, s.maxIdlePeriod)
	for id := range longIdle {
		toTerminate[id] = true
	}

	// Terminate instances to keep the number of instances less than the limit. We start by
	// terminating instances that are idle and haven't been terminated.
	// Then we terminate the ones that are most recently provisioned.
	numExceeds := len(s.instanceSnapshot) - maxInstanceNum
	for inst := range s.pastIdleInstances {
		if len(toTerminate) >= numExceeds {
			break
		}
		delete(s.pastIdleInstances, inst)
		toTerminate[inst] = true
	}
	instances := make([]*Instance, 0)
	for _, inst := range s.instanceSnapshot {
		instances = append(instances, inst)
	}
	sort.Slice(instances, func(i, j int) bool {
		return instances[i].LaunchTime.After(instances[j].LaunchTime)
	})
	for i := 0; i < len(instances) && len(toTerminate) < numExceeds; i++ {
		toTerminate[instances[i].ID] = true
	}

	res := make([]string, 0, len(toTerminate))
	for inst := range toTerminate {
		res = append(res, inst)
	}
	return res
}

func (s *scaleDecider) calculateNumInstancesToLaunch(
	instanceType instanceType,
	maxInstanceNum int,
) int {
	if instanceType.slots() == 0 {
		return 0
	}

	instances := make([]*Instance, 0, len(s.instanceSnapshot))
	for _, inst := range s.instanceSnapshot {
		switch inst.State {
		case Starting, Running:
			instances = append(instances, inst)
		}
	}

	slotSum := 0
	for _, t := range s.taskSnapshot {
		slotSum += t.SlotsNeeded
	}
	numNeeded := (slotSum + instanceType.slots() - 1) / instanceType.slots()
	if len(instances) == 0 && len(s.taskSnapshot) > 0 && numNeeded == 0 {
		numNeeded = 1
	}

	// Check recently launched instances and subtract them from the total needed number.
	now := time.Now()
	numRecentlyLaunched := 0
	for _, inst := range instances {
		if inst.LaunchTime.Add(s.maxStartingPeriod).After(now) {
			numRecentlyLaunched++
		}
	}

	numToLaunch := numNeeded - numRecentlyLaunched
	numToLaunch = min(numToLaunch, maxInstanceNum-len(instances))
	return max(0, numToLaunch)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func findInstancesLongInSameState(
	pastInstancesInState map[string]time.Time,
	presentInstanceInState map[string]bool,
	duration time.Duration,
) (map[string]time.Time, map[string]bool) {
	updatedInstancesInState := make(map[string]time.Time)
	durationExceededInState := make(map[string]bool)
	now := time.Now()
	for id := range presentInstanceInState {
		switch t, ok := pastInstancesInState[id]; {
		case ok && now.After(t.Add(duration)):
			durationExceededInState[id] = true
		case ok:
			updatedInstancesInState[id] = t
		default:
			updatedInstancesInState[id] = now
		}
	}
	return updatedInstancesInState, durationExceededInState
}
