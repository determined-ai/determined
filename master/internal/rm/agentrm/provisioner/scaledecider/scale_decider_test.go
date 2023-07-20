package scaledecider

import (
	"runtime/debug"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/internal/mocks"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/model"
)

func newInstanceIDSet(instanceIDs []string) map[string]bool {
	set := make(map[string]bool, len(instanceIDs))
	for _, inst := range instanceIDs {
		set[inst] = true
	}
	return set
}

func assertEqualInstancesMarked(t *testing.T, left, right map[string]time.Time) {
	const timeErrorTolerance = 2 * time.Second
	for inst, t1 := range left {
		if t2, ok := right[inst]; ok {
			if !t1.Add(-timeErrorTolerance).Before(t2) && t1.Add(timeErrorTolerance).After(t2) {
				t.Errorf("=== DIFF %s\n-: %v\n+: %v", inst, t1, t2)
				debug.PrintStack()
			}
		} else {
			t.Errorf("=== DIFF %s\n-: %v\n+: <non-existent>", inst, t1)
			debug.PrintStack()
		}
	}
	for inst := range right {
		if t1, ok := left[inst]; !ok {
			t.Errorf("=== DIFF %s\n-: <non-existent>\n+: %v", inst, t1)
			debug.PrintStack()
		}
	}
}

func TestCalculateInstanceStates(t *testing.T) {
	type testcase struct {
		name         string
		scaleDecider ScaleDecider

		disconnected     map[string]time.Time
		idle             map[string]time.Time
		longDisconnected map[string]bool
		longIdle         map[string]bool
		stopped          map[string]bool
		recentlyLaunched map[string]bool
	}
	tcs := []testcase{
		{
			name: "overall",
			scaleDecider: ScaleDecider{
				maxIdlePeriod:       10 * time.Minute,
				maxStartingPeriod:   10 * time.Minute,
				maxDisconnectPeriod: 10 * time.Minute,
				maxInstanceNum:      10,
				instanceSnapshot: map[string]*model.Instance{
					"stopped": {
						ID:         "stopped",
						LaunchTime: time.Now().Add(-time.Hour),
						AgentName:  "stopped",
						State:      model.Stopped,
					},
					"unconnected starting": {
						ID:         "unconnected starting",
						LaunchTime: time.Now().Add(-time.Minute),
						AgentName:  "unconnected starting",
						State:      model.Running,
					},
					"unconnected running": {
						ID:         "unconnected running",
						LaunchTime: time.Now().Add(-time.Minute),
						AgentName:  "unconnected running",
						State:      model.Running,
					},
					"past disconnected": {
						ID:         "past disconnected",
						LaunchTime: time.Now().Add(-time.Hour),
						AgentName:  "past disconnected",
						State:      model.Running,
					},
					"new disconnected": {
						ID:         "new disconnected",
						LaunchTime: time.Now().Add(-time.Hour),
						AgentName:  "new disconnected",
						State:      model.Running,
					},
					"long disconnected": {
						ID:         "long disconnected",
						LaunchTime: time.Now().Add(-time.Hour),
						AgentName:  "long disconnected",
						State:      model.Running,
					},
					"past idle": {
						ID:         "past idle",
						LaunchTime: time.Now().Add(-time.Hour),
						AgentName:  "past idle",
						State:      model.Running,
					},
					"new idle": {
						ID:         "new idle",
						LaunchTime: time.Now().Add(-time.Hour),
						AgentName:  "new idle",
						State:      model.Running,
					},
					"long idle": {
						ID:         "long idle",
						LaunchTime: time.Now().Add(-time.Hour),
						AgentName:  "long idle",
						State:      model.Running,
					},
					"occupied": {
						ID:         "occupied",
						LaunchTime: time.Now().Add(-time.Hour),
						AgentName:  "occupied",
						State:      model.Running,
					},
				},
				connectedAgentSnapshot: map[string]sproto.AgentSummary{
					"past disconnected": {Name: "past disconnected"},
					"past idle":         {Name: "past idle"},
					"new idle":          {Name: "new idle", IsIdle: true},
					"long idle":         {Name: "long idle", IsIdle: true},
					"occupied":          {Name: "occupied", IsIdle: true},
				},
				idleAgentSnapshot: map[string]sproto.AgentSummary{
					"new idle":  {Name: "new idle", IsIdle: true},
					"long idle": {Name: "long idle", IsIdle: true},
				},
				desiredNewInstances: 1,
				disconnected: map[string]time.Time{
					"past disconnected": time.Now().Add(-time.Hour),
					"long disconnected": time.Now().Add(-time.Hour),
				},
				idle: map[string]time.Time{
					"past idle": time.Now().Add(-time.Hour),
					"long idle": time.Now().Add(-time.Hour),
				},
				longDisconnected: map[string]bool{},
				longIdle:         map[string]bool{},
				stopped:          map[string]bool{},
				recentlyLaunched: map[string]bool{},
			},
			disconnected: map[string]time.Time{
				"new disconnected":  time.Now(),
				"long disconnected": time.Now().Add(-time.Hour),
			},
			idle: map[string]time.Time{
				"new idle":  time.Now(),
				"long idle": time.Now().Add(-time.Hour),
			},
			longDisconnected: map[string]bool{
				"long disconnected": true,
			},
			longIdle: map[string]bool{
				"long idle": true,
			},
			stopped: map[string]bool{
				"stopped": true,
			},
			recentlyLaunched: map[string]bool{
				"unconnected starting": true,
				"unconnected running":  true,
			},
		},
	}
	for idx := range tcs {
		tc := &tcs[idx]
		t.Run(tc.name, func(t *testing.T) {
			tc.scaleDecider.CalculateInstanceStates()
			assertEqualInstancesMarked(t, tc.scaleDecider.disconnected, tc.disconnected)
			assertEqualInstancesMarked(t, tc.scaleDecider.idle, tc.idle)
			assert.DeepEqual(t, tc.scaleDecider.longDisconnected, tc.longDisconnected)
			assert.DeepEqual(t, tc.scaleDecider.longIdle, tc.longIdle)
			assert.DeepEqual(t, tc.scaleDecider.stopped, tc.stopped)
			assert.DeepEqual(t, tc.scaleDecider.recentlyLaunched, tc.recentlyLaunched)
		})
	}
}

