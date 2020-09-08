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

func TestNeedScale(t *testing.T) {
	type testcase struct {
		name string

		scaleDecider scaleDecider

		needScale bool
	}
	var tcs = []testcase{
		{
			name: "no need to scale",
			scaleDecider: scaleDecider{
				maxIdlePeriod:        time.Minute,
				lastProviderUpdated:  time.Now().Add(-time.Hour),
				lastSchedulerUpdated: time.Now().Add(-time.Hour),
				lastProvision:        time.Now(),
			},
			needScale: false,
		},
		{
			name: "minimum retrying interval",
			scaleDecider: scaleDecider{
				maxIdlePeriod:        time.Minute,
				lastProviderUpdated:  time.Now().Add(-time.Hour),
				lastSchedulerUpdated: time.Now().Add(-time.Hour),
				lastProvision:        time.Now().Add(-time.Minute),
			},
			needScale: true,
		},
		{
			name: "last provision < last update + the maximum agent idle period",
			scaleDecider: scaleDecider{
				maxIdlePeriod:        time.Hour,
				lastProviderUpdated:  time.Now().Add(-time.Minute),
				lastSchedulerUpdated: time.Now().Add(-time.Minute),
				lastProvision:        time.Now(),
			},
			needScale: true,
		},
	}
	for idx := range tcs {
		tc := tcs[idx]
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.scaleDecider.needScale(), tc.needScale)
		})
	}
}

