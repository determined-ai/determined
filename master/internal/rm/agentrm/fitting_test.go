package agentrm

import (
	"fmt"
	"sort"
	"testing"

	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/internal/logpattern"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/set"
)

func TestIsViable(t *testing.T) {
	system := actor.NewSystem(t.Name())
	req := &sproto.AllocateRequest{SlotsNeeded: 2}

	assert.Assert(t, isViable(req,
		newFakeAgentState(t, system, "agent1", 4, 0, 100, 0), slotsSatisfied))
	assert.Assert(t, !isViable(req,
		newFakeAgentState(t, system, "agent2", 1, 0, 100, 0), slotsSatisfied))
	assert.Assert(t, !isViable(req,
		newFakeAgentState(t, system, "agent4", 1, 0, 100, 0), slotsSatisfied))
}

func TestFindFits(t *testing.T) {
	type testCase struct {
		Name          string
		Task          sproto.AllocateRequest
		Agents        []*MockAgent
		FittingMethod SoftConstraint

		ExpectedAgentFit int
	}

	testCases := []testCase{
		{
			Name: "0-slot multiple fits, idle agents",
			Task: sproto.AllocateRequest{AllocationID: "task1", SlotsNeeded: 0},
			Agents: []*MockAgent{
				NewMockAgent("agent1", 4, 0, 100, 0),
				NewMockAgent("agent2", 4, 0, 100, 0),
			},
			FittingMethod:    BestFit,
			ExpectedAgentFit: 0,
		},
		{
			Name: "0-slot multiple fits, idle agents, out-of-order",
			Task: sproto.AllocateRequest{AllocationID: "task1", SlotsNeeded: 0},
			Agents: []*MockAgent{
				NewMockAgent("agent2", 4, 0, 100, 0),
				NewMockAgent("agent1", 4, 0, 100, 0),
			},
			FittingMethod:    BestFit,
			ExpectedAgentFit: 1,
		},
		{
			Name: "0-slot multiple fits, in-use agents",
			Task: sproto.AllocateRequest{AllocationID: "task1", SlotsNeeded: 0},
			Agents: []*MockAgent{
				NewMockAgent("agent1", 4, 0, 100, 2),
				NewMockAgent("agent2", 4, 0, 100, 2),
			},
			FittingMethod:    BestFit,
			ExpectedAgentFit: 0,
		},
		{
			Name: "0-slot multiple fits, in-use agents, out-of-order",
			Task: sproto.AllocateRequest{AllocationID: "task1", SlotsNeeded: 0},
			Agents: []*MockAgent{
				NewMockAgent("agent2", 4, 0, 100, 2),
				NewMockAgent("agent1", 4, 0, 100, 2),
			},
			FittingMethod:    BestFit,
			ExpectedAgentFit: 1,
		},
		{
			Name: "0-slot multiple fits, max zero slot containers",
			Task: sproto.AllocateRequest{AllocationID: "task1", SlotsNeeded: 0},
			Agents: []*MockAgent{
				NewMockAgent("agent1", 4, 0, 2, 0),
				NewMockAgent("agent2", 4, 0, 4, 2),
			},
			FittingMethod:    BestFit,
			ExpectedAgentFit: 0,
		},
		{
			Name: "0-slot multiple fits, max zero slot containers, out-of-order",
			Task: sproto.AllocateRequest{AllocationID: "task1", SlotsNeeded: 0},
			Agents: []*MockAgent{
				NewMockAgent("agent2", 4, 0, 4, 2),
				NewMockAgent("agent1", 4, 0, 2, 0),
			},
			FittingMethod:    BestFit,
			ExpectedAgentFit: 1,
		},
		{
			Name: "0-slot single fit",
			Task: sproto.AllocateRequest{AllocationID: "task1", SlotsNeeded: 0},
			Agents: []*MockAgent{
				NewMockAgent("agent1", 4, 0, 100, 1),
				NewMockAgent("agent2", 4, 0, 100, 0),
			},
			FittingMethod:    BestFit,
			ExpectedAgentFit: 0,
		},
		{
			Name: "0-slot single fit, max zero slot containers",
			Task: sproto.AllocateRequest{AllocationID: "task1", SlotsNeeded: 0},
			Agents: []*MockAgent{
				NewMockAgent("agent1", 4, 0, 2, 1),
				NewMockAgent("agent2", 4, 0, 2, 0),
			},
			FittingMethod:    BestFit,
			ExpectedAgentFit: 0,
		},
		{
			Name: "2-slot multiple fits",
			Task: sproto.AllocateRequest{AllocationID: "task1", SlotsNeeded: 2},
			Agents: []*MockAgent{
				NewMockAgent("agent1", 2, 0, 100, 0),
				NewMockAgent("agent2", 4, 0, 100, 0),
			},
			FittingMethod:    BestFit,
			ExpectedAgentFit: 0,
		},
		{
			Name: "1-slot multiple fits",
			Task: sproto.AllocateRequest{AllocationID: "task1", SlotsNeeded: 1},
			Agents: []*MockAgent{
				NewMockAgent("agent1", 1, 0, 100, 0),
				NewMockAgent("agent2", 4, 0, 100, 0),
			},
			FittingMethod:    BestFit,
			ExpectedAgentFit: 0,
		},
		{
			Name: "1-slot out-of-order multiple fits",
			Task: sproto.AllocateRequest{AllocationID: "task1", SlotsNeeded: 1},
			Agents: []*MockAgent{
				NewMockAgent("agent1", 4, 0, 100, 0),
				NewMockAgent("agent2", 1, 0, 100, 0),
			},
			FittingMethod:    BestFit,
			ExpectedAgentFit: 1,
		},
		{
			Name: "4-slot single fit",
			Task: sproto.AllocateRequest{AllocationID: "task1", SlotsNeeded: 4},
			Agents: []*MockAgent{
				NewMockAgent("agent1", 4, 0, 100, 0),
				NewMockAgent("agent2", 1, 0, 100, 0),
			},
			FittingMethod:    BestFit,
			ExpectedAgentFit: 0,
		},
		{
			Name: "4-slot multiple fits",
			Task: sproto.AllocateRequest{AllocationID: "task1", SlotsNeeded: 1},
			Agents: []*MockAgent{
				NewMockAgent("agent1", 4, 0, 100, 0),
				NewMockAgent("agent2", 4, 0, 100, 0),
			},
			FittingMethod:    BestFit,
			ExpectedAgentFit: 0,
		},
		{
			Name: "2-slot multiple fits, in-use agents",
			Task: sproto.AllocateRequest{AllocationID: "task1", SlotsNeeded: 2},
			Agents: []*MockAgent{
				NewMockAgent("agent1", 2, 0, 100, 0),
				NewMockAgent("agent2", 4, 2, 100, 0),
			},
			FittingMethod:    BestFit,
			ExpectedAgentFit: 0,
		},
		{
			Name: "1-slot multiple fits, in-use agents",
			Task: sproto.AllocateRequest{AllocationID: "task1", SlotsNeeded: 1},
			Agents: []*MockAgent{
				NewMockAgent("agent1", 2, 1, 100, 0),
				NewMockAgent("agent2", 4, 1, 100, 0),
			},
			FittingMethod:    WorstFit,
			ExpectedAgentFit: 1,
		},
		{
			Name: "1-slot multiple fits, in-use agents, out of order",
			Task: sproto.AllocateRequest{AllocationID: "task1", SlotsNeeded: 1},
			Agents: []*MockAgent{
				NewMockAgent("agent1", 4, 1, 100, 0),
				NewMockAgent("agent2", 2, 1, 100, 0),
			},
			FittingMethod:    WorstFit,
			ExpectedAgentFit: 0,
		},
		{
			Name: "1-slot multiple fits, in-use-agents, odd numbers",
			Task: sproto.AllocateRequest{AllocationID: "task1", SlotsNeeded: 1},
			Agents: []*MockAgent{
				NewMockAgent("agent1", 2, 1, 100, 0),
				NewMockAgent("agent2", 5, 3, 100, 0),
			},
			FittingMethod:    WorstFit,
			ExpectedAgentFit: 0,
		},
		{
			Name: "2-slot multiple fits, in-use-agents, odd numbers",
			Task: sproto.AllocateRequest{AllocationID: "task1", SlotsNeeded: 2},
			Agents: []*MockAgent{
				NewMockAgent("agent1", 2, 0, 100, 0),
				NewMockAgent("agent2", 5, 3, 100, 0),
			},
			FittingMethod:    WorstFit,
			ExpectedAgentFit: 0,
		},
		{
			Name: "4-slot multiple fits, unoccupied",
			Task: sproto.AllocateRequest{AllocationID: "task1", SlotsNeeded: 4},
			Agents: []*MockAgent{
				NewMockAgent("agent1", 4, 0, 100, 0),
				NewMockAgent("agent2", 4, 0, 100, 0),
			},
			FittingMethod:    WorstFit,
			ExpectedAgentFit: 0,
		},
		{
			Name: "4-slot multiple fits, one exact",
			Task: sproto.AllocateRequest{AllocationID: "task1", SlotsNeeded: 4},
			Agents: []*MockAgent{
				NewMockAgent("agent1", 8, 0, 100, 0),
				NewMockAgent("agent2", 4, 0, 100, 0),
			},
			FittingMethod:    WorstFit,
			ExpectedAgentFit: 0,
		},
		{
			Name: "4-slot multiple fits, one exact, single agent",
			Task: sproto.AllocateRequest{
				AllocationID: "task1",
				SlotsNeeded:  4,
				FittingRequirements: sproto.FittingRequirements{
					SingleAgent: true,
				},
			},
			Agents: []*MockAgent{
				NewMockAgent("agent1", 8, 0, 100, 0),
				NewMockAgent("agent2", 4, 0, 100, 0),
			},
			FittingMethod:    WorstFit,
			ExpectedAgentFit: 0,
		},
		{
			Name: "2-slot multiple inexact fits",
			Task: sproto.AllocateRequest{AllocationID: "task1", SlotsNeeded: 2},
			Agents: []*MockAgent{
				NewMockAgent("agent1", 4, 0, 100, 0),
				NewMockAgent("agent2", 4, 0, 100, 0),
			},
			FittingMethod:    BestFit,
			ExpectedAgentFit: 0,
		},
		{
			Name: "2-slot multiple inexact fits, single agent",
			Task: sproto.AllocateRequest{
				AllocationID: "task1",
				SlotsNeeded:  2,
				FittingRequirements: sproto.FittingRequirements{
					SingleAgent: true,
				},
			},
			Agents: []*MockAgent{
				NewMockAgent("agent1", 4, 0, 100, 0),
				NewMockAgent("agent2", 4, 0, 100, 0),
			},
			FittingMethod:    BestFit,
			ExpectedAgentFit: 0,
		},
		{
			Name: "2-slot fit, single agent",
			Task: sproto.AllocateRequest{
				AllocationID: "task1",
				SlotsNeeded:  2,
				FittingRequirements: sproto.FittingRequirements{
					SingleAgent: true,
				},
			},
			Agents: []*MockAgent{
				NewMockAgent("agent1", 1, 0, 100, 0),
				NewMockAgent("agent2", 3, 0, 100, 0),
			},
			FittingMethod:    BestFit,
			ExpectedAgentFit: 1,
		},
		{
			Name: "2-slot fit, no single agent requirement",
			Task: sproto.AllocateRequest{AllocationID: "task1", SlotsNeeded: 2},
			Agents: []*MockAgent{
				NewMockAgent("agent1", 1, 0, 100, 0),
				NewMockAgent("agent2", 3, 0, 100, 0),
			},
			FittingMethod:    BestFit,
			ExpectedAgentFit: 1,
		},
	}

	for idx := range testCases {
		tc := testCases[idx]

		t.Run(tc.Name, func(t *testing.T) {
			system := actor.NewSystem(t.Name())
			agents := []*agentState{}
			for _, agent := range tc.Agents {
				agents = append(agents, newFakeAgentState(
					t,
					system,
					agent.ID,
					agent.Slots,
					agent.SlotsUsed,
					agent.MaxZeroSlotContainers,
					agent.ZeroSlotContainers,
				))
			}
			agentsByHandler, agentsByIndex := byHandler(agents...)
			fits := findFits(&tc.Task, agentsByHandler, tc.FittingMethod, false)
			assert.Assert(t, len(fits) > 0)
			assert.Equal(t, fits[0].Agent, agentsByIndex[tc.ExpectedAgentFit])
		})
	}
}

