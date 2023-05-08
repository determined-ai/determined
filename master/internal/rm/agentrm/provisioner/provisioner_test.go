package provisioner

import (
	"fmt"
	"testing"
	"time"

	"golang.org/x/time/rate"

	"github.com/google/uuid"
	"gotest.tools/assert"

	. "github.com/determined-ai/determined/master/internal/config/provconfig"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/model"
)

type TestInstanceType struct {
	NameString string
	NumSlots   int
}

func (t TestInstanceType) Name() string {
	return t.NameString
}

func (t TestInstanceType) Slots() int {
	return t.NumSlots
}

func newInstanceIDSet(instanceIDs []string) map[string]bool {
	set := make(map[string]bool, len(instanceIDs))
	for _, inst := range instanceIDs {
		set[inst] = true
	}
	return set
}

type mockConfig struct {
	*Config
	maxDisconnectPeriod time.Duration
	instanceType        model.InstanceType
	initInstances       []*model.Instance
	failProvisioning    bool
	numPerProvision     int
}

type mockEnvironment struct {
	cluster     *mockProvider
	system      *actor.System
	provisioner *actor.Ref
}

func newMockEnvironment(t *testing.T, setup *mockConfig) (*mockEnvironment, *Provisioner) {
	system := actor.NewSystem(t.Name())
	cluster, err := newMockProvider(setup)
	assert.NilError(t, err)
	p := &Provisioner{
		provider: cluster,
		scaleDecider: newScaleDecider(
			"default",
			time.Duration(setup.MaxIdleAgentPeriod),
			time.Duration(setup.MaxAgentStartingPeriod),
			setup.maxDisconnectPeriod,
			setup.MinInstances,
			setup.MaxInstances,
			nil,
		),
		telemetryLimiter: rate.NewLimiter(rate.Every(telemetryCooldown), 1),
	}
	if setup.ErrorTimeout != nil {
		p.errorTimeout = time.Duration(*setup.ErrorTimeout)
	}
	provisioner, created := system.ActorOf(actor.Addr("provisioner"), p)
	assert.Assert(t, created)

	environment := mockEnvironment{
		cluster:     cluster,
		system:      system,
		provisioner: provisioner,
	}
	return &environment, p
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
	mockInstanceType model.InstanceType
	maxInstances     int
	instances        map[string]*model.Instance
	failProvisioning bool
	numPerProvision  int
	history          []mockFuncCall
}

func newMockProvider(config *mockConfig) (*mockProvider, error) {
	instMap := make(map[string]*model.Instance, len(config.initInstances))
	for _, inst := range config.initInstances {
		instMap[inst.ID] = inst
	}
	cluster := &mockProvider{
		mockInstanceType: config.instanceType,
		maxInstances:     config.MaxInstances,
		instances:        instMap,
		failProvisioning: config.failProvisioning,
		numPerProvision:  config.numPerProvision,
	}
	return cluster, nil
}

func (c *mockProvider) instanceType() model.InstanceType {
	return c.mockInstanceType
}

func (c *mockProvider) slotsPerInstance() int {
	return c.mockInstanceType.Slots()
}

func (c *mockProvider) list(ctx *actor.Context) ([]*model.Instance, error) {
	c.history = append(c.history, newMockFuncCall("list"))
	instances := make([]*model.Instance, 0, len(c.instances))
	for _, inst := range c.instances {
		instCopy := *inst
		instances = append(instances, &instCopy)
	}
	return instances, nil
}

func (c *mockProvider) prestart(ctx *actor.Context) {}

func (c *mockProvider) launch(ctx *actor.Context, instanceNum int) error {
	switch {
	case c.failProvisioning:
		return c.launchFail()
	case c.numPerProvision > 0:
		return c.launchOne(ctx, c.numPerProvision)
	default:
		return c.launchSuccess(ctx, instanceNum)
	}
}

func (c *mockProvider) launchSuccess(ctx *actor.Context, instanceNum int) error {
	c.history = append(c.history, newMockFuncCall("launch", c.mockInstanceType, instanceNum))
	for i := 0; i < instanceNum; i++ {
		name := uuid.New().String()
		inst := model.Instance{
			ID:         name,
			AgentName:  name,
			LaunchTime: time.Now(),
			State:      model.Running,
		}
		c.instances[inst.ID] = &inst
	}
	return nil
}

func (c *mockProvider) launchOne(ctx *actor.Context, instanceNum int) error {
	c.history = append(c.history, newMockFuncCall("launch", c.mockInstanceType, instanceNum))
	if c.failProvisioning && len(c.instances) == c.maxInstances-1 {
		return fmt.Errorf("max instances reached")
	}
	name := uuid.New().String()
	inst := model.Instance{
		ID:         name,
		AgentName:  name,
		LaunchTime: time.Now(),
		State:      model.Running,
	}
	c.instances[inst.ID] = &inst
	return nil
}

func (c *mockProvider) launchFail() error {
	return fmt.Errorf("failed to launch")
}

func (c *mockProvider) terminate(ctx *actor.Context, instanceIDs []string) {
	c.history = append(c.history, newMockFuncCall("terminate", newInstanceIDSet(instanceIDs)))
	for _, id := range instanceIDs {
		delete(c.instances, id)
	}
}