func TestFindInstancesToTerminate(t *testing.T) {
	type testcase struct {
		name string

		scaleDecider   scaleDecider
		maxInstanceNum int

		pastIdleInstanceAfter map[string]time.Time
		toTerminate           []string
	}
	var tcs = []testcase{
		{
			name: "terminate stopped instances",
			scaleDecider: scaleDecider{
				instanceSnapshot: map[string]*Instance{
					"instance1": {
						ID:        "instance1",
						AgentName: "agent1",
						State:     Stopped,
					},
					"instance2": {
						ID:         "instance2",
						LaunchTime: time.Now().Add(-time.Hour),
						AgentName:  "agent2",
						State:      Running,
					},
				},
				connectedAgentSnapshot: map[string]*scheduler.AgentSummary{
					"agent1": {
						Name: "agent1",
					},
					"agent2": {
						Name: "agent2",
					},
				},
				launchTime: time.Now(),
			},
			maxInstanceNum:        10,
			pastIdleInstanceAfter: map[string]time.Time{},
			toTerminate: []string{
				"instance1",
			},
		},
		{
			name: "terminate agents that are idle for a long time",
			scaleDecider: scaleDecider{
				maxIdlePeriod: 10 * time.Minute,
				idleAgentSnapshot: map[string]*scheduler.AgentSummary{
					"agent1": {
						Name: "agent1",
					},
					"agent2": {
						Name: "agent2",
					},
					"agent3": {
						Name: "agent3",
					},
				},
				connectedAgentSnapshot: map[string]*scheduler.AgentSummary{
					"agent1": {
						Name: "agent1",
					},
					"agent2": {
						Name: "agent2",
					},
					"agent3": {
						Name: "agent3",
					},
					"agent4": {
						Name: "agent4",
					},
				},
				instanceSnapshot: map[string]*Instance{
					"instance1": {
						ID:         "instance1",
						LaunchTime: time.Now().Add(-time.Hour),
						AgentName:  "agent1",
						State:      Running,
					},
					"instance2": {
						ID:         "instance2",
						LaunchTime: time.Now().Add(-time.Hour),
						AgentName:  "agent2",
						State:      Running,
					},
					"instance3": {
						ID:         "instance3",
						LaunchTime: time.Now().Add(-time.Hour),
						AgentName:  "agent3",
						State:      Running,
					},
					"instance4": {
						ID:         "instance4",
						LaunchTime: time.Now().Add(-time.Hour),
						AgentName:  "agent4",
						State:      Running,
					},
				},
				pastIdleInstances: map[string]time.Time{
					"instance1": time.Now().Add(-time.Hour),
					"instance2": time.Now().Add(-time.Minute),
				},
				launchTime: time.Now(),
			},
			maxInstanceNum: 10,
			pastIdleInstanceAfter: map[string]time.Time{
				"instance2": time.Now(),
				"instance3": time.Now(),
			},
			toTerminate: []string{
				"instance1",
			},
		},
		{
			name: "terminate instances to the instance limit",
			scaleDecider: scaleDecider{
				maxIdlePeriod: 10 * time.Minute,
				idleAgentSnapshot: map[string]*scheduler.AgentSummary{
					"agent1": {
						Name: "agent1",
					},
				},
				connectedAgentSnapshot: map[string]*scheduler.AgentSummary{
					"agent1": {
						Name: "agent1",
					},
					"agent2": {
						Name: "agent2",
					},
					"agent3": {
						Name: "agent3",
					},
				},
				instanceSnapshot: map[string]*Instance{
					"instance1": {
						ID:         "instance1",
						LaunchTime: time.Now().Add(-time.Hour),
						AgentName:  "agent1",
						State:      Running,
					},
					"instance2": {
						ID:         "instance2",
						LaunchTime: time.Now().Add(-time.Hour),
						AgentName:  "agent2",
						State:      Running,
					},
					"instance3": {
						ID:         "instance3",
						LaunchTime: time.Now().Add(-time.Minute),
						AgentName:  "agent3",
						State:      Running,
					},
				},
				pastIdleInstances: map[string]time.Time{},
				launchTime:        time.Now(),
			},
			maxInstanceNum: 1,
			toTerminate: []string{
				"instance1",
				"instance3",
			},
		},
		{
			name: "overall besides the instance limit",
			scaleDecider: scaleDecider{
				maxIdlePeriod: 10 * time.Minute,
				idleAgentSnapshot: map[string]*scheduler.AgentSummary{
					"agent1": {
						Name: "agent1",
					},
					"agent2": {
						Name: "agent2",
					},
					"agent3": {
						Name: "agent3",
					},
					"agent4": {
						Name: "agent4",
					},
				},
				connectedAgentSnapshot: map[string]*scheduler.AgentSummary{
					"agent1": {
						Name: "agent1",
					},
					"agent2": {
						Name: "agent2",
					},
					"agent3": {
						Name: "agent3",
					},
					"agent4": {
						Name: "agent4",
					},
					"agent5": {
						Name: "agent5",
					},
				},
				instanceSnapshot: map[string]*Instance{
					"instance1": {
						ID:         "instance1",
						LaunchTime: time.Now().Add(-time.Hour),
						AgentName:  "agent1",
						State:      Stopped,
					},
					"instance2": {
						ID:         "instance2",
						LaunchTime: time.Now().Add(-time.Hour),
						AgentName:  "agent2",
						State:      Running,
					},
					"instance3": {
						ID:         "instance3",
						LaunchTime: time.Now().Add(-time.Hour),
						AgentName:  "agent3",
						State:      Running,
					},
					"instance4": {
						ID:         "instance4",
						LaunchTime: time.Now().Add(-time.Hour),
						AgentName:  "agent4",
						State:      Running,
					},
					"instance5": {
						ID:         "instance5",
						LaunchTime: time.Now().Add(-time.Hour),
						AgentName:  "agent5",
						State:      Running,
					},
				},
				pastIdleInstances: map[string]time.Time{
					"instance1": time.Now().Add(-time.Hour),
					"instance2": time.Now().Add(-time.Hour),
					"instance3": time.Now().Add(-time.Minute),
				},
				launchTime: time.Now(),
			},
			maxInstanceNum: 10,
			pastIdleInstanceAfter: map[string]time.Time{
				"instance3": time.Now(),
				"instance4": time.Now(),
			},
			toTerminate: []string{
				"instance1",
				"instance2",
			},
		},
		{
			name: "terminate un-connected instances",
			scaleDecider: scaleDecider{
				instanceSnapshot: map[string]*Instance{
					"instance1": {
						ID:         "instance1",
						LaunchTime: time.Now().Add(-time.Hour),
						AgentName:  "agent1",
						State:      Running,
					},
					"instance2": {
						ID:         "instance2",
						LaunchTime: time.Now().Add(-time.Hour),
						AgentName:  "agent2",
						State:      Running,
					},
				},
				connectedAgentSnapshot: map[string]*scheduler.AgentSummary{
					"agent2": {
						Name: "agent2",
					},
				},
				launchTime: time.Now().Add(-time.Hour),
			},
			maxInstanceNum:        10,
			pastIdleInstanceAfter: map[string]time.Time{},
			toTerminate: []string{
				"instance1",
			},
		},
	}
	for idx := range tcs {
		tc := tcs[idx]
		t.Run(tc.name, func(t *testing.T) {
			toTerminate := tc.scaleDecider.findInstancesToTerminate(nil, tc.maxInstanceNum)
			assert.DeepEqual(t, newInstanceIDSet(toTerminate), newInstanceIDSet(tc.toTerminate))
			assertEqualInstancesMarked(t, tc.scaleDecider.pastIdleInstances, tc.pastIdleInstanceAfter)
		})
	}
}

