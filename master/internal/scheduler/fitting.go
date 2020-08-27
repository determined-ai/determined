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
	if fit := findSharedAgentFit(task, agents, fittingMethod); fit != nil {
		return []*fittingState{fit}
	}
	if task.fittingRequirements.SingleAgent || task.slotsNeeded <= 1 {
		return nil
	}
	if fits := findDedicatedAgentFits(task, agents, fittingMethod); len(fits) != 0 {
		return fits
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

func findDedicatedAgentFits(
	task *Task, agentStates map[*actor.Ref]*agentState, fittingMethod SoftConstraint,
) []*fittingState {
	if len(agentStates) == 0 {
		return nil
	}

	// Scheduling assumption(s):
	// 1) Multi-agent tasks will receive the same number of slots on every agent. This is
	//    a valid assumption, because the only multi-agent tasks are distributed experiments
	//    which always want equal distribution across agents.
	// 2) Multi-agent tasks will receive all the slots on every agent they are scheduled on.
	agentsByNumSlots := make(map[int][]*agentState)
	for _, agent := range agentStates {
		if agent.numUsedSlots() != 0 {
			continue
		}

		constraints := []HardConstraint{labelSatisfied}
		if isViable(task, agent, constraints...) {
			agentsByNumSlots[agent.numEmptySlots()] = append(agentsByNumSlots[agent.numEmptySlots()], agent)
		}
	}

	var numSlots sort.IntSlice
	for n := range agentsByNumSlots {
		numSlots = append(numSlots, n)
	}

	// For the distributed case, we want to use as few
	// agents as possible, thus we prioritize the largest agents.
	sort.Sort(sort.Reverse(numSlots))

	var candidateNumSlots int
	for _, n := range numSlots {
		if n == 0 {
			continue
		}

		if s := task.SlotsNeeded(); s%n != 0 {
			continue
		}

		maxSlots := len(agentsByNumSlots[n]) * n
		if s := task.SlotsNeeded(); maxSlots >= s {
			candidateNumSlots = n
			break
		}
	}

	if candidateNumSlots == 0 {
		log.Infof("Task: %s which requires %d slots, can not be scheduled onto multiple agents "+
			"in the current cluster configuration. When running on multiple agents, number of "+
			"slots per trial must be either set to 1 or a multiple of the GPUs per agent.",
			task.ID, task.SlotsNeeded(),
		)
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

	sort.Sort(candidates)

	numContainers := task.SlotsNeeded() / candidateNumSlots
	slotsPerContainer := task.SlotsNeeded() / numContainers
	fits := candidates[:numContainers]
	for _, c := range fits {
		c.Slots = slotsPerContainer
	}

	return fits
}

func findSharedAgentFit(
	task *Task, agents map[*actor.Ref]*agentState, fittingMethod SoftConstraint,
) *fittingState {
	var candidates candidateList
	for _, agent := range agents {
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
