package scheduler

import (
	"crypto/md5" // #nosec
	"encoding/binary"
	"sort"

	log "github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/pkg/actor"
)

// HardConstraint returns true if the task can be assigned to the agent and false otherwise.
type HardConstraint func(task *Task, agent *agentState) bool

// SoftConstraint returns a score from 0 (lowest) to 1 (highest) representing how optimal is the
// state of the cluster if the task were assigned to the agent.
type SoftConstraint func(task *Task, agent *agentState) float64

// fittingState is the basis for assigning a task to one or more agents for execution.
type fittingState struct {
	Agent        *agentState
	Score        float64
	HashDistance uint64
	Slots        int
}

// FittingRequirements allow tasks to specify requirements for their placement.
type FittingRequirements struct {
	// SingleAgent specifies that the task must be located within a single agent.
	SingleAgent bool
	// DedicatedAgent specified that there must be no other tasks running on the agent.
	DedicatedAgent bool
}

type candidateList []*fittingState

func (c candidateList) Len() int {
	return len(c)
}

func (c candidateList) Less(i, j int) bool {
	// Multiple zero-slot tasks will all end up on the same agent if we only consider the fitting
	// score. To combat this, break ties by selecting the agent whose hashed ID is closest to the
	// hashed ID of the task.

	a := c[i]
	b := c[j]
	switch {
	case a.Score > b.Score:
		return true
	case a.Score < b.Score:
		return false
	case a.HashDistance < b.HashDistance:
		return true
	case a.HashDistance > b.HashDistance:
		return false
	default:
		// As a final tiebreaker, compare the agent IDs in string order.
		return a.Agent.handler.Address().String() < b.Agent.handler.Address().String()
	}
}

func (c candidateList) Swap(i, j int) {
	c[j], c[i] = c[i], c[j]
}

func findFits(
	task *Task, agents map[*actor.Ref]*agentState, fittingMethod SoftConstraint,
) []*fittingState {
	if task.SlotsNeeded() <= 1 {
		if fit := findNotMultiSlotFit(task, agents, fittingMethod); fit != nil {
			return []*fittingState{
				fit,
			}
		}
	} else {
		if fits := findMultiSlotFits(task, agents, fittingMethod); len(fits) != 0 {
			return fits
		}
	}

	return nil
}

func isViable(task *Task, agent *agentState, constraints ...HardConstraint) bool {
	for _, constraint := range constraints {
		if !constraint(task, agent) {
			return false
		}
	}
	return true
}

func findMultiSlotFits(
	task *Task, agentStates map[*actor.Ref]*agentState, fittingMethod SoftConstraint,
) []*fittingState {
	if len(agentStates) == 0 {
		return nil
	}

	// Scheduling assumption(s):
	// 1) Multi-agent tasks will receive the same number of slots on every agent. This is
	//    a valid assumption, because the only multi-agent tasks are distributed experiments
	//    which always want equal distribution across agents.
	// 2) When a task enables `DedicatedAgent`, it will only schedule a task on a agent if it
	//    takes up all of the agent's slots.

	agentsByNumSlots := make(map[int][]*agentState)
	for _, agent := range agentStates {
		if task.fittingRequirements.DedicatedAgent && agent.numUsedSlots() != 0 {
			continue
		}

		constraints := []HardConstraint{labelSatisfied}
		if task.fittingRequirements.SingleAgent {
			constraints = append(constraints, slotsSatisfied)
		}

		if isViable(task, agent, constraints...) {
			agentsByNumSlots[agent.numEmptySlots()] = append(agentsByNumSlots[agent.numEmptySlots()], agent)
		}
	}

	var numSlots sort.IntSlice
	for n := range agentsByNumSlots {
		numSlots = append(numSlots, n)
	}

	if task.fittingRequirements.SingleAgent {
		// For the single agent case, we want prioritize using the smallest
		// available agents.
		sort.Sort(numSlots)
	} else {
		// For the (potentially) distributed case, we want to use as few
		// agents as possible, thus we prioritize the largest agents.
		sort.Sort(sort.Reverse(numSlots))
	}

	var candidateNumSlots int
	for _, n := range numSlots {
		if n == 0 {
			continue
		}

		var maxSlots int
		if task.fittingRequirements.SingleAgent {
			maxSlots = n
		} else {
			agents := agentsByNumSlots[n]
			maxSlots = len(agents) * n
		}

		if s := task.SlotsNeeded(); (task.fittingRequirements.DedicatedAgent || s > n) && s%n != 0 {
			continue
		}

		if s := task.SlotsNeeded(); maxSlots >= s {
			candidateNumSlots = n
			break
		}
	}

	if candidateNumSlots == 0 {
		if task.fittingRequirements.DedicatedAgent {
			log.Infof("Task: %s which requires %d slots, can not be scheduled in the current "+
				"cluster configuration. The number of slots per trial must be either set to 1 or "+
				"a multiple of the GPUs per agent.", task.ID, task.SlotsNeeded(),
			)
		}
		return nil
	}

	var candidates candidateList
	for _, agent := range agentsByNumSlots[candidateNumSlots] {
		candidates = append(candidates, &fittingState{
			Agent:        agent,
			Score:        fittingMethod(task, agent),
			HashDistance: hashDistance(task, agent),
		})
	}

	numContainers := task.SlotsNeeded() / candidateNumSlots
	if task.SlotsNeeded()%candidateNumSlots != 0 {
		// For tasks that do not take up a whole agent.
		numContainers++
	}
	slotsPerContainer := task.SlotsNeeded() / numContainers

	if len(candidates) < numContainers {
		log.Infof("Task: %s requires %d machines with %d slots to be scheduled. There are "+
			"currently %d machines fully available.", task.ID, numContainers,
			candidateNumSlots, len(candidates),
		)
		return nil
	}

	sort.Sort(candidates)

	fits := candidates[:numContainers]
	for _, c := range fits {
		c.Slots = slotsPerContainer
	}

	return fits
}

func findNotMultiSlotFit(
	task *Task, agents map[*actor.Ref]*agentState, fittingMethod SoftConstraint,
) *fittingState {
	var candidates candidateList
	for _, agent := range agents {
		if task.fittingRequirements.DedicatedAgent && agent.numUsedSlots() != 0 {
			continue
		}

		if !isViable(task, agent, slotsSatisfied, labelSatisfied) {
			continue
		}

		candidates = append(candidates, &fittingState{
			Agent:        agent,
			Score:        fittingMethod(task, agent),
			HashDistance: hashDistance(task, agent),
		})
	}

	if len(candidates) == 0 {
		return nil
	}

	sort.Sort(candidates)

	candidates[0].Slots = task.SlotsNeeded()
	return candidates[0]
}

func stringHashNumber(s string) uint64 {
	// An array must have an address (essentially, be assigned to a variable) to be sliced.
	hash := md5.Sum([]byte(s)) // #nosec
	return binary.LittleEndian.Uint64(hash[:])
}

func hashDistance(task *Task, agent *agentState) uint64 {
	return stringHashNumber(string(task.ID)) - stringHashNumber(agent.handler.Address().String())
}
