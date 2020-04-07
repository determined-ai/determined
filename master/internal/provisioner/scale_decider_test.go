package provisioner

import (
	"testing"
	"time"

	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/internal/scheduler"
)

func assertEqualInstancesMarked(t *testing.T, left, right map[string]time.Time) {
	const timeErrorTolerance = 2 * time.Second
	for inst, t1 := range left {
		if t2, ok := right[inst]; ok {
			if !t1.Add(-timeErrorTolerance).Before(t2) && t1.Add(timeErrorTolerance).After(t2) {
				t.Errorf("%s\n-: %v\n+: %v", inst, t1, t2)
			}
		} else {
			t.Errorf("%s\n-: %v\n+: <non-existent>", inst, t1)
		}
	}
	for inst := range right {
		if t1, ok := left[inst]; !ok {
			t.Errorf("%s\n-: <non-existent>\n+: %v", inst, t1)
		}
	}
}

func TestFindInstancesToTerminate(t *testing.T) {
	type testcase struct {
		name string

		maxIdleAgentPeriod    time.Duration
		instancesMarkedBefore map[string]time.Time
		instancesMarkedAfter  map[string]time.Time

		idleAgents     []*scheduler.AgentSummary
		instances      []*Instance
		maxInstanceNum int

		toTerminate []string
	}
	var tcs = []testcase{
		{
			name: "stopped instances",
			instances: []*Instance{
				testInstances[5],
			},
			toTerminate: []string{
				testInstances[5].ID,
			},
		},
		{
			name:                 "empty agents",
			instancesMarkedAfter: map[string]time.Time{},
			idleAgents:           []*scheduler.AgentSummary{},
			instances: []*Instance{
				testInstances[0],
				testInstances[1],
			},
			maxInstanceNum: 100,
		},
		{
			name:               "duplicate agents",
			maxIdleAgentPeriod: testTenMin,
			instancesMarkedBefore: map[string]time.Time{
				testInstances[0].ID: testOneHourAgo,
			},
			instancesMarkedAfter: map[string]time.Time{},
			idleAgents: []*scheduler.AgentSummary{
				testAgents[0],
				testAgents[4],
			},
			instances: []*Instance{
				testInstances[0],
			},
			maxInstanceNum: 100,
			toTerminate: []string{
				testInstances[0].ID,
			},
		},
		{
			name:               "idle agents",
			maxIdleAgentPeriod: testTenMin,
			instancesMarkedBefore: map[string]time.Time{
				testInstances[0].ID: testOneHourAgo,
				testInstances[1].ID: testOneHourAgo,
			},
			instancesMarkedAfter: map[string]time.Time{},
			idleAgents: []*scheduler.AgentSummary{
				testAgents[0],
				testAgents[1],
			},
			instances: []*Instance{
				testInstances[0],
				testInstances[1],
			},
			maxInstanceNum: 100,
			toTerminate: []string{
				testInstances[0].ID,
				testInstances[1].ID,
			},
		},
		{
			name:               "unhealthy instances",
			maxIdleAgentPeriod: testTenMin,
			instancesMarkedBefore: map[string]time.Time{
				testInstances[1].ID: testOneHourAgo,
				testInstances[2].ID: testOneHourAgo,
				testInstances[3].ID: testOneHourAgo,
			},
			instancesMarkedAfter: map[string]time.Time{},
			instances: []*Instance{
				testInstances[0],
				testInstances[1],
				testInstances[2],
				testInstances[3],
			},
			maxInstanceNum: 100,
			toTerminate: []string{
				testInstances[1].ID,
				testInstances[2].ID,
				testInstances[3].ID,
			},
		},
		{
			name:               "mark new idle instances without terminating",
			maxIdleAgentPeriod: testTenMin,
			idleAgents: []*scheduler.AgentSummary{
				testAgents[0],
			},
			instances: []*Instance{
				testInstances[0],
			},
			instancesMarkedAfter: map[string]time.Time{
				testInstances[0].ID: testNow,
			},
			maxInstanceNum: 100,
		},
		{
			name:               "terminate marked instances",
			maxIdleAgentPeriod: testTenMin,
			idleAgents: []*scheduler.AgentSummary{
				testAgents[0],
				testAgents[1],
			},
			instances: []*Instance{
				testInstances[0],
				testInstances[1],
			},
			instancesMarkedBefore: map[string]time.Time{
				testInstances[0].ID: testOneMinuteAgo,
				testInstances[1].ID: testOneHourAgo,
			},
			maxInstanceNum: 100,
			toTerminate: []string{
				testInstances[1].ID,
			},
			instancesMarkedAfter: map[string]time.Time{
				testInstances[0].ID: testOneMinuteAgo,
			},
		},
		{
			name:                  "terminate instances to the instance limit",
			maxIdleAgentPeriod:    testTenMin,
			instancesMarkedBefore: map[string]time.Time{},
			idleAgents: []*scheduler.AgentSummary{
				testAgents[0],
				testAgents[1],
			},
			instances: []*Instance{
				testInstances[0],
				testInstances[1],
				testInstances[4],
			},
			maxInstanceNum: 1,
			toTerminate: []string{
				testInstances[0].ID,
				testInstances[1].ID,
			},
		},
		{
			name:               "overall besides the instance limit",
			maxIdleAgentPeriod: testTenMin,
			instancesMarkedBefore: map[string]time.Time{
				testInstances[0].ID: testOneHourAgo,
				testInstances[1].ID: testOneHourAgo,
				testInstances[2].ID: testOneHourAgo,
				testInstances[3].ID: testOneHourAgo,
			},
			instancesMarkedAfter: map[string]time.Time{},
			idleAgents: []*scheduler.AgentSummary{
				testAgents[0],
				testAgents[1],
				testAgents[2],
				testAgents[3],
				testAgents[4],
			},
			instances: []*Instance{
				testInstances[0],
				testInstances[1],
				testInstances[2],
				testInstances[3],
				testInstances[4],
				testInstances[5],
			},
			maxInstanceNum: 100,
			toTerminate: []string{
				testInstances[0].ID,
				testInstances[1].ID,
				testInstances[2].ID,
				testInstances[3].ID,
				testInstances[5].ID,
			},
		},
	}
	for idx := range tcs {
		tc := tcs[idx]
		t.Run(tc.name, func(t *testing.T) {
			s := &scaleDecider{
				maxIdleAgentPeriod: tc.maxIdleAgentPeriod,
				instancesMarked:    tc.instancesMarkedBefore,
				schedulerSnapshot: &scheduler.ViewSnapshot{
					Agents: tc.idleAgents,
				},
				instanceSnapshot: tc.instances,
			}
			toTerminate := s.findInstancesToTerminate(tc.maxInstanceNum)
			assert.DeepEqual(t, newInstanceIDSet(toTerminate), newInstanceIDSet(tc.toTerminate))
			assertEqualInstancesMarked(t, s.instancesMarked, tc.instancesMarkedAfter)
		})
	}
}