func TestFindInstancesToTerminate(t *testing.T) {
	type testcase struct {
		name         string
		scaleDecider ScaleDecider
		toTerminate  []string
	}
	tcs := []testcase{
		{
			name: "terminate stopped",
			scaleDecider: ScaleDecider{
				instances:      map[string]*model.Instance{"stopped": {}},
				stopped:        map[string]bool{"stopped": true},
				maxInstanceNum: 10,
			},
			toTerminate: []string{"stopped"},
		},
		{
			name: "terminate long idle",
			scaleDecider: ScaleDecider{
				instances:      map[string]*model.Instance{"long idle": {}},
				longIdle:       map[string]bool{"long idle": true},
				maxInstanceNum: 10,
			},
			toTerminate: []string{"long idle"},
		},
		{
			name: "terminate long disconnected",
			scaleDecider: ScaleDecider{
				instances:        map[string]*model.Instance{"long disconnected": {}},
				longDisconnected: map[string]bool{"long disconnected": true},
				maxInstanceNum:   10,
			},
			toTerminate: []string{"long disconnected"},
		},
		{
			name: "terminate instances until below the maximum",
			scaleDecider: ScaleDecider{
				instances: map[string]*model.Instance{
					"earliest": {
						ID:         "earliest",
						LaunchTime: time.Now().Add(-time.Hour),
						AgentName:  "earliest",
						State:      model.Running,
					},
					"most recent": {
						ID:         "most recent",
						LaunchTime: time.Now().Add(-time.Minute),
						AgentName:  "most recent",
						State:      model.Running,
					},
					"new idle": {
						ID:         "new idle",
						LaunchTime: time.Now().Add(-time.Minute),
						AgentName:  "new idle",
						State:      model.Running,
					},
					"long idle": {
						ID:         "long idle",
						LaunchTime: time.Now().Add(-time.Minute),
						AgentName:  "long idle",
						State:      model.Running,
					},
				},
				idle: map[string]time.Time{
					"new idle":  time.Now(),
					"long idle": time.Now().Add(-10 * time.Minute),
				},
				longIdle:       map[string]bool{"long idle": true},
				maxInstanceNum: 1,
			},
			toTerminate: []string{"new idle", "long idle", "most recent"},
		},
		{
			name: "don't terminate instances if below minimum",
			scaleDecider: ScaleDecider{
				instances: map[string]*model.Instance{
					"stopped":           {},
					"occupied":          {LaunchTime: time.Now().Add(-time.Minute)},
					"past idle":         {LaunchTime: time.Now().Add(-time.Minute)},
					"new idle":          {LaunchTime: time.Now()},
					"long idle":         {},
					"past disconnected": {LaunchTime: time.Now().Add(-time.Minute)},
					"new disconnected":  {LaunchTime: time.Now()},
					"long disconnected": {},
				},
				idleAgentSnapshot: map[string]sproto.AgentSummary{
					"new idle":  {Name: "new idle", IsIdle: true},
					"long idle": {Name: "long idle", IsIdle: true},
				},
				connectedAgentSnapshot: map[string]sproto.AgentSummary{
					"occupied":          {Name: "occupied"},
					"past idle":         {Name: "past idle", IsIdle: true},
					"new idle":          {Name: "new idle", IsIdle: true},
					"long idle":         {Name: "long idle", IsIdle: true},
					"past disconnected": {Name: "past disconnected"},
				},
				idle: map[string]time.Time{
					"past idle": time.Now().Add(-10 * time.Minute),
					"long idle": time.Now().Add(-10 * time.Minute),
				},
				disconnected: map[string]time.Time{
					"past disconnected": time.Now().Add(-10 * time.Minute),
					"long disconnected": time.Now().Add(-10 * time.Minute),
				},
				longIdle:         map[string]bool{"long idle": true},
				longDisconnected: map[string]bool{"long disconnected": true},
				stopped:          map[string]bool{"stopped": true},
				maxInstanceNum:   10,
				minInstanceNum:   6,
			},
			toTerminate: []string{
				"stopped",
				"long disconnected",
			},
		},
	}
	for idx := range tcs {
		tc := &tcs[idx]
		t.Run(tc.name, func(t *testing.T) {
			toTerminate := tc.scaleDecider.FindInstancesToTerminate()
			assert.DeepEqual(
				t,
				newInstanceIDSet(toTerminate.InstanceIDs),
				newInstanceIDSet(tc.toTerminate),
			)
		})
	}
}