func TestFindDedicatedAgentFits(t *testing.T) {
	system := actor.NewSystem(t.Name())

	type testCase struct {
		Name                string
		SlotsNeeded         int
		AgentCapacities     []int
		ExpectedAgentFit    []int
		ExpectedLength      int
		FittingRequirements sproto.FittingRequirements
		HeterogenousFit     bool
	}

	testCases := []testCase{
		{
			Name:             "Simple satisfaction",
			SlotsNeeded:      10,
			AgentCapacities:  []int{1, 5, 10},
			ExpectedAgentFit: []int{2},
		},
		{
			Name:             "Compound satisfaction",
			SlotsNeeded:      16,
			AgentCapacities:  []int{8, 7, 7, 4, 4, 4, 4},
			ExpectedAgentFit: []int{3, 4, 5, 6},
			FittingRequirements: sproto.FittingRequirements{
				SingleAgent: false,
			},
		},
		{
			Name:            "Compound unsatisfaction",
			SlotsNeeded:     16,
			AgentCapacities: []int{3, 3, 3, 3, 3},
		},
		{
			Name:            "Slots needed should be a multiple of agent capacities",
			SlotsNeeded:     12,
			AgentCapacities: []int{8, 8, 3, 3, 3},
		},
		{
			Name:            "Not all agents need to be used to satisfy",
			SlotsNeeded:     8,
			AgentCapacities: []int{4, 4, 4, 4, 4},
			ExpectedLength:  2,
			FittingRequirements: sproto.FittingRequirements{
				SingleAgent: false,
			},
		},
		{
			Name:            "Heterogeneous fit - exact",
			SlotsNeeded:     4,
			AgentCapacities: []int{2, 1, 1},
			ExpectedLength:  3,
			HeterogenousFit: true,
		},
		{
			Name:            "Heterogeneous fit - not allowed",
			SlotsNeeded:     4,
			AgentCapacities: []int{2, 1, 1},
			ExpectedLength:  0,
			HeterogenousFit: false,
		},
		{
			Name:            "Heterogeneous fit - prefer homoegeneous fit",
			SlotsNeeded:     4,
			AgentCapacities: []int{2, 1, 1, 1, 1},
			ExpectedLength:  4,
			HeterogenousFit: true,
		},
		{
			Name:            "Heterogeneous fit - extra slots not used",
			SlotsNeeded:     4,
			AgentCapacities: []int{2, 1, 1, 1},
			ExpectedLength:  3,
			HeterogenousFit: true,
		},
		{
			Name:            "Heterogeneous fit - not enough consumes none",
			SlotsNeeded:     8,
			AgentCapacities: []int{2, 1, 1},
			ExpectedLength:  0,
			HeterogenousFit: true,
		},
		{
			Name:            "Heterogeneous fit - one last test",
			SlotsNeeded:     63,
			AgentCapacities: []int{32, 32, 16, 16, 8, 8, 4, 4, 2, 2, 1, 1},
			ExpectedLength:  6,
			HeterogenousFit: true,
		},
	}

	for idx := range testCases {
		tc := testCases[idx]

		t.Run(tc.Name, func(t *testing.T) {
			var index []*agentState
			for i, capacity := range tc.AgentCapacities {
				index = append(
					index,
					newFakeAgentState(
						t,
						system,
						fmt.Sprintf("%s-agent-%d", tc.Name, i),
						capacity,
						0,
						100,
						0,
					),
				)
			}
			agents, index := byHandler(index...)
			agentIndex := make(map[*agentState]int)
			for idx, agent := range index {
				agentIndex[agent] = idx
			}

			req := &sproto.AllocateRequest{
				SlotsNeeded:         tc.SlotsNeeded,
				FittingRequirements: tc.FittingRequirements,
			}
			fits := findDedicatedAgentFits(req, agents, WorstFit, tc.HeterogenousFit)

			var agentFit sort.IntSlice
			for _, fit := range fits {
				agentFit = append(agentFit, agentIndex[fit.Agent])
			}

			// For testing distributed fits, we are not too concerned about the
			// tie-breaking strategy or order that agents are assigned to a
			// task, so let's just depend on sorting to get deterministic
			// results rather than setting random seeds.
			sort.Sort(agentFit)

			if tc.ExpectedLength > 0 {
				assert.Equal(t, tc.ExpectedLength, len(agentFit))
			} else {
				assert.DeepEqual(t, tc.ExpectedAgentFit, []int(agentFit))
			}
		})
	}
}

