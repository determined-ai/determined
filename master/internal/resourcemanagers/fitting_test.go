package resourcemanagers

import (
	"fmt"
	"sort"
	"testing"

	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/pkg/actor"
)

func TestIsViable(t *testing.T) {
	system := actor.NewSystem(t.Name())
	req := &AllocateRequest{SlotsNeeded: 2}

	assert.Assert(t, isViable(req, newMockAgent(t, system, "agent1", 4, ""), slotsSatisfied))
	assert.Assert(t, !isViable(req, newMockAgent(t, system, "agent2", 1, ""), slotsSatisfied))
	assert.Assert(t, !isViable(req,
		newMockAgent(t, system, "agent4", 1, ""), slotsSatisfied))
}

func TestFindFit(t *testing.T) {
	system := actor.NewSystem(t.Name())

	type testCase struct {
		Name                string
		SlotsNeeded         int
		AgentCapacities     []int
		AgentOccupiedSlots  []int
		AgentLabels         [2]string
		FittingMethod       SoftConstraint
		ExpectedAgentFit    int
		TaskLabel           string
		FittingRequirements FittingRequirements
	}

	testCases := []testCase{
		{
			Name:             "2-slot multiple fits",
			SlotsNeeded:      2,
			AgentCapacities:  []int{2, 4},
			FittingMethod:    BestFit,
			ExpectedAgentFit: 0,
			FittingRequirements: FittingRequirements{
				SingleAgent: false,
			},
		},
		{
			Name:             "1-slot multiple fits",
			SlotsNeeded:      1,
			AgentCapacities:  []int{1, 4},
			FittingMethod:    BestFit,
			ExpectedAgentFit: 0,
		},
		{
			Name:             "1-slot out-of-order multiple fits",
			SlotsNeeded:      1,
			AgentCapacities:  []int{4, 1},
			FittingMethod:    BestFit,
			ExpectedAgentFit: 1,
		},
		{
			Name:             "4-slot single fit",
			SlotsNeeded:      4,
			AgentCapacities:  []int{4, 1},
			FittingMethod:    BestFit,
			ExpectedAgentFit: 0,
		},
		{
			Name:             "4-slot multiple fits",
			SlotsNeeded:      4,
			AgentCapacities:  []int{4, 4},
			FittingMethod:    BestFit,
			ExpectedAgentFit: 1,
		},
		{
			Name:               "2-slot multiple fits, in-use agents",
			SlotsNeeded:        2,
			AgentCapacities:    []int{2, 4},
			AgentOccupiedSlots: []int{0, 2},
			FittingMethod:      BestFit,
			ExpectedAgentFit:   0,
		},
		{
			Name:               "1-slot multiple fits, in-use agents",
			SlotsNeeded:        1,
			AgentCapacities:    []int{2, 4},
			AgentOccupiedSlots: []int{1, 1},
			FittingMethod:      WorstFit,
			ExpectedAgentFit:   1,
		},
		{
			Name:               "1-slot multiple fits, in-use agents, out of order",
			SlotsNeeded:        1,
			AgentCapacities:    []int{4, 2},
			AgentOccupiedSlots: []int{1, 1},
			FittingMethod:      WorstFit,
			ExpectedAgentFit:   0,
		},
		{
			Name:               "1-slot multiple fits, in-use-agents, odd numbers",
			SlotsNeeded:        1,
			AgentCapacities:    []int{2, 5},
			AgentOccupiedSlots: []int{1, 3},
			FittingMethod:      WorstFit,
			ExpectedAgentFit:   0,
		},
		{
			Name:               "2-slot multiple fits, in-use-agents, odd numbers",
			SlotsNeeded:        2,
			AgentCapacities:    []int{2, 5},
			AgentOccupiedSlots: []int{0, 3},
			FittingMethod:      WorstFit,
			ExpectedAgentFit:   0,
		},
		{
			Name:               "4-slot multiple fits, unoccupied",
			SlotsNeeded:        4,
			AgentCapacities:    []int{4, 4},
			AgentOccupiedSlots: []int{0, 0},
			FittingMethod:      WorstFit,
			ExpectedAgentFit:   1,
		},
		{
			Name:               "4-slot multiple fits, one exact",
			SlotsNeeded:        4,
			AgentCapacities:    []int{8, 4},
			AgentOccupiedSlots: []int{0, 0},
			FittingMethod:      WorstFit,
			ExpectedAgentFit:   1,
			FittingRequirements: FittingRequirements{
				SingleAgent: false,
			},
		},
		{
			Name:               "4-slot multiple fits, one exact, single agent",
			SlotsNeeded:        4,
			AgentCapacities:    []int{8, 4},
			AgentOccupiedSlots: []int{0, 0},
			FittingMethod:      WorstFit,
			ExpectedAgentFit:   1,
			FittingRequirements: FittingRequirements{
				SingleAgent: true,
			},
		},
		{
			Name:             "4-slot multiple fits, label hard constraint",
			SlotsNeeded:      4,
			AgentCapacities:  []int{4, 4},
			AgentLabels:      [2]string{"label1", "label2"},
			FittingMethod:    BestFit,
			TaskLabel:        "label2",
			ExpectedAgentFit: 1,
		},
		{
			Name:             "0-slot multiple fits, label hard constraint",
			SlotsNeeded:      0,
			AgentCapacities:  []int{4, 4},
			AgentLabels:      [2]string{"label1", "label2"},
			FittingMethod:    BestFit,
			TaskLabel:        "label2",
			ExpectedAgentFit: 1,
		},
		{
			Name:            "2-slot multiple inexact fits",
			SlotsNeeded:     2,
			AgentCapacities: []int{4, 4},
			FittingMethod:   BestFit,
			FittingRequirements: FittingRequirements{
				SingleAgent: false,
			},
			ExpectedAgentFit: 1,
		},
		{
			Name:            "2-slot multiple inexact fits, single agent",
			SlotsNeeded:     2,
			AgentCapacities: []int{4, 4},
			FittingMethod:   BestFit,
			FittingRequirements: FittingRequirements{
				SingleAgent: true,
			},
			ExpectedAgentFit: 0,
		},
		{
			Name:            "2-slot fit, single agent",
			SlotsNeeded:     2,
			AgentCapacities: []int{1, 3},
			FittingMethod:   BestFit,
			FittingRequirements: FittingRequirements{
				SingleAgent: true,
			},
			ExpectedAgentFit: 1,
		},
		{
			Name:            "2-slot fit, no single agent requirement",
			SlotsNeeded:     2,
			AgentCapacities: []int{1, 3},
			FittingMethod:   BestFit,
			FittingRequirements: FittingRequirements{
				SingleAgent: false,
			},
			ExpectedAgentFit: 1,
		},
	}

	for idx := range testCases {
		tc := testCases[idx]
		reqID := TaskID(fmt.Sprintf("task%d", idx))

		t.Run(tc.Name, func(t *testing.T) {
			req := &AllocateRequest{
				ID:                  reqID,
				SlotsNeeded:         tc.SlotsNeeded,
				Label:               tc.TaskLabel,
				FittingRequirements: tc.FittingRequirements,
			}

			agent1 := newMockAgent(
				t,
				system,
				fmt.Sprintf("agent-%s-a", reqID),
				tc.AgentCapacities[0],
				tc.AgentLabels[0],
			)

			agent2 := newMockAgent(
				t,
				system,
				fmt.Sprintf("agent-%s-b", reqID),
				tc.AgentCapacities[1],
				tc.AgentLabels[1],
			)

			agents, index := byHandler(agent1, agent2)
			if tc.AgentOccupiedSlots != nil {
				consumeSlots(index[0], tc.AgentOccupiedSlots[0])
				consumeSlots(index[1], tc.AgentOccupiedSlots[1])
			}
			fits := findFits(req, agents, tc.FittingMethod)
			assert.Assert(t, len(fits) > 0)
			assert.Equal(t, fits[0].Agent, index[tc.ExpectedAgentFit])
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
		FittingRequirements FittingRequirements
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
			FittingRequirements: FittingRequirements{
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
			FittingRequirements: FittingRequirements{
				SingleAgent: false,
			},
		},
	}

	for idx := range testCases {
		tc := testCases[idx]

		t.Run(tc.Name, func(t *testing.T) {
			var index []*agentState
			for i, capacity := range tc.AgentCapacities {
				index = append(index,
					newMockAgent(t, system, fmt.Sprintf("%s-agent-%d", tc.Name, i), capacity, ""))
			}
			agents, index := byHandler(index...)
			agentIndex := make(map[*agentState]int)
			for idx, agent := range index {
				agentIndex[agent] = idx
			}

			req := &AllocateRequest{
				SlotsNeeded:         tc.SlotsNeeded,
				FittingRequirements: tc.FittingRequirements,
			}
			fits := findDedicatedAgentFits(req, agents, WorstFit)

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

func byHandler(handlers ...*agentState) (map[*actor.Ref]*agentState, []*agentState) {
	agents := make(map[*actor.Ref]*agentState)
	index := make([]*agentState, 0, len(handlers))
	for _, agent := range handlers {
		agents[agent.handler] = agent
		index = append(index, agent)
	}
	return agents, index
}
