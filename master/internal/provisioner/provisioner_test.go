package provisioner

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/internal/scheduler"
	"github.com/determined-ai/determined/master/pkg/actor"
)

type TestInstanceType struct {
	Name  string
	Slots int
}

func (t TestInstanceType) name() string {
	return t.Name
}
func (t TestInstanceType) slots() int {
	return t.Slots
}

var (
	testAgents = []*scheduler.AgentSummary{
		{
			Name: "agent1",
		},
		{
			Name: "agent2",
		},
		{
			Name: "agent3",
		},
		{
			Name: "agent4",
		},
		{
			Name: "agent1",
		},
	}
	testOneHourAgo   = time.Now().Add(-time.Hour)
	testOneMinuteAgo = time.Now().Add(-time.Minute)
	testNow          = time.Now()
	testTenMin       = 10 * time.Minute
	testInstances    = []*Instance{
		{
			ID:         "instance1",
			LaunchTime: testOneHourAgo,
			AgentName:  "agent1",
			State:      Running,
		},
		{
			ID:         "instance2",
			LaunchTime: testOneMinuteAgo,
			AgentName:  "agent2",
			State:      Running,
		},
		{
			ID:        "instance3",
			AgentName: "agent2",
			State:     Running,
		},
		{
			ID:    "instance4",
			State: Running,
		},
		{
			ID:        "instance5",
			AgentName: "agent5",
			State:     Running,
		},
		{
			ID:        "instance6",
			AgentName: "agent6",
			State:     Stopped,
		},
	}
	testInstanceTypes = []TestInstanceType{
		{
			Name:  "test.instanceType",
			Slots: 0,
		},
		{
			Name:  "test.instanceType",
			Slots: 1,
		},
		{
			Name:  "test.instanceType",
			Slots: 2,
		},
		{
			Name:  "test.instanceType",
			Slots: 3,
		},
		{
			Name:  "test.instanceType",
			Slots: 4,
		},
	}
)

func newTasks(slotsNeeded []int) []*scheduler.TaskSummary {
	tasks := make([]*scheduler.TaskSummary, 0, len(slotsNeeded))
	for i := range slotsNeeded {
		tasks = append(tasks, &scheduler.TaskSummary{
			SlotsNeeded: slotsNeeded[i],
		})
	}
	return tasks
}

func newInstanceIDSet(instanceIDs []string) map[string]bool {
	set := make(map[string]bool)
	for _, inst := range instanceIDs {
		set[inst] = true
	}
	return set
}

type mockConfig struct {
	initTime               time.Time
	maxAgentStartingPeriod time.Duration
	maxIdleAgentPeriod     time.Duration
	instanceType           instanceType
	maxInstances           int
	initInstances          []*Instance
}

type mockEnvironment struct {
	cluster     *mockProvider
	system      *actor.System
	provisioner *actor.Ref
}

func newMockEnvironment(t *testing.T, setup *mockConfig) *mockEnvironment {
	system := actor.NewSystem(t.Name())
	cluster, err := newMockCluster(setup)
	assert.NilError(t, err)
	p := &Provisioner{
		provider: cluster,
		scaleDecider: &scaleDecider{
			maxAgentStartingPeriod: setup.maxAgentStartingPeriod,
			maxIdleAgentPeriod:     setup.maxIdleAgentPeriod,
		},
	}
	provisioner, created := system.ActorOf(actor.Addr("provisioner"), p)
	assert.Assert(t, created)

	environment := mockEnvironment{
		cluster:     cluster,
		system:      system,
		provisioner: provisioner,
	}
	return &environment
}

type mockFuncCall struct {
	Name       string
	Parameters []interface{}
}

func newMockFuncCall(name string, parameters ...interface{}) mockFuncCall {
	return mockFuncCall{
		Name:       name,
		Parameters: parameters,
	}
}

// mockProvider implements a cluster that accepts requests from the provisioner and responds
// with mock results. It has pre-programmed behavior, which simulates a real provider.
type mockProvider struct {
	mockInstanceType instanceType
	maxInstances     int
	instances        map[string]*Instance
	history          []mockFuncCall
}

func newMockCluster(config *mockConfig) (*mockProvider, error) {
	instMap := make(map[string]*Instance)
	for _, inst := range config.initInstances {
		instMap[inst.ID] = inst
	}
	cluster := &mockProvider{
		mockInstanceType: config.instanceType,
		maxInstances:     config.maxInstances,
		instances:        instMap,
	}
	return cluster, nil
}

func (c *mockProvider) instanceType() instanceType {
	return c.mockInstanceType
}

func (c *mockProvider) maxInstanceNum() int {
	return c.maxInstances
}

func (c *mockProvider) list(ctx *actor.Context) ([]*Instance, error) {
	c.history = append(c.history, newMockFuncCall("list"))
	instances := make([]*Instance, 0, len(c.instances))
	for _, inst := range c.instances {
		instCopy := *inst
		instances = append(instances, &instCopy)
	}
	return instances, nil
}

