package provisioner

import (
	"testing"
	"time"

	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/internal/sproto"
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

		scaleDecider scaleDecider

		pastIdleInstanceAfter map[string]time.Time
		toTerminate           []string
	}
	var tcs = []testcase{
		{
			name: "terminate stopped instances",
			scaleDecider: scaleDecider{
				instanceSnapshot: map[string]*Instance{
					"stopped instance": {
						ID:        "stopped instance",
						AgentName: "agent1",
						State:     Stopped,
					},
					"running instance": {
						ID:         "running instance",
						LaunchTime: time.Now().Add(-time.Hour),
						AgentName:  "agent2",
						State:      Running,
					},
				},
				connectedAgentSnapshot: map[string]sproto.AgentSummary{
					"agent1": {Name: "agent1"},
					"agent2": {Name: "agent2"},
				},
				maxInstanceNum: 10,
			},
			pastIdleInstanceAfter: map[string]time.Time{},
			toTerminate:           []string{"stopped instance"},
		},
		{
			name: "terminate long idle agents",
			scaleDecider: scaleDecider{
				maxIdlePeriod: 10 * time.Minute,
				idleAgentSnapshot: map[string]sproto.AgentSummary{
					"agent3": {Name: "agent3", IsIdle: true},
					"agent4": {Name: "agent4", IsIdle: true},
					"agent5": {Name: "agent5", IsIdle: true},
				},
				connectedAgentSnapshot: map[string]sproto.AgentSummary{
					"agent1": {Name: "agent1"},
					"agent2": {Name: "agent2"},
					"agent3": {Name: "agent3", IsIdle: true},
					"agent4": {Name: "agent4", IsIdle: true},
					"agent5": {Name: "agent5", IsIdle: true},
				},
				instanceSnapshot: map[string]*Instance{
					"long occupied instance": {
						ID:         "long occupied instance",
						LaunchTime: time.Now().Add(-time.Hour),
						AgentName:  "agent1",
						State:      Running,
					},
					"previous idle instance": {
						ID:         "previous idle instance",
						LaunchTime: time.Now().Add(-time.Hour),
						AgentName:  "agent2",
						State:      Running,
					},
					"new idle instance": {
						ID:         "new idle instance",
						LaunchTime: time.Now().Add(-time.Hour),
						AgentName:  "agent3",
						State:      Running,
					},
					"short idle instance": {
						ID:         "short idle instance",
						LaunchTime: time.Now().Add(-time.Hour),
						AgentName:  "agent4",
						State:      Running,
					},
					"long idle instance": {
						ID:         "long idle instance",
						LaunchTime: time.Now().Add(-time.Hour),
						AgentName:  "agent5",
						State:      Running,
					},
				},
				pastIdleInstances: map[string]time.Time{
					"previous idle instance": time.Now().Add(-time.Hour),
					"short idle instance":    time.Now().Add(-time.Minute),
					"long idle instance":     time.Now().Add(-time.Hour),
				},
				maxInstanceNum: 10,
			},
			pastIdleInstanceAfter: map[string]time.Time{
				"new idle instance":   time.Now(),
				"short idle instance": time.Now(),
			},
			toTerminate: []string{"long idle instance"},
		},
		{
			name: "terminate long disconnected instances",
			scaleDecider: scaleDecider{
				maxDisconnectPeriod: 10 * time.Minute,
				instanceSnapshot: map[string]*Instance{
					"connected instance": {
						ID:         "connected instance",
						LaunchTime: time.Now().Add(-time.Hour),
						AgentName:  "agent1",
						State:      Running,
					},
					"previous disconnected instance": {
						ID:         "previous disconnected instance",
						LaunchTime: time.Now().Add(-time.Hour),
						AgentName:  "agent2",
						State:      Running,
					},
					"new disconnected instance": {
						ID:         "new disconnected instance",
						LaunchTime: time.Now().Add(-time.Hour),
						AgentName:  "agent3",
						State:      Running,
					},
					"short disconnected instance": {
						ID:         "short disconnected instance",
						LaunchTime: time.Now().Add(-time.Hour),
						AgentName:  "agent4",
						State:      Running,
					},
					"long disconnected instance": {
						ID:         "long disconnected instance",
						LaunchTime: time.Now().Add(-time.Hour),
						AgentName:  "agent5",
						State:      Running,
					},
				},
				connectedAgentSnapshot: map[string]sproto.AgentSummary{
					"agent1": {Name: "agent1"},
					"agent2": {Name: "agent2"},
				},
				pastDisconnectedInstances: map[string]time.Time{
					"previous disconnected instance": time.Now().Add(-time.Hour),
					"short disconnected instance":    time.Now().Add(-time.Minute),
					"long disconnected instance":     time.Now().Add(-time.Hour),
				},
				maxInstanceNum: 10,
			},
			pastIdleInstanceAfter: map[string]time.Time{},
			toTerminate:           []string{"long disconnected instance"},
		},
		{
			name: "terminate instances due to max instance",
			scaleDecider: scaleDecider{
				maxIdlePeriod: 10 * time.Minute,
				idleAgentSnapshot: map[string]sproto.AgentSummary{
					"agent1": {Name: "agent1", IsIdle: true},
				},
				connectedAgentSnapshot: map[string]sproto.AgentSummary{
					"agent1": {Name: "agent1", IsIdle: true},
					"agent2": {Name: "agent2"},
					"agent3": {Name: "agent3"},
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
				maxInstanceNum:    1,
			},
			toTerminate: []string{"instance1", "instance3"},
		},
	}
	for idx := range tcs {
		tc := tcs[idx]
		t.Run(tc.name, func(t *testing.T) {
			toTerminate := tc.scaleDecider.findInstancesToTerminate()
			assert.DeepEqual(t, newInstanceIDSet(toTerminate.InstanceIDs), newInstanceIDSet(tc.toTerminate))
			assertEqualInstancesMarked(t, tc.scaleDecider.pastIdleInstances, tc.pastIdleInstanceAfter)
		})
	}
}

