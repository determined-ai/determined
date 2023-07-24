//go:build integration

package task

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/mocks"
	"github.com/determined-ai/determined/master/internal/portregistry"
	"github.com/determined-ai/determined/master/internal/proxy"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/internal/task/tasklogger"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/aproto"
	"github.com/determined-ai/determined/master/pkg/cproto"
	"github.com/determined-ai/determined/master/pkg/device"
	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/logger"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/syncx/queue"
)

func TestStartAllocation(t *testing.T) {
	db, _, id, q, exitFuture := requireStarted(t)
	defer requireKilled(t, db, id, q, exitFuture)
}

func TestRestoreFailed(t *testing.T) {
	db, _, id, q, exitFuture := requireStarted(t)
	defer requireKilled(t, db, id, q, exitFuture)

	q.Put(&sproto.ResourcesFailure{
		FailureType: sproto.RestoreError,
		ErrMsg:      "things weren't there",
	})
	requireTerminated(t, db, id, exitFuture)
}

func TestInvalidResourcesRequest(t *testing.T) {
	// TODO(DET-9699): Unify InvalidResourcesRequestError and ResourcesFailure code paths.
	db, _, id, q, exitFuture := requireStarted(t)
	defer requireKilled(t, db, id, q, exitFuture)

	q.Put(&sproto.InvalidResourcesRequestError{
		Cause: fmt.Errorf("eternal gke quota error"),
	})
	requireTerminated(t, db, id, exitFuture)
}

type checkWriter struct {
	expected string
	received atomic.Int64
}

// AddTaskLogs implements tasklogger.Writer.
func (c *checkWriter) AddTaskLogs(logs []*model.TaskLog) error {
	for _, l := range logs {
		if strings.Contains(l.Message(), c.expected) {
			c.received.Add(1)
		}
	}
	return nil
}

func TestSendLog(t *testing.T) {
	db, _, id, q, exitFuture := requireStarted(t)
	defer requireKilled(t, db, id, q, exitFuture)

	log := "hello, world"
	wr := checkWriter{expected: log}
	tasklogger.SetDefaultLogger(tasklogger.New(&wr))
	defer tasklogger.SetDefaultLogger(tasklogger.New(&nullWriter{}))

	DefaultService.SendLog(context.TODO(), id, &sproto.ContainerLog{AuxMessage: &log})
	require.True(t, waitForCondition(time.Second, func() bool {
		return wr.received.Load() == 1
	}), "no log within timeout")

	q.Put(&sproto.ContainerLog{AuxMessage: &log})
	require.True(t, waitForCondition(time.Second, func() bool {
		return wr.received.Load() == 2
	}), "no log within timeout")
}

func TestSetReady(t *testing.T) {
	db, _, id, q, exitFuture := requireStarted(t)
	defer requireKilled(t, db, id, q, exitFuture)

	err := DefaultService.SetReady(context.TODO(), id)
	require.NoError(t, err)

	state, dbState := requireState(t, db, id, model.AllocationStateRunning)
	require.True(t, state.Ready)
	require.NotNil(t, dbState.IsReady)
	require.True(t, *dbState.IsReady)
}

func TestSetWaiting(t *testing.T) {
	db, _, id, q, exitFuture := requireStarted(t)
	defer requireKilled(t, db, id, q, exitFuture)

	err := DefaultService.SetWaiting(context.TODO(), id)
	require.NoError(t, err)

	requireState(t, db, id, model.AllocationStateWaiting)
}

func TestSetProxyAddress(t *testing.T) {
	proxy.InitProxy(nil)
	db, _, id, q, exitFuture := requireStarted(t, func(ar *sproto.AllocateRequest) {
		ar.ProxyPorts = append(ar.ProxyPorts, &sproto.ProxyPortConfig{
			ServiceID: "someid",
			Port:      25,
		})
	})
	defer requireKilled(t, db, id, q, exitFuture)

	addr := "localhost"
	err := DefaultService.SetProxyAddress(context.TODO(), id, addr)
	require.NoError(t, err)

	_, dbState := requireState(t, db, id, model.AllocationStatePending)
	require.NotNil(t, dbState.ProxyAddress)
	require.Equal(t, addr, *dbState.ProxyAddress)

	svc := proxy.DefaultProxy.GetService("someid")
	require.NotNil(t, svc)
	require.False(t, svc.ProxyTCP)
}