func TestCalculateNumInstancesToLaunch(t *testing.T) {
	type testcase struct {
		name         string
		scaleDecider ScaleDecider
		numToLaunch  int
	}
	tcs := []testcase{
		{
			name: "keep above min instance num",
			scaleDecider: ScaleDecider{
				maxStartingPeriod:   time.Minute,
				minInstanceNum:      1,
				maxInstanceNum:      10,
				desiredNewInstances: 1,
			},
			numToLaunch: 1,
		},
		{
			name: "keep under max instance num",
			scaleDecider: ScaleDecider{
				maxStartingPeriod: time.Minute,
				maxInstanceNum:    2,
				instances: map[string]*model.Instance{
					"instance1": {
						ID:         "instance1",
						LaunchTime: time.Now().Add(-time.Hour),
						AgentName:  "agent1",
						State:      model.Running,
					},
				},
				desiredNewInstances: 4,
			},
			numToLaunch: 1,
		},
		{
			name: "provision less if having starting instances",
			scaleDecider: ScaleDecider{
				maxStartingPeriod: 10 * time.Minute,
				maxInstanceNum:    10,
				instances: map[string]*model.Instance{
					"running instance": {
						ID:         "running instance",
						LaunchTime: time.Now().Add(-time.Hour),
						AgentName:  "agent1",
						State:      model.Running,
					},
					"starting instance": {
						ID:         "starting instance",
						LaunchTime: time.Now().Add(-time.Minute),
						AgentName:  "agent2",
						State:      model.Running,
					},
				},
				recentlyLaunched: map[string]bool{
					"starting instance": true,
				},
				desiredNewInstances: 4,
			},
			numToLaunch: 3,
		},
		{
			name: "starting instances already more than needed",
			scaleDecider: ScaleDecider{
				maxStartingPeriod: 10 * time.Minute,
				maxInstanceNum:    10,
				instances: map[string]*model.Instance{
					"instance1": {
						ID:         "instance1",
						LaunchTime: time.Now().Add(-time.Minute),
						AgentName:  "agent1",
						State:      model.Running,
					},
					"instance2": {
						ID:         "instance2",
						LaunchTime: time.Now().Add(-time.Minute),
						AgentName:  "agent2",
						State:      model.Running,
					},
				},
				recentlyLaunched: map[string]bool{
					"instance1": true,
					"instance2": true,
				},
				desiredNewInstances: 1,
			},
			numToLaunch: 0,
		},
	}

	for idx := range tcs {
		tc := &tcs[idx]
		t.Run(tc.name, func(t *testing.T) {
			actual := tc.scaleDecider.CalculateNumInstancesToLaunch()
			assert.Equal(t, actual, tc.numToLaunch)
		})
	}
}

func TestRecordInstanceStats(t *testing.T) {
	db := &mocks.DB{}
	sd := ScaleDecider{
		db: db,
		instances: map[string]*model.Instance{
			"instance1": {
				ID:         "instance1",
				LaunchTime: time.Now().Add(-time.Hour),
				AgentName:  "agent1",
				State:      model.Running,
			},
			"instance2": {
				ID:         "instance2",
				LaunchTime: time.Now().Add(-time.Hour),
				AgentName:  "agent1",
				State:      model.Running,
			},
		},
		disconnected: map[string]time.Time{
			"instance3": time.Now(),
		},
		stopped: map[string]bool{
			"instance4": true,
		},
	}
	db.On("RecordInstanceStats", mock.Anything).Return(nil)
	db.On("EndInstanceStats", mock.Anything).Return(nil)
	err := sd.RecordInstanceStats(2)
	assert.NilError(t, err)
}