func TestFindFitDisallowedNodes(t *testing.T) {
	system := actor.NewSystem(t.Name())
	agents := []*agentState{
		newFakeAgentState(t, system, "agent1", 4, 0, 100, 0),
		newFakeAgentState(t, system, "agent2", 4, 0, 100, 0),
	}
	agentsByHandler, _ := byHandler(agents...)

	logpattern.SetDisallowedNodesCacheTest(t, map[model.TaskID]*set.Set[string]{
		"noAgents":    ptrs.Ptr(set.FromSlice([]string{"agent1", "agent2"})),
		"notOnAgent1": ptrs.Ptr(set.FromSlice([]string{"agent1"})),
		"notOnAgent2": ptrs.Ptr(set.FromSlice([]string{"agent2"})),
	})

	task := &sproto.AllocateRequest{
		AllocationID: "a",
		SlotsNeeded:  1,
		TaskID:       "noAgents",
	}
	fits := findFits(task, agentsByHandler, BestFit, false)
	assert.Assert(t, len(fits) == 0)

	task = &sproto.AllocateRequest{
		AllocationID: "a",
		SlotsNeeded:  1,
		TaskID:       "notOnAgent1",
	}
	fits = findFits(task, agentsByHandler, BestFit, false)
	assert.Assert(t, len(fits) == 1)
	assert.Equal(t, fits[0].Agent, agents[1])

	task = &sproto.AllocateRequest{
		AllocationID: "a",
		SlotsNeeded:  1,
		TaskID:       "notOnAgent2",
	}
	fits = findFits(task, agentsByHandler, BestFit, false)
	assert.Assert(t, len(fits) == 1)
	assert.Equal(t, fits[0].Agent, agents[0])
}

func byHandler(
	handlers ...*agentState,
) (map[*actor.Ref]*agentState, []*agentState) {
	agents := make(map[*actor.Ref]*agentState, len(handlers))
	index := make([]*agentState, 0, len(handlers))
	for _, agent := range handlers {
		agents[agent.Handler] = agent
		index = append(index, agent)
	}
	return agents, index
}