func TestProvisionerScaleUp(t *testing.T) {
	setup := &mockConfig{
		maxDisconnectPeriod: 5 * time.Minute,
		instanceType: TestInstanceType{
			NameString: "test.instanceType",
			NumSlots:   4,
		},
		Config: &Config{
			MaxInstances: 100,
		},
		initInstances: []*model.Instance{},
	}
	mock, _ := newMockEnvironment(t, setup)
	mock.system.Ask(mock.provisioner, sproto.ScalingInfo{DesiredNewInstances: 4}).Get()
	mock.system.Ask(mock.provisioner, provisionerTick{}).Get()
	assert.NilError(t, mock.system.StopAndAwaitTermination())
	assert.DeepEqual(t, mock.cluster.history, []mockFuncCall{
		newMockFuncCall("list"),
		newMockFuncCall("launch", TestInstanceType{
			NameString: "test.instanceType",
			NumSlots:   4,
		}, 4),
	})
}

func TestProvisionerScaleUpNotPastMax(t *testing.T) {
	setup := &mockConfig{
		maxDisconnectPeriod: 5 * time.Minute,
		instanceType: TestInstanceType{
			NameString: "test.instanceType",
			NumSlots:   4,
		},
		Config: &Config{
			MaxInstances: 1,
		},
		initInstances: []*model.Instance{},
	}
	mock, _ := newMockEnvironment(t, setup)
	mock.system.Ask(mock.provisioner, sproto.ScalingInfo{DesiredNewInstances: 3}).Get()
	mock.system.Ask(mock.provisioner, provisionerTick{}).Get()
	assert.NilError(t, mock.system.StopAndAwaitTermination())
	assert.DeepEqual(t, mock.cluster.history, []mockFuncCall{
		newMockFuncCall("list"),
		newMockFuncCall("launch", TestInstanceType{
			NameString: "test.instanceType",
			NumSlots:   4,
		}, 1),
	})
}

func TestProvisionerScaleDown(t *testing.T) {
	setup := &mockConfig{
		maxDisconnectPeriod: 5 * time.Minute,
		instanceType: TestInstanceType{
			NameString: "test.instanceType",
			NumSlots:   4,
		},
		Config: &Config{
			MaxIdleAgentPeriod: model.Duration(50 * time.Millisecond),
			MaxInstances:       100,
		},
		initInstances: []*model.Instance{
			{
				ID:         "instance1",
				LaunchTime: time.Now().Add(-time.Hour),
				AgentName:  "agent1",
				State:      model.Running,
			},
			{
				ID:         "instance2",
				LaunchTime: time.Now().Add(-time.Minute),
				AgentName:  "agent2",
				State:      model.Running,
			},
		},
	}
	mock, _ := newMockEnvironment(t, setup)

	mock.system.Ask(mock.provisioner, sproto.ScalingInfo{
		DesiredNewInstances: 0,
		Agents: map[string]sproto.AgentSummary{
			"agent1": {Name: "agent1", IsIdle: true},
			"agent2": {Name: "agent2", IsIdle: true},
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
			"instance1",
			"instance2",
		})),
	})
}

func TestProvisionerNotProvisionExtraInstances(t *testing.T) {
	setup := &mockConfig{
		maxDisconnectPeriod: 5 * time.Minute,
		instanceType: TestInstanceType{
			NameString: "test.instanceType",
			NumSlots:   4,
		},
		Config: &Config{
			// If startup period is too short, we might try to re-launch agents.
			MaxAgentStartingPeriod: model.Duration(1 * time.Hour),
			// If idle period is too short, this test might do extra terminate/launch cycles.
			MaxIdleAgentPeriod: model.Duration(1 * time.Hour),
			MaxInstances:       100,
		},
		initInstances: []*model.Instance{
			{
				ID:         "instance1",
				LaunchTime: time.Now().Add(-time.Hour),
				AgentName:  "agent1",
				State:      model.Running,
			},
			{
				ID:         "instance2",
				LaunchTime: time.Now().Add(-time.Minute),
				AgentName:  "agent2",
				State:      model.Running,
			},
		},
	}
	mock, _ := newMockEnvironment(t, setup)

	// Start the master.
	mock.system.Ask(mock.provisioner,
		sproto.ScalingInfo{
			DesiredNewInstances: 0,
			Agents: map[string]sproto.AgentSummary{
				"agent1": {Name: "agent1", IsIdle: true},
				"agent2": {Name: "agent2", IsIdle: true},
				"agent3": {Name: "agent3", IsIdle: true},
			},
		}).Get()
	mock.system.Ask(mock.provisioner, provisionerTick{}).Get()

	// Submit jobs.
	mock.system.Ask(mock.provisioner, sproto.ScalingInfo{
		DesiredNewInstances: 2,
		Agents: map[string]sproto.AgentSummary{
			"agent1": {Name: "agent1", IsIdle: true},
			"agent2": {Name: "agent2", IsIdle: true},
			"agent3": {Name: "agent3", IsIdle: true},
		},
	}).Get()

	// Give the provisioner chances to launch too many instances.
	mock.system.Ask(mock.provisioner, provisionerTick{}).Get()
	mock.system.Ask(mock.provisioner, provisionerTick{}).Get()
	mock.system.Ask(mock.provisioner, provisionerTick{}).Get()
	mock.system.Ask(mock.provisioner, provisionerTick{}).Get()

	assert.NilError(t, mock.system.StopAndAwaitTermination())

	// We should have exactly 1 launch call.
	calls := 0
	for _, call := range mock.cluster.history {
		if call.Name == "launch" {
			calls++
		}
	}

	assert.DeepEqual(t, calls, 1)
}