func TestCalculateNumInstancesToLaunch(t *testing.T) {
	type testcase struct {
		name string

		maxAgentStartingPeriod time.Duration

		slotsNeeded    []int
		instances      []*Instance
		instanceType   instanceType
		maxInstanceNum int

		numToLaunch int
	}
	var tcs = []testcase{
		{
			name:                   "empty pending tasks",
			maxAgentStartingPeriod: testTenMin,
			slotsNeeded:            nil,
			instanceType:           testInstanceTypes[1],
			maxInstanceNum:         10,
			numToLaunch:            0,
		},
		{
			name:           "zero instance slot",
			slotsNeeded:    []int{1, 2, 1, 1},
			instanceType:   testInstanceTypes[0],
			maxInstanceNum: 10,
			numToLaunch:    0,
		},
		{
			name:           "zero size task",
			slotsNeeded:    []int{0, 0, 0, 0},
			instanceType:   testInstanceTypes[2],
			maxInstanceNum: 10,
			numToLaunch:    1,
		},
		{
			name:           "zero size task",
			slotsNeeded:    []int{0, 2},
			instanceType:   testInstanceTypes[2],
			maxInstanceNum: 10,
			numToLaunch:    1,
		},
		{
			name:           "oversized task",
			slotsNeeded:    []int{1, 2, 3, 1},
			instanceType:   testInstanceTypes[1],
			maxInstanceNum: 10,
			numToLaunch:    7,
		},
		{
			name:           "big task",
			slotsNeeded:    []int{3, 3, 3, 3},
			instanceType:   testInstanceTypes[4],
			maxInstanceNum: 10,
			numToLaunch:    3,
		},
		{
			name:           "fill small gap",
			slotsNeeded:    []int{3, 3, 3, 1},
			instanceType:   testInstanceTypes[4],
			maxInstanceNum: 10,
			numToLaunch:    3,
		},
		{
			name:           "fill gaps",
			slotsNeeded:    []int{3, 2, 1, 1, 3},
			instanceType:   testInstanceTypes[4],
			maxInstanceNum: 10,
			numToLaunch:    3,
		},
		{
			name:           "fill gaps",
			slotsNeeded:    []int{3, 2, 1, 1, 3},
			instanceType:   testInstanceTypes[4],
			maxInstanceNum: 10,
			numToLaunch:    3,
		},
		{
			name:        "max instance num",
			slotsNeeded: []int{3, 3, 3, 3},
			instances: []*Instance{
				testInstances[0],
			},
			instanceType:   testInstanceTypes[4],
			maxInstanceNum: 2,
			numToLaunch:    1,
		},
		{
			name:                   "provision less if having starting instances",
			maxAgentStartingPeriod: testTenMin,
			slotsNeeded:            []int{3, 3, 3, 3},
			instances: []*Instance{
				testInstances[0],
				testInstances[1],
			},
			instanceType:   testInstanceTypes[3],
			maxInstanceNum: 10,
			numToLaunch:    3,
		},
		{
			name:                   "starting instances already more than needed",
			maxAgentStartingPeriod: testTenMin,
			slotsNeeded:            []int{3},
			instances: []*Instance{
				testInstances[0],
				testInstances[1],
			},
			instanceType:   testInstanceTypes[3],
			maxInstanceNum: 10,
			numToLaunch:    0,
		},
	}

	for idx := range tcs {
		tc := tcs[idx]
		t.Run(tc.name, func(t *testing.T) {
			s := &scaleDecider{
				maxAgentStartingPeriod: tc.maxAgentStartingPeriod,
				schedulerSnapshot: &scheduler.ViewSnapshot{
					Tasks: newTasks(tc.slotsNeeded),
				},
				instanceSnapshot: tc.instances,
			}
			actual := s.calculateNumInstancesToLaunch(tc.instanceType, tc.maxInstanceNum)
			assert.Equal(t, actual, tc.numToLaunch)
		})
	}
}
