package scheduler

import (
	"fmt"
	"sort"
	"strconv"
	"testing"

	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/pkg/actor"
)

func TestIsViable(t *testing.T) {
	system := actor.NewSystem(t.Name())
	task := newTask(&Task{
		slotsNeeded: 2,
	})

	assert.Assert(t, isViable(task, newMockAgent(t, system, "agent1", 4, ""), slotsSatisfied))
	assert.Assert(t, !isViable(task, newMockAgent(t, system, "agent2", 1, ""), slotsSatisfied))
	assert.Assert(t, !isViable(task,
		newMockAgent(t, system, "agent4", 1, ""), slotsSatisfied))
}

func TestFindFit(t *testing.T) {
	system := actor.NewSystem(t.Name())

	type testCase struct {
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
			SlotsNeeded:      2,
			AgentCapacities:  []int{2, 4},
			FittingMethod:    BestFit,
			ExpectedAgentFit: 0,
			FittingRequirements: FittingRequirements{
				SingleAgent:    false,
				DedicatedAgent: true,
			},
		},
		{
			SlotsNeeded:      1,
			AgentCapacities:  []int{1, 4},
			FittingMethod:    BestFit,
			ExpectedAgentFit: 0,
		},
		{
			SlotsNeeded:      1,
			AgentCapacities:  []int{4, 1},
			FittingMethod:    BestFit,
			ExpectedAgentFit: 1,
		},
		{
			SlotsNeeded:      4,
			AgentCapacities:  []int{4, 1},
			FittingMethod:    BestFit,
			ExpectedAgentFit: 0,
		},
		{
			SlotsNeeded:      4,
			AgentCapacities:  []int{4, 4},
			FittingMethod:    BestFit,
			ExpectedAgentFit: 1,
		},
		{
			SlotsNeeded:        2,
			AgentCapacities:    []int{2, 4},
			AgentOccupiedSlots: []int{0, 2},
			FittingMethod:      BestFit,
			ExpectedAgentFit:   0,
		},
		{
			SlotsNeeded:        1,
			AgentCapacities:    []int{2, 4},
			AgentOccupiedSlots: []int{1, 1},
			FittingMethod:      WorstFit,
			ExpectedAgentFit:   1,
		},
		{
			SlotsNeeded:        1,
			AgentCapacities:    []int{4, 2},
			AgentOccupiedSlots: []int{1, 1},
			FittingMethod:      WorstFit,
			ExpectedAgentFit:   0,
		},
		{
			SlotsNeeded:        1,
			AgentCapacities:    []int{2, 5},
			AgentOccupiedSlots: []int{1, 3},
			FittingMethod:      WorstFit,
			ExpectedAgentFit:   0,
		},
		{
			SlotsNeeded:        2,
			AgentCapacities:    []int{2, 5},
			AgentOccupiedSlots: []int{0, 3},
			FittingMethod:      WorstFit,
			ExpectedAgentFit:   0,
		},
		{
			SlotsNeeded:        4,
			AgentCapacities:    []int{4, 4},
			AgentOccupiedSlots: []int{0, 0},
			FittingMethod:      WorstFit,
			ExpectedAgentFit:   1,
		},
		{
			SlotsNeeded:        4,
			AgentCapacities:    []int{8, 4},
			AgentOccupiedSlots: []int{0, 0},
			FittingMethod:      WorstFit,
			ExpectedAgentFit:   1,
			FittingRequirements: FittingRequirements{
				SingleAgent:    false,
				DedicatedAgent: true,
			},
		},
		{
			SlotsNeeded:        4,
			AgentCapacities:    []int{8, 4},
			AgentOccupiedSlots: []int{0, 0},
			FittingMethod:      WorstFit,
			ExpectedAgentFit:   1,
			FittingRequirements: FittingRequirements{
				SingleAgent:    true,
				DedicatedAgent: false,
			},
		},
		{
			SlotsNeeded:        4,
			AgentCapacities:    []int{8, 4},
			AgentOccupiedSlots: []int{0, 0},
			FittingMethod:      WorstFit,
			ExpectedAgentFit:   0,
			FittingRequirements: FittingRequirements{
				SingleAgent:    false,
				DedicatedAgent: false,
			},
		},
		{
			SlotsNeeded:      4,
			AgentCapacities:  []int{4, 4},
			AgentLabels:      [2]string{"label1", "label2"},
			FittingMethod:    BestFit,
			TaskLabel:        "label2",
			ExpectedAgentFit: 1,
		},
		{
			SlotsNeeded:      0,
			AgentCapacities:  []int{4, 4},
			AgentLabels:      [2]string{"label1", "label2"},
			FittingMethod:    BestFit,
			TaskLabel:        "label2",
			ExpectedAgentFit: 1,
		},
	}

	for idx := range testCases {
		tc := testCases[idx]
		taskID := TaskID(fmt.Sprintf("task%d", idx))

		t.Run(strconv.Itoa(idx), func(t *testing.T) {
			task := newTask(&Task{
				ID:                  taskID,
				slotsNeeded:         tc.SlotsNeeded,
				agentLabel:          tc.TaskLabel,
				fittingRequirements: tc.FittingRequirements,
			})

			agent1 := newMockAgent(
				t,
				system,
				fmt.Sprintf("agent-%s-a", taskID),
				tc.AgentCapacities[0],
				tc.AgentLabels[0],
			)

			agent2 := newMockAgent(
				t,
				system,
				fmt.Sprintf("agent-%s-b", taskID),
				tc.AgentCapacities[1],
				tc.AgentLabels[1],
			)

			agents, index := byHandler(agent1, agent2)
			if tc.AgentOccupiedSlots != nil {
				consumeSlots(index[0], tc.AgentOccupiedSlots[0])
				consumeSlots(index[1], tc.AgentOccupiedSlots[1])
			}
			fits := findFits(task, agents, tc.FittingMethod)
			assert.Assert(t, len(fits) > 0)
			assert.Equal(t, fits[0].Agent, index[tc.ExpectedAgentFit])
		})
	}
}

func TestFindMultiSlotFits(t *testing.T) {
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
				SingleAgent:    false,
				DedicatedAgent: true,
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
				SingleAgent:    false,
				DedicatedAgent: true,
			},
		},
		{
			Name:            "Scheduling a multi-slot command",
			SlotsNeeded:     2,
			AgentCapacities: []int{4, 4, 4, 4, 4},
			ExpectedLength:  1,
			FittingRequirements: FittingRequirements{
				SingleAgent:    true,
				DedicatedAgent: false,
			},
		},
		{
			Name:             "Scheduling a multi-slot command should pick smallest possible agent",
			SlotsNeeded:      2,
			AgentCapacities:  []int{4, 1, 3, 4, 4},
			ExpectedAgentFit: []int{2},
			FittingRequirements: FittingRequirements{
				SingleAgent:    true,
				DedicatedAgent: false,
			},
		},
		{
			Name:            "Scheduling a multi-slot command that's too big",
			SlotsNeeded:     8,
			AgentCapacities: []int{4, 4, 4},
			FittingRequirements: FittingRequirements{
				SingleAgent:    true,
				DedicatedAgent: false,
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

			fits := findMultiSlotFits(newTask(&Task{
				slotsNeeded:         tc.SlotsNeeded,
				fittingRequirements: tc.FittingRequirements,
			}), agents, WorstFit)

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