func TestServiceRendezvous(t *testing.T) {
	db, _, id, q, exitFuture := requireStarted(t)
	defer requireKilled(t, db, id, q, exitFuture)

	rID, _ := requireAssigned(t, db, id, q)
	q.Put(&sproto.ResourcesStateChanged{
		ResourcesID:    rID,
		ResourcesState: sproto.Running,
		ResourcesStarted: &sproto.ResourcesStarted{
			Addresses: []cproto.Address{
				{
					ContainerIP:   "localhost",
					ContainerPort: minLocalRendezvousPort,
					HostIP:        "remotehost",
					HostPort:      minLocalRendezvousPort,
				},
				{
					ContainerIP:   "localhost",
					ContainerPort: 24,
					HostIP:        "remotehost",
					HostPort:      25,
				},
			},
		},
	})
	requireState(t, db, id, model.AllocationStateRunning)

	info, err := DefaultService.WatchRendezvous(context.TODO(), id, rID)
	require.NoError(t, err)
	require.Len(t, info.Addresses, 1)
	require.Equal(t, "remotehost", info.Addresses[0])
}

func TestAllGather(t *testing.T) {
	db, _, id, q, exitFuture := requireStarted(t)
	defer requireKilled(t, db, id, q, exitFuture)

	rID, _ := requireAssigned(t, db, id, q)
	q.Put(&sproto.ResourcesStateChanged{
		ResourcesID:    rID,
		ResourcesState: sproto.Running,
		ResourcesStarted: &sproto.ResourcesStarted{
			Addresses: []cproto.Address{
				{
					ContainerIP:   "localhost",
					ContainerPort: minLocalRendezvousPort,
					HostIP:        "remotehost",
					HostPort:      minLocalRendezvousPort,
				},
				{
					ContainerIP:   "localhost",
					ContainerPort: 24,
					HostIP:        "remotehost",
					HostPort:      25,
				},
			},
		},
	})
	requireState(t, db, id, model.AllocationStateRunning)

	wID := uuid.New()
	msg := "hello world"
	info, err := DefaultService.AllGather(context.TODO(), id, wID, 1, msg)
	require.NoError(t, err)
	require.Len(t, info, 1)
	require.Equal(t, msg, info[0])
}

func TestPreemption(t *testing.T) {
	type args struct {
		sig func(t *testing.T, id model.AllocationID, q *queue.Queue[sproto.ResourcesEvent])
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "user calls terminate",
			args: args{sig: func(
				t *testing.T,
				id model.AllocationID,
				q *queue.Queue[sproto.ResourcesEvent],
			) {
				err := DefaultService.Signal(id, TerminateAllocation, "stop")
				require.NoError(t, err)
			}},
		},
		{
			name: "scheduler sends release resources",
			args: args{sig: func(
				t *testing.T,
				id model.AllocationID,
				q *queue.Queue[sproto.ResourcesEvent],
			) {
				q.Put(&sproto.ReleaseResources{ForcePreemption: true})
			}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, _, id, q, exitFuture := requireStarted(t, func(ar *sproto.AllocateRequest) {
				ar.Preemptible = true
			})
			defer requireKilled(t, db, id, q, exitFuture)

			rID, _ := requireAssigned(t, db, id, q)
			q.Put(&sproto.ResourcesStateChanged{
				ResourcesID:    rID,
				ResourcesState: sproto.Starting,
			})
			requireState(t, db, id, model.AllocationStateStarting)

			q.Put(&sproto.ResourcesStateChanged{
				ResourcesID:      rID,
				ResourcesState:   sproto.Running,
				ResourcesStarted: &sproto.ResourcesStarted{},
			})
			requireState(t, db, id, model.AllocationStateRunning)
			err := DefaultService.SetReady(context.Background(), id)
			require.NoError(t, err)

			tt.args.sig(t, id, q)

			preempted, err := DefaultService.WatchPreemption(context.Background(), id)
			require.NoError(t, err)
			require.True(t, preempted)

			err = DefaultService.AckPreemption(context.Background(), id)
			require.NoError(t, err)
			require.True(t, preempted)

			q.Put(&sproto.ResourcesStateChanged{
				ResourcesID:      rID,
				ResourcesState:   sproto.Terminated,
				ResourcesStopped: &sproto.ResourcesStopped{},
			})
			requireTerminated(t, db, id, exitFuture)
		})
	}
}