func (c *mockProvider) launch(ctx *actor.Context, instanceType instanceType, instanceNum int) {
	c.history = append(c.history, newMockFuncCall("launch", instanceType, instanceNum))
	for i := 0; i < instanceNum; i++ {
		name := uuid.New().String()
		inst := Instance{
			ID:         name,
			AgentName:  name,
			LaunchTime: time.Now(),
			State:      Running,
		}
		c.instances[inst.ID] = &inst
	}
}

func (c *mockProvider) terminate(ctx *actor.Context, instanceIDs []string) {
	c.history = append(c.history, newMockFuncCall("terminate", newInstanceIDSet(instanceIDs)))
	for _, id := range instanceIDs {
		delete(c.instances, id)
	}
}

func TestProvisionerScaleUp(t *testing.T) {
	setup := &mockConfig{
		initTime:      time.Now(),
		instanceType:  testInstanceTypes[4],
		maxInstances:  100,
		initInstances: []*Instance{},
	}
	mock := newMockEnvironment(t, setup)
	mock.system.Ask(mock.provisioner, scheduler.ViewSnapshot{
		Tasks: newTasks([]int{1, 2, 3, 4, 5}),
	}).Get()
	mock.system.Ask(mock.provisioner, provisionerTick{}).Get()
	assert.NilError(t, mock.system.StopAndAwaitTermination())
	assert.DeepEqual(t, mock.cluster.history, []mockFuncCall{
		newMockFuncCall("list"),
		newMockFuncCall("launch", testInstanceTypes[4], 4),
	})
}

func TestProvisionerScaleUpNotPastMax(t *testing.T) {
	setup := &mockConfig{
		initTime:      time.Now(),
		instanceType:  testInstanceTypes[4],
		maxInstances:  1,
		initInstances: []*Instance{},
	}
	mock := newMockEnvironment(t, setup)
	mock.system.Ask(mock.provisioner, scheduler.ViewSnapshot{
		Tasks: newTasks([]int{1, 2, 3, 5}),
	}).Get()
	mock.system.Ask(mock.provisioner, provisionerTick{}).Get()
	assert.NilError(t, mock.system.StopAndAwaitTermination())
	assert.DeepEqual(t, mock.cluster.history, []mockFuncCall{
		newMockFuncCall("list"),
		newMockFuncCall("launch", testInstanceTypes[4], 1),
	})
}

func TestProvisionerScaleDown(t *testing.T) {
	setup := &mockConfig{
		initTime:           time.Now(),
		maxIdleAgentPeriod: 50 * time.Millisecond,
		instanceType:       testInstanceTypes[4],
		maxInstances:       100,
		initInstances: []*Instance{
			testInstances[0],
			testInstances[1],
		},
	}
	mock := newMockEnvironment(t, setup)

	mock.system.Ask(mock.provisioner, scheduler.ViewSnapshot{
		Agents: []*scheduler.AgentSummary{
			testAgents[0],
			testAgents[1],
			testAgents[2],
		},
	}).Get()
	mock.system.Ask(mock.provisioner, provisionerTick{}).Get()
	time.Sleep(100 * time.Millisecond)
	mock.system.Ask(mock.provisioner, provisionerTick{}).Get()

	assert.NilError(t, mock.system.StopAndAwaitTermination())
	assert.DeepEqual(t, mock.cluster.history, []mockFuncCall{
		newMockFuncCall("list"),
		newMockFuncCall("list"),
		newMockFuncCall("terminate", newInstanceIDSet([]string{
			testInstances[0].ID,
			testInstances[1].ID,
		})),
	})
}

func TestProvisionerNotProvisionExtraInstances(t *testing.T) {
	setup := &mockConfig{
		initTime:               time.Now(),
		maxAgentStartingPeriod: 100 * time.Millisecond,
		maxIdleAgentPeriod:     100 * time.Millisecond,
		instanceType:           testInstanceTypes[4],
		maxInstances:           100,
		initInstances: []*Instance{
			testInstances[0],
			testInstances[1],
		},
	}
	mock := newMockEnvironment(t, setup)

	// Start the master.
	mock.system.Ask(mock.provisioner, scheduler.ViewSnapshot{
		Agents: []*scheduler.AgentSummary{
			testAgents[0],
			testAgents[1],
			testAgents[2],
		},
	}).Get()
	mock.system.Ask(mock.provisioner, provisionerTick{}).Get()

	// Submit jobs.
	mock.system.Ask(mock.provisioner, scheduler.ViewSnapshot{
		Tasks: newTasks([]int{1, 2, 3}),
	}).Get()
	mock.system.Ask(mock.provisioner, provisionerTick{}).Get()
	time.Sleep(50 * time.Millisecond)
	mock.system.Ask(mock.provisioner, provisionerTick{}).Get()

	assert.NilError(t, mock.system.StopAndAwaitTermination())
	assert.DeepEqual(t, mock.cluster.history[len(mock.cluster.history)-1], newMockFuncCall("list"))
}