func TestCalculateNumInstancesToLaunch(t *testing.T) {
	type testcase struct {
		name         string
		scaleDecider scaleDecider
		numToLaunch  int
	}
	var tcs = []testcase{
		{
			name: "keep under max instance num",
			scaleDecider: scaleDecider{
				maxStartingPeriod: time.Minute,
				maxInstanceNum:    2,
				instanceSnapshot: map[string]*Instance{
					"instance1": {
						ID:         "instance1",
						LaunchTime: time.Now().Add(-time.Hour),
						AgentName:  "agent1",
						State:      Running,
					},
				},
				desiredNewInstances: 4,
			},
			numToLaunch: 1,
		},
		{
			name: "provision less if having starting instances",
			scaleDecider: scaleDecider{
				maxStartingPeriod: 10 * time.Minute,
				maxInstanceNum:    10,
				instanceSnapshot: map[string]*Instance{
					"running instance": {
						ID:         "running instance",
						LaunchTime: time.Now().Add(-time.Hour),
						AgentName:  "agent1",
						State:      Running,
					},
					"starting instance": {
						ID:         "starting instance",
						LaunchTime: time.Now().Add(-time.Minute),
						AgentName:  "agent2",
						State:      Running,
					},
				},
				desiredNewInstances: 4,
			},
			numToLaunch: 3,
		},
		{
			name: "starting instances already more than needed",
			scaleDecider: scaleDecider{
				maxStartingPeriod: 10 * time.Minute,
				maxInstanceNum:    10,
				instanceSnapshot: map[string]*Instance{
					"instance1": {
						ID:         "instance1",
						LaunchTime: time.Now().Add(-time.Minute),
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
				desiredNewInstances: 1,
			},
			numToLaunch: 0,
		},
	}

	for idx := range tcs {
		tc := tcs[idx]
		t.Run(tc.name, func(t *testing.T) {
			actual := tc.scaleDecider.calculateNumInstancesToLaunch()
			assert.Equal(t, actual, tc.numToLaunch)
		})
	}
}
