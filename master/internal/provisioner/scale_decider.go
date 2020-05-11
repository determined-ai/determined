package provisioner

import (
	"sort"
	"time"

	"github.com/determined-ai/determined/master/internal/scheduler"
)

const (
	maxAgentStartingPeriod = 300 * time.Second
	minRetryInterval       = 10 * time.Second
)

// scaleDecider assumes all pending tasks cannot fit into idle agents when receiving the
// `ViewSnapshot` actor message.
//
// There are several cases we must consider:
// 1. Agents that are not associated with instances.
//    1.a. Agents can be in static instances.
//    1.b. Instances can be terminated but the websocket of these instances may not be closed.
// 2. Agents that are associated with instances are complete agents.
//    --> Terminate instances that have idle agents inside.
// 3. Instances that are not associated to agents.
//    3.a. Instances are in starting state including Instance starting and Agent starting
//         --> For the MVP, we simply use a large ActionCoolDown time.
//    3.b. Unhealthy instances: agents cannot connect to the master for wrong configuration or
//         unknown reason.
//         --> For the MVP, we don't do health check.
// 4  Instances with empty or duplicate agent names are error instances.
type scaleDecider struct {
	maxAgentStartingPeriod time.Duration
	maxIdleAgentPeriod     time.Duration

	lastProvision        time.Time
	lastSchedulerUpdated time.Time
	lastProviderUpdated  time.Time
	instanceSnapshot     []*Instance
	schedulerSnapshot    *scheduler.ViewSnapshot

	instancesMarked map[string]time.Time
}

func newScaleDecider(maxIdleAgentPeriod time.Duration) *scaleDecider {
	return &scaleDecider{
		maxAgentStartingPeriod: maxAgentStartingPeriod,
		maxIdleAgentPeriod:     maxIdleAgentPeriod,
		schedulerSnapshot:      &scheduler.ViewSnapshot{},
	}
}

func (s *scaleDecider) needScale() bool {
	lastUpdated := s.lastProviderUpdated
	if lastUpdated.Before(s.lastSchedulerUpdated) {
		lastUpdated = s.lastSchedulerUpdated
	}
	safePeriod := s.maxAgentStartingPeriod
	if safePeriod < s.maxIdleAgentPeriod {
		safePeriod = s.maxIdleAgentPeriod
	}

	now := time.Now()
	if now.After(s.lastProvision.Add(minRetryInterval)) ||
		s.lastProvision.Before(lastUpdated.Add(safePeriod)) {
		s.lastProvision = now
		return true
	}
	return false
}

func (s *scaleDecider) updateSchedulerSnapshot(snapshot *scheduler.ViewSnapshot) bool {
	s.schedulerSnapshot = snapshot
	s.lastSchedulerUpdated = time.Now()
	return true
}

func (s *scaleDecider) updateInstanceSnapshot(instances []*Instance) bool {
	now := time.Now()
	updated := func() {
		s.instanceSnapshot = instances
		s.lastProviderUpdated = now
	}
	if s.instanceSnapshot == nil || len(s.instanceSnapshot) != len(instances) {
		updated()
		return true
	}
	instanceMap := make(map[string]*Instance)
	for _, inst := range s.instanceSnapshot {
		instanceMap[inst.ID] = inst
	}
	for _, inst := range instances {
		if other, ok := instanceMap[inst.ID]; !ok || !inst.equals(*other) {
			updated()
			return true
		}
	}
	return false
}

func (s *scaleDecider) findInstancesToTerminate(
	maxInstanceNum int,
) []string {
	idleAgents := s.schedulerSnapshot.Agents
	instances := s.instanceSnapshot
	now := time.Now()
	// Terminate stopped instances.
	stoppedInstanceIDs := make([]string, 0, len(instances))
	candidates := make([]*Instance, 0, len(instances))
	for _, inst := range instances {
		switch inst.State {
		case Stopped:
			stoppedInstanceIDs = append(stoppedInstanceIDs, inst.ID)
		case Starting, Running:
			candidates = append(candidates, inst)
		}
	}
	instances = candidates

	// We assume that there is no duplicate instance ID in the input.
	// Separate out unique agents. Duplicate agents are not handled.
	uniqueIdleAgents := make(map[string]*scheduler.AgentSummary)
	for _, agent := range idleAgents {
		if _, ok := uniqueIdleAgents[agent.Name]; !ok {
			uniqueIdleAgents[agent.Name] = agent
		}
	}

	// Find instances to terminate.
	toMark := make(map[string]bool)
	uniqueAgentNames := make(map[string]*Instance)
	for _, inst := range instances {
		switch first, ok := uniqueAgentNames[inst.AgentName]; {
		case ok:
			toMark[inst.ID] = true
			toMark[first.ID] = true
		case inst.AgentName == "":
			toMark[inst.ID] = true
		default:
			uniqueAgentNames[inst.AgentName] = inst
		}
	}
	for _, inst := range instances {
		if _, ok := uniqueIdleAgents[inst.AgentName]; ok {
			toMark[inst.ID] = true
		}
	}

	// Mark instances to terminate for some time before actually terminating them.
	toTerminate := make(map[string]bool)
	instancesMarked := make(map[string]time.Time)
	for instID := range toMark {
		switch t, ok := s.instancesMarked[instID]; {
		case ok && now.After(t.Add(s.maxIdleAgentPeriod)):
			toTerminate[instID] = true
		case ok:
			instancesMarked[instID] = t
		default:
			instancesMarked[instID] = now
		}
	}
	s.instancesMarked = instancesMarked

	// Terminate instances to keep the number of instances less than the limit. We start by
	// terminating instances we've already marked for termination. Then we delete the ones that were
	// most recently provisioned.
	numExceeds := len(instances) - maxInstanceNum
	for inst := range s.instancesMarked {
		if len(toTerminate) >= numExceeds {
			break
		}
		delete(s.instancesMarked, inst)
		toTerminate[inst] = true
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
	res = append(res, stoppedInstanceIDs...)
	return res
}

func (s *scaleDecider) calculateNumInstancesToLaunch(
	instanceType instanceType,
	maxInstanceNum int,
) int {
	pendingTasks := s.schedulerSnapshot.Tasks
	instances := s.instanceSnapshot
	now := time.Now()
	validInstances := make([]*Instance, 0, len(instances))
	for _, inst := range instances {
		switch inst.State {
		case Starting, Running:
			validInstances = append(validInstances, inst)
		}
	}
	instances = validInstances

	if instanceType.slots() == 0 {
		return 0
	}
	var sum int
	for _, t := range pendingTasks {
		sum += t.SlotsNeeded
	}
	numNeeded := (sum + instanceType.slots() - 1) / instanceType.slots()
	if len(instances) == 0 && len(pendingTasks) > 0 && numNeeded == 0 {
		numNeeded = 1
	}

	// Check recently launched instances and negate them from the total needed number.
	numRecentlyLaunched := 0
	for _, inst := range instances {
		if inst.LaunchTime.Add(s.maxAgentStartingPeriod).After(now) {
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