func TestProvisionerTerminateDisconnectedInstances(t *testing.T) {
	setup := &mockConfig{
		maxDisconnectPeriod: 50 * time.Millisecond,
		instanceType: TestInstanceType{
			NameString: "test.instanceType",
			NumSlots:   4,
		},
		Config: &Config{
			MaxAgentStartingPeriod: model.Duration(3 * time.Minute),
			MaxIdleAgentPeriod:     model.Duration(50 * time.Millisecond),
			MaxInstances:           100,
		},
		initInstances: []*model.Instance{
			{
				ID:         "disconnectedInstance",
				LaunchTime: time.Now().Add(-time.Hour),
				AgentName:  "agent1",
				State:      model.Running,
			},
			{
				ID:         "startingInstance",
				LaunchTime: time.Now().Add(-time.Minute),
				AgentName:  "agent2",
				State:      model.Running,
			},
		},
	}
	mock, _ := newMockEnvironment(t, setup)

	mock.system.Ask(mock.provisioner, sproto.ScalingInfo{}).Get()
	mock.system.Ask(mock.provisioner, provisionerTick{}).Get()
	time.Sleep(100 * time.Millisecond)
	mock.system.Ask(mock.provisioner, provisionerTick{}).Get()

	assert.NilError(t, mock.system.StopAndAwaitTermination())
	assert.DeepEqual(t, mock.cluster.history, []mockFuncCall{
		newMockFuncCall("list"),
		newMockFuncCall("list"),
		newMockFuncCall("terminate", newInstanceIDSet([]string{
			"disconnectedInstance",
		})),
	})
}

func TestProvisionerLaunchFailure(t *testing.T) {
	timeout := model.Duration(5 * time.Second)
	setup := &mockConfig{
		instanceType: TestInstanceType{},
		Config: &Config{
			MaxInstances: 2,
			ErrorTimeout: &timeout,
		},
		failProvisioning: true,
	}
	mock, provisioner := newMockEnvironment(t, setup)

	mock.system.Ask(mock.provisioner, sproto.ScalingInfo{DesiredNewInstances: 4}).Get()
	mock.system.Ask(mock.provisioner, provisionerTick{}).Get()
	assert.Error(t, provisioner.GetError(), "failed to launch", "expected error")
}

func TestProvisionerLaunchOneAtATime(t *testing.T) {
	timeout := model.Duration(5 * time.Second)
	setup := &mockConfig{
		instanceType: TestInstanceType{},
		Config: &Config{
			MaxInstances:        4,
			ErrorTimeout:        &timeout,
			ErrorTimeoutRetries: 4,
		},
		maxDisconnectPeriod: 5 * time.Second,
		numPerProvision:     1,
	}
	mock, provisioner := newMockEnvironment(t, setup)

	mock.system.Ask(mock.provisioner, sproto.ScalingInfo{DesiredNewInstances: 4}).Get()
	mock.system.Ask(mock.provisioner, provisionerTick{}).Get()
	assert.NilError(t, provisioner.GetError(), "received error %t", provisioner.GetError())

	setup.Config.MaxInstances = 3
	mock, provisioner = newMockEnvironment(t, setup)
	mock.system.Ask(mock.provisioner, sproto.ScalingInfo{DesiredNewInstances: 4}).Get()
	mock.system.Ask(mock.provisioner, provisionerTick{}).Get()
	mock.system.Ask(mock.provisioner, provisionerTick{}).Get()
	mock.system.Ask(mock.provisioner, provisionerTick{}).Get()
	assert.NilError(t, provisioner.GetError(), "received error %t", provisioner.GetError())
}

func TestProvisionerLaunchOneAtATimeFail(t *testing.T) {
	timeout := model.Duration(5 * time.Second)
	setup := &mockConfig{
		instanceType: TestInstanceType{},
		Config: &Config{
			MaxInstances:        4,
			ErrorTimeout:        &timeout,
			ErrorTimeoutRetries: 4,
		},
		maxDisconnectPeriod: 5 * time.Second,
		numPerProvision:     1,
		failProvisioning:    true,
	}
	mock, provisioner := newMockEnvironment(t, setup)

	mock.system.Ask(mock.provisioner, sproto.ScalingInfo{DesiredNewInstances: 4}).Get()
	for i := 0; i < 4; i++ {
		mock.system.Ask(mock.provisioner, provisionerTick{}).Get()
	}
	assert.Error(t, provisioner.GetError(), "failed to launch", "expected error")
}