func TestCalculateNumInstancesToLaunch(t *testing.T) {
	type testcase struct {
		name string

		maxAgentStartingPeriod time.Duration

		slotsNeeded      []int
		instanceSnapshot map[string]*Instance
		instanceType     instanceType
		maxInstanceNum   int

		numToLaunch int
	}
	var tcs = []testcase{
		{
			name:                   "empty pending tasks",
			maxAgentStartingPeriod: 10 * time.Minute,
			slotsNeeded:            nil,
			instanceType: TestInstanceType{
				Name:  "test.instanceType",
				Slots: 1,
			},
			maxInstanceNum: 10,
			numToLaunch:    0,
		},
		{
			name:        "zero instance slot",
			slotsNeeded: []int{1, 2, 1, 1},
			instanceType: TestInstanceType{
				Name:  "test.instanceType",
				Slots: 0,
			},
			maxInstanceNum: 10,
			numToLaunch:    0,
		},
		{
			name:        "zero size task",
			slotsNeeded: []int{0, 0, 0, 0},
			instanceType: TestInstanceType{
				Name:  "test.instanceType",
				Slots: 2,
			},
			maxInstanceNum: 10,
			numToLaunch:    1,
		},
		{
			name:        "zero size task",
			slotsNeeded: []int{0, 2},
			instanceType: TestInstanceType{
				Name:  "test.instanceType",
				Slots: 2,
			},
			maxInstanceNum: 10,
			numToLaunch:    1,
		},
		{
			name:        "oversized task",
			slotsNeeded: []int{1, 2, 3, 1},
			instanceType: TestInstanceType{
				Name:  "test.instanceType",
				Slots: 1,
			},
			maxInstanceNum: 10,
			numToLaunch:    7,
		},
		{
			name:        "big task",
			slotsNeeded: []int{3, 3, 3, 3},
			instanceType: TestInstanceType{
				Name:  "test.instanceType",
				Slots: 4,
			},
			maxInstanceNum: 10,
			numToLaunch:    3,
		},
		{
			name:        "fill small gap",
			slotsNeeded: []int{3, 3, 3, 1},
			instanceType: TestInstanceType{
				Name:  "test.instanceType",
				Slots: 4,
			},
			maxInstanceNum: 10,
			numToLaunch:    3,
		},
		{
			name:        "fill gaps",
			slotsNeeded: []int{3, 2, 1, 1, 3},
			instanceType: TestInstanceType{
				Name:  "test.instanceType",
				Slots: 4,
			},
			maxInstanceNum: 10,
			numToLaunch:    3,
		},
		{
			name:        "fill gaps",
			slotsNeeded: []int{3, 2, 1, 1, 3},
			instanceType: TestInstanceType{
				Name:  "test.instanceType",
				Slots: 4,
			},
			maxInstanceNum: 10,
			numToLaunch:    3,
		},
		{
			name:        "max instance num",
			slotsNeeded: []int{3, 3, 3, 3},
			instanceSnapshot: map[string]*Instance{
				"instance1": {
					ID:         "instance1",
					LaunchTime: time.Now().Add(-time.Hour),
					AgentName:  "agent1",
					State:      Running,
				},
			},
			instanceType: TestInstanceType{
				Name:  "test.instanceType",
				Slots: 4,
			},
			maxInstanceNum: 2,
			numToLaunch:    1,
		},
		{
			name:                   "provision less if having starting instances",
			maxAgentStartingPeriod: 10 * time.Minute,
			slotsNeeded:            []int{3, 3, 3, 3},
			instanceSnapshot: map[string]*Instance{
				"instance1": {
					ID:         "instance1",
					LaunchTime: time.Now().Add(-time.Hour),
					AgentName:  "agent1",
					State:      Running,
				},
				"instance2": {
					ID:         "instance2",
					LaunchTime: time.Now().Add(-time.Minute),
					AgentName:  "agent2",
					State:      Running,
				},
			},
			instanceType: TestInstanceType{
				Name:  "test.instanceType",
				Slots: 3,
			},
			maxInstanceNum: 10,
			numToLaunch:    3,
		},
		{
			name:                   "starting instances already more than needed",
			maxAgentStartingPeriod: 10 * time.Minute,
			slotsNeeded:            []int{3},
			instanceSnapshot: map[string]*Instance{
				"instance1": {
					ID:         "instance1",
					LaunchTime: time.Now().Add(-time.Hour),
					AgentName:  "agent1",
					State:      Running,
				},
				"instance2": {
					ID:         "instance2",
					LaunchTime: time.Now().Add(-time.Minute),
					AgentName:  "agent2",
					State:      Running,
				},
			},
			instanceType: TestInstanceType{
				Name:  "test.instanceType",
				Slots: 3,
			},
			maxInstanceNum: 10,
			numToLaunch:    0,
		},
	}

	for idx := range tcs {
		tc := tcs[idx]
		t.Run(tc.name, func(t *testing.T) {
			s := &scaleDecider{
				maxStartingPeriod: tc.maxAgentStartingPeriod,
				taskSnapshot:      newTasks(tc.slotsNeeded),
				instanceSnapshot:  tc.instanceSnapshot,
			}
			actual := s.calculateNumInstancesToLaunch(tc.instanceType, tc.maxInstanceNum)
			assert.Equal(t, actual, tc.numToLaunch)
		})
	}
}