func TestSignalBeforeLaunch(t *testing.T) {
	type args struct {
		sig AllocationSignal
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "kill",
			args: args{sig: KillAllocation},
		},
		{
			name: "terminate",
			args: args{sig: TerminateAllocation},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, rm, id, q, exitFuture := requireStarted(t)
			defer requireKilled(t, db, id, q, exitFuture)

			err := DefaultService.Signal(id, tt.args.sig, "some severe reason")
			require.NoError(t, err)

			exit := requireTerminated(t, db, id, exitFuture)
			require.NoError(t, exit.Err)
			require.True(t, rm.AssertExpectations(t), "rm didn't receive release in time")
		})
	}
}

func TestSignalBeforeReady(t *testing.T) {
	type args struct {
		sig AllocationSignal
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "kill",
			args: args{sig: KillAllocation},
		},
		{
			name: "terminate",
			args: args{sig: TerminateAllocation},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, rm, id, q, exitFuture := requireStarted(t)
			defer requireKilled(t, db, id, q, exitFuture)

			_, _ = requireAssigned(t, db, id, q)

			err := DefaultService.Signal(id, tt.args.sig, "some severe reason")
			require.NoError(t, err)

			exit := requireTerminated(t, db, id, exitFuture)
			require.NoError(t, exit.Err)
			require.True(t, rm.AssertExpectations(t), "rm didn't receive release in time")
		})
	}
}

func TestSetResourcesDaemon(t *testing.T) {
	db, rm, id, q, exitFuture := requireStarted(t)
	resources := requireAssignedMany(t, db, id, q, 3)

	t.Log("setting daemon should have no effect (yet)")
	var ranked []sproto.ResourcesID
	for rID := range resources {
		ranked = append(ranked, rID)
	}
	for _, rID := range ranked[1:] {
		err := DefaultService.SetResourcesAsDaemon(context.TODO(), id, rID)
		require.NoError(t, err)
		requireState(t, db, id, model.AllocationStateAssigned) // should still be running
	}

	t.Log("daemon exit should wait on chief")
	q.Put(&sproto.ResourcesStateChanged{
		ResourcesID:      ranked[1],
		ResourcesState:   sproto.Terminated,
		ResourcesStopped: &sproto.ResourcesStopped{},
	})
	requireState(t, db, id, model.AllocationStateTerminating)
	require.False(t, waitForCondition(time.Second, func() bool {
		return exitFuture.Load() != nil
	}), "allocation exited prematurely")

	t.Log("chief exit should bring down the allocation")
	q.Put(&sproto.ResourcesStateChanged{
		ResourcesID:      ranked[0],
		ResourcesState:   sproto.Terminated,
		ResourcesStopped: &sproto.ResourcesStopped{},
	})

	exit := requireTerminated(t, db, id, exitFuture)
	require.NoError(t, exit.Err)
	require.True(t, resources[ranked[2]].AssertExpectations(t), "daemon wasn't killed")
	require.True(t, rm.AssertExpectations(t), "rm didn't receive release in time")
}

func TestStartError(t *testing.T) {
	pgDB := requireDeps(t)

	var rm mocks.ResourceManager
	expectedErr := fmt.Errorf("rm crashed")
	rm.On("Allocate", mock.Anything, mock.Anything).Return(nil, expectedErr)

	taskModel := db.RequireMockTask(t, pgDB, nil)
	ar := stubAllocateRequest(taskModel)
	err := DefaultService.StartAllocation(
		logger.Context{},
		ar,
		pgDB,
		&rm,
		mockTaskSpecifier{},
		actor.NewSystem(uuid.NewString()),
		func(ae *AllocationExited) {},
	)
	require.ErrorContains(t, err, expectedErr.Error())
}

func TestRestore(t *testing.T) {
	pgDB := requireDeps(t)

	restoredTask := db.RequireMockTask(t, pgDB, nil)
	restoredAr := stubAllocateRequest(restoredTask)
	restoredAr.Restore = true

	err := pgDB.AddAllocation(&model.Allocation{
		AllocationID: restoredAr.AllocationID,
		TaskID:       restoredAr.TaskID,
		Slots:        restoredAr.SlotsNeeded,
		ResourcePool: restoredAr.ResourcePool,
		StartTime:    ptrs.Ptr(time.Now().UTC()),
		State:        ptrs.Ptr(model.AllocationStatePending),
	})
	require.NoError(t, err)

	db, _, id, q, exitFuture := requireStarted(t, func(ar *sproto.AllocateRequest) {
		*ar = restoredAr
	})
	defer requireKilled(t, db, id, q, exitFuture)
}

func requireDeps(t *testing.T) *db.PgDB {
	tasklogger.SetDefaultLogger(tasklogger.New(&nullWriter{}))
	portregistry.InitPortRegistry()
	require.NoError(t, etc.SetRootPath("../static/srv"))
	return db.MustSetupTestPostgres(t)
}

func requireStarted(t *testing.T, opts ...func(*sproto.AllocateRequest)) (
	*db.PgDB,
	*mocks.ResourceManager,
	model.AllocationID,
	*queue.Queue[sproto.ResourcesEvent],
	*atomic.Pointer[AllocationExited],
) {
	pgDB := requireDeps(t)

	var rm mocks.ResourceManager

	taskModel := db.RequireMockTask(t, pgDB, nil)

	var subClosed atomic.Bool
	q := queue.New[sproto.ResourcesEvent]()
	sub := sproto.NewAllocationSubscription(q, func() { subClosed.Store(true) })

	rm.On("Allocate", mock.Anything, mock.Anything).Return(sub, nil)
	rm.On("Release", mock.Anything, mock.Anything).Return().Run(func(args mock.Arguments) {
		msg := args[1].(sproto.ResourcesReleased)
		if msg.ResourcesID == nil {
			q.Put(sproto.ResourcesReleasedEvent{})
		}
	})

	var exitFuture atomic.Pointer[AllocationExited]
	ar := stubAllocateRequest(taskModel)
	for _, opt := range opts {
		opt(&ar)
	}
	err := DefaultService.StartAllocation(
		logger.Context{},
		ar,
		pgDB,
		&rm,
		mockTaskSpecifier{},
		actor.NewSystem(uuid.NewString()),
		func(ae *AllocationExited) { exitFuture.Store(ae) },
	)
	require.NoError(t, err)

	waitForCondition(time.Second, func() bool {
		state, err := DefaultService.State(ar.AllocationID)
		return err == nil && state.State == model.AllocationStatePending
	})
	state, err := DefaultService.State(ar.AllocationID)
	require.NoError(t, err)
	require.Nil(t, state.SingleContainer())
	require.Nil(t, state.SingleContainerAddresses())
	require.Contains(t, DefaultService.GetAllAllocationIDs(), ar.AllocationID)

	return pgDB, &rm, ar.AllocationID, q, &exitFuture
}

func stubAllocateRequest(task *model.Task) sproto.AllocateRequest {
	return sproto.AllocateRequest{
		TaskID:       task.TaskID,
		AllocationID: model.AllocationID(fmt.Sprintf("%s.0", task.TaskID)),
		SlotsNeeded:  2,
		Preemptible:  true,
		ResourcePool: stubResourcePoolName,
	}
}

var stubResourcePoolName = "default"

var stubAgentName = aproto.ID("agentx")

func requireAssigned(
	t *testing.T,
	db *db.PgDB,
	id model.AllocationID,
	q *queue.Queue[sproto.ResourcesEvent],
) (sproto.ResourcesID, *mocks.Resources) {
	for rID, r := range requireAssignedMany(t, db, id, q, 1) {
		return rID, r
	}
	panic("impossible")
}

func requireAssignedMany(
	t *testing.T,
	db *db.PgDB,
	id model.AllocationID,
	q *queue.Queue[sproto.ResourcesEvent],
	numResources int,
) map[sproto.ResourcesID]*mocks.Resources {
	resources := map[sproto.ResourcesID]*mocks.Resources{}
	assigned := map[sproto.ResourcesID]sproto.Resources{}
	for i := 0; i < numResources; i++ {
		rID := sproto.ResourcesID(cproto.NewID())
		var r mocks.Resources
		r.On("Start", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(nil).Times(1)
		r.On("Summary").Return(sproto.ResourcesSummary{
			AllocationID:  id,
			ResourcesID:   rID,
			ResourcesType: sproto.ResourcesTypeDockerContainer,
			AgentDevices:  map[aproto.ID][]device.Device{stubAgentName: nil},
		})
		r.On("Kill", mock.Anything, mock.Anything).Return().Run(func(_ mock.Arguments) {
			q.Put(&sproto.ResourcesStateChanged{
				ResourcesID:    rID,
				ResourcesState: sproto.Terminated,
				ResourcesStopped: &sproto.ResourcesStopped{
					Failure: &sproto.ResourcesFailure{
						FailureType: sproto.TaskError,
						ErrMsg:      "exit code 137",
						ExitCode:    ptrs.Ptr(sproto.ExitCode(137)),
					},
				},
			})
		})
		resources[rID] = &r
		assigned[rID] = &r
	}

	q.Put(&sproto.ResourcesAllocated{
		ID:           id,
		ResourcePool: stubResourcePoolName,
		Resources:    assigned,
	})
	requireState(t, db, id, model.AllocationStateAssigned)
	return resources
}

func requireKilled(
	t *testing.T,
	db *db.PgDB,
	id model.AllocationID,
	q *queue.Queue[sproto.ResourcesEvent],
	exitFuture *atomic.Pointer[AllocationExited],
) *AllocationExited {
	if ae := exitFuture.Load(); ae != nil {
		return ae
	}

	_ = DefaultService.Signal(id, KillAllocation, "cleanup for tests")
	return requireTerminated(t, db, id, exitFuture)
}

func requireTerminated(
	t *testing.T,
	db *db.PgDB,
	id model.AllocationID,
	exitFuture *atomic.Pointer[AllocationExited],
) *AllocationExited {
	require.True(t, waitForCondition(time.Second, func() bool {
		return exitFuture.Load() != nil
	}), "allocation did not exit in time")
	exit := exitFuture.Load()
	require.True(t, exit.FinalState.State == model.AllocationStateTerminated)
	requireDBState(t, db, id, model.AllocationStateTerminated)
	return exit
}

func requireState(
	t *testing.T,
	db *db.PgDB,
	id model.AllocationID,
	state model.AllocationState,
) (AllocationState, *model.Allocation) {
	return requireAllocationState(t, id, state), requireDBState(t, db, id, state)
}

func requireAllocationState(
	t *testing.T,
	id model.AllocationID,
	expected model.AllocationState,
) AllocationState {
	var state AllocationState
	require.True(t, waitForCondition(5*time.Second, func() bool {
		s, err := DefaultService.State(id)
		if err != nil {
			return false
		}
		state = s

		switch actual := s.State; {
		case expected == actual:
			state = s
			return true
		case model.MostProgressedAllocationState(actual, expected) == actual:
			require.Fail(t, fmt.Sprintf("state progressed past expected (%s > %s)", actual, expected))
			return false
		case model.MostProgressedAllocationState(actual, expected) == expected:
			return false
		default:
			panic("impossible")
		}
	}), fmt.Errorf("never reached state %s (got %s)", expected, state.State))
	return state
}

func requireDBState(
	t *testing.T,
	db *db.PgDB,
	id model.AllocationID,
	expected model.AllocationState,
) *model.Allocation {
	dbState, err := db.AllocationByID(id)
	require.NoError(t, err)
	require.NotNil(t, dbState.State)
	require.Equal(t, expected, *dbState.State)
	return dbState
}
