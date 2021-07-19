package internal

import (
	"crypto/rand"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/internal/mocks"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	aproto "github.com/determined-ai/determined/master/pkg/agent"
	"github.com/determined-ai/determined/master/pkg/archive"
	"github.com/determined-ai/determined/master/pkg/device"
	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/master/pkg/searcher"
	"github.com/determined-ai/determined/master/pkg/tasks"

	"github.com/davecgh/go-spew/spew"

	cproto "github.com/determined-ai/determined/master/pkg/container"

	"github.com/google/uuid"
)

type (
	forward struct {
		to  *actor.Ref
		msg actor.Message
	}
	mockActor struct {
		messages []actor.Message
	}
)

func (a *mockActor) Receive(ctx *actor.Context) error {
	a.messages = append(a.messages, ctx.Message())
	switch msg := ctx.Message().(type) {
	case error:
		return msg
	case forward:
		if ctx.ExpectingResponse() {
			ctx.Respond(ctx.Ask(msg.to, msg.msg).Get())
		}
	default:
		if ctx.ExpectingResponse() {
			ctx.Respond(ctx.Message())
		}
	}
	return nil
}

func TestTrialMultiAlloc(t *testing.T) {
	require.NoError(t, etc.SetRootPath("../static/srv"))
	system := actor.NewSystem("system")

	// mock resource manager.
	rmImpl := mockActor{}
	rm := system.MustActorOf(actor.Addr("rm"), &rmImpl)

	// mock logger.
	loggerImpl := mockActor{}
	logger := system.MustActorOf(actor.Addr("logger"), &loggerImpl)

	// mock db.
	db := &mocks.DB{}

	// instantiate the trial
	rID := model.NewRequestID(rand.Reader)
	tr := newTrial(
		1,
		model.PausedState,
		TrialSearcherState{Create: searcher.Create{RequestID: rID}, Complete: true},
		rm, logger,
		db,
		schemas.WithDefaults(expconf.ExperimentConfig{
			RawCheckpointStorage: &expconf.CheckpointStorageConfigV0{
				RawSharedFSConfig: &expconf.SharedFSConfig{
					RawHostPath:      ptrs.StringPtr("/tmp"),
					RawContainerPath: ptrs.StringPtr("determined-sharedfs"),
				},
			},
		}).(expconf.ExperimentConfig),
		&model.Checkpoint{},
		&tasks.TaskSpec{
			AgentUserGroup: &model.AgentUserGroup{},
		},
		archive.Archive{},
	)
	self := system.MustActorOf(actor.Addr("trial"), tr)

	// Pre-scheduled stage.
	require.NoError(t, system.Ask(self, model.ActiveState).Error())
	require.NoError(t, system.Ask(self, TrialSearcherState{
		Create: searcher.Create{RequestID: rID},
		Op: searcher.ValidateAfter{
			RequestID: rID,
			Length:    expconf.NewLengthInBatches(10),
		},
		Complete: false,
		Closed:   true,
	}).Error())
	require.NotNil(t, tr.task)
	require.Contains(t, rmImpl.messages, *tr.task)

	// Pre-allocated stage.
	mockAlloc := func(cID cproto.ID, agentID string) sproto.Allocation {
		alloc := &mocks.Allocation{}
		alloc.On("Start", mock.Anything, mock.Anything, mock.Anything).Return()
		alloc.On("Summary").Return(sproto.ContainerSummary{
			TaskID: tr.task.ID,
			ID:     cID,
			Agent:  agentID,
		})
		alloc.On("Kill", mock.Anything).Return()
		return alloc
	}

	allocations := []sproto.Allocation{
		mockAlloc(cproto.NewID(), "agent-1"),
		mockAlloc(cproto.NewID(), "agent-2"),
	}
	db.On("AddTrial", mock.Anything).Return(nil)
	db.On("UpdateTrialRunID", 0, 1).Return(nil)
	db.On("AddTask", mock.Anything, 0, tr.task.ID).Return(nil)
	db.On("StartTaskSession", string(tr.task.ID)).Return("", nil)
	db.On("LatestCheckpointForTrial", 0).Return(&model.Checkpoint{}, nil)
	require.NoError(t, system.Ask(rm, forward{
		to: self,
		msg: sproto.ResourcesAllocated{
			ID:           tr.task.ID,
			ResourcePool: "default",
			Allocations:  allocations,
		},
	}).Error())
	require.NotEmpty(t, tr.allocations)

	// Pre-ready stage.
	for _, a := range allocations {
		containerStateChanged := sproto.TaskContainerStateChanged{
			Container: cproto.Container{
				Parent:  actor.Address{},
				ID:      a.Summary().ID,
				State:   cproto.Assigned,
				Devices: []device.Device{},
			},
		}
		require.NoError(t, system.Ask(self, containerStateChanged).Error())
		containerStateChanged.Container.State = cproto.Pulling
		require.NoError(t, system.Ask(self, containerStateChanged).Error())
		containerStateChanged.Container.State = cproto.Starting
		require.NoError(t, system.Ask(self, containerStateChanged).Error())
		containerStateChanged.Container.State = cproto.Running
		containerStateChanged.ContainerStarted = &sproto.TaskContainerStarted{
			Addresses: []cproto.Address{
				{
					ContainerIP:   "0.0.0.0",
					ContainerPort: 1734,
					HostIP:        a.Summary().Agent,
					HostPort:      1734,
				},
			},
		}
		require.NoError(t, system.Ask(self, containerStateChanged).Error())
		containerStateChanged.ContainerStarted = nil
		require.NoError(t, system.Ask(self, watchRendezvousInfo{
			taskID: tr.task.ID,
			id:     a.Summary().ID,
		}).Error())
	}
	require.True(t, tr.rendezvous.ready())

	// Running stage.
	require.NoError(t, system.Ask(self, TrialSearcherState{
		Create: searcher.Create{RequestID: rID},
		Op: searcher.ValidateAfter{
			RequestID: rID,
			Length:    expconf.NewLengthInBatches(10),
		},
		Complete: true,
		Closed:   true,
	}).Error())

	// Terminating stage.
	db.On("DeleteTaskSessionByTaskID", string(tr.task.ID)).Return(nil)
	db.On("CompleteTask", mock.Anything, 0, tr.task.ID).Return(nil)
	db.On("EndTasks", mock.Anything, 0).Return(nil)
	db.On("UpdateTrial", 0, model.CompletedState).Return(nil)
	for _, a := range allocations {
		containerStateChanged := sproto.TaskContainerStateChanged{
			Container: cproto.Container{
				Parent:  actor.Address{},
				ID:      a.Summary().ID,
				State:   cproto.Terminated,
				Devices: []device.Device{},
			},
			ContainerStopped: &sproto.TaskContainerStopped{
				ContainerStopped: aproto.ContainerStopped{
					Failure: nil,
				},
			},
		}
		require.NoError(t, system.Ask(self, containerStateChanged).Error())
	}

	require.True(t, tr.stopped)
	require.NoError(t, self.AwaitTermination())
}

func TestTrialDelayedSearcherClose(t *testing.T) {
	require.NoError(t, etc.SetRootPath("../static/srv"))
	system := actor.NewSystem("system")

	// mock resource manager.
	rmImpl := mockActor{}
	rm := system.MustActorOf(actor.Addr("rm"), &rmImpl)

	// mock logger.
	loggerImpl := mockActor{}
	logger := system.MustActorOf(actor.Addr("logger"), &loggerImpl)

	// mock db.
	db := &mocks.DB{}

	// instantiate the trial
	rID := model.NewRequestID(rand.Reader)
	tr := newTrial(
		1,
		model.PausedState,
		TrialSearcherState{Create: searcher.Create{RequestID: rID}, Complete: true},
		rm, logger,
		db,
		schemas.WithDefaults(expconf.ExperimentConfig{
			RawCheckpointStorage: &expconf.CheckpointStorageConfigV0{
				RawSharedFSConfig: &expconf.SharedFSConfig{
					RawHostPath:      ptrs.StringPtr("/tmp"),
					RawContainerPath: ptrs.StringPtr("determined-sharedfs"),
				},
			},
		}).(expconf.ExperimentConfig),
		&model.Checkpoint{},
		&tasks.TaskSpec{
			AgentUserGroup: &model.AgentUserGroup{},
		},
		archive.Archive{},
	)
	self := system.MustActorOf(actor.Addr("trial"), tr)

	// Pre-scheduled stage.
	require.NoError(t, system.Ask(self, model.ActiveState).Error())
	require.NoError(t, system.Ask(self, TrialSearcherState{
		Create: searcher.Create{RequestID: rID},
		Op: searcher.ValidateAfter{
			RequestID: rID,
			Length:    expconf.NewLengthInBatches(10),
		},
		Complete: false,
		Closed:   true,
	}).Error())
	require.NotNil(t, tr.task)
	require.Contains(t, rmImpl.messages, *tr.task)

	// Pre-allocated stage.
	mockAlloc := func(cID cproto.ID, agentID string) sproto.Allocation {
		alloc := &mocks.Allocation{}
		alloc.On("Start", mock.Anything, mock.Anything, mock.Anything).Return()
		alloc.On("Summary").Return(sproto.ContainerSummary{
			TaskID: tr.task.ID,
			ID:     cID,
			Agent:  agentID,
		})
		alloc.On("Kill", mock.Anything).Return()
		return alloc
	}

	allocations := []sproto.Allocation{
		mockAlloc(cproto.NewID(), "agent-1"),
		mockAlloc(cproto.NewID(), "agent-2"),
	}
	db.On("AddTrial", mock.Anything).Return(nil)
	db.On("UpdateTrialRunID", 0, 1).Return(nil)
	db.On("AddTask", mock.Anything, 0, tr.task.ID).Return(nil)
	db.On("StartTaskSession", string(tr.task.ID)).Return("", nil)
	db.On("LatestCheckpointForTrial", 0).Return(&model.Checkpoint{}, nil)
	require.NoError(t, system.Ask(rm, forward{
		to: self,
		msg: sproto.ResourcesAllocated{
			ID:           tr.task.ID,
			ResourcePool: "default",
			Allocations:  allocations,
		},
	}).Error())
	require.NotEmpty(t, tr.allocations)

	// Pre-ready stage.
	for _, a := range allocations {
		containerStateChanged := sproto.TaskContainerStateChanged{
			Container: cproto.Container{
				Parent:  actor.Address{},
				ID:      a.Summary().ID,
				State:   cproto.Assigned,
				Devices: []device.Device{},
			},
		}
		require.NoError(t, system.Ask(self, containerStateChanged).Error())
		containerStateChanged.Container.State = cproto.Pulling
		require.NoError(t, system.Ask(self, containerStateChanged).Error())
		containerStateChanged.Container.State = cproto.Starting
		require.NoError(t, system.Ask(self, containerStateChanged).Error())
		containerStateChanged.Container.State = cproto.Running
		containerStateChanged.ContainerStarted = &sproto.TaskContainerStarted{
			Addresses: []cproto.Address{
				{
					ContainerIP:   "0.0.0.0",
					ContainerPort: 1734,
					HostIP:        a.Summary().Agent,
					HostPort:      1734,
				},
			},
		}
		require.NoError(t, system.Ask(self, containerStateChanged).Error())
		containerStateChanged.ContainerStarted = nil
		require.NoError(t, system.Ask(self, watchRendezvousInfo{
			taskID: tr.task.ID,
			id:     a.Summary().ID,
		}).Error())
	}
	require.True(t, tr.rendezvous.ready())

	// Running stage.
	require.NoError(t, system.Ask(self, TrialSearcherState{
		Create: searcher.Create{RequestID: rID},
		Op: searcher.ValidateAfter{
			RequestID: rID,
			Length:    expconf.NewLengthInBatches(10),
		},
		Complete: true,
		Closed:   false,
	}).Error())
	require.NoError(t, system.Ask(self, ackPreemption{taskID: tr.task.ID}).Error())

	// Terminating stage.
	db.On("DeleteTaskSessionByTaskID", string(tr.task.ID)).Return(nil)
	db.On("CompleteTask", mock.Anything, 0, tr.task.ID).Return(nil)
	db.On("UpdateTrialRunIDAndRestarts", 0, 1, 0).Return(nil)
	for _, a := range allocations {
		containerStateChanged := sproto.TaskContainerStateChanged{
			Container: cproto.Container{
				Parent:  actor.Address{},
				ID:      a.Summary().ID,
				State:   cproto.Terminated,
				Devices: []device.Device{},
			},
			ContainerStopped: &sproto.TaskContainerStopped{
				ContainerStopped: aproto.ContainerStopped{
					Failure: nil,
				},
			},
		}
		require.NoError(t, system.Ask(self, containerStateChanged).Error())
	}

	// Later searcher decides it's done.
	db.On("EndTasks", mock.Anything, 0).Return(nil)
	db.On("UpdateTrial", 0, model.CompletedState).Return(nil)
	require.NoError(t, system.Ask(self, TrialSearcherState{
		Create: searcher.Create{RequestID: rID},
		Op: searcher.ValidateAfter{
			RequestID: rID,
			Length:    expconf.NewLengthInBatches(10),
		},
		Complete: true,
		Closed:   true,
	}).Error())

	require.True(t, tr.stopped)
	require.NoError(t, self.AwaitTermination())
}

func TestTrialRestarts(t *testing.T) {
	require.NoError(t, etc.SetRootPath("../static/srv"))
	system := actor.NewSystem("system")

	// mock resource manager.
	rmImpl := mockActor{}
	rm := system.MustActorOf(actor.Addr("rm"), &rmImpl)

	// mock logger.
	loggerImpl := mockActor{}
	logger := system.MustActorOf(actor.Addr("logger"), &loggerImpl)

	// mock db.
	db := &mocks.DB{}

	// instantiate the trial
	rID := model.NewRequestID(rand.Reader)
	tr := newTrial(
		1,
		model.PausedState,
		TrialSearcherState{Create: searcher.Create{RequestID: rID}, Complete: true},
		rm, logger,
		db,
		schemas.WithDefaults(expconf.ExperimentConfig{
			RawMaxRestarts: ptrs.IntPtr(5),
			RawCheckpointStorage: &expconf.CheckpointStorageConfigV0{
				RawSharedFSConfig: &expconf.SharedFSConfig{
					RawHostPath:      ptrs.StringPtr("/tmp"),
					RawContainerPath: ptrs.StringPtr("determined-sharedfs"),
				},
			},
		}).(expconf.ExperimentConfig),
		&model.Checkpoint{},
		&tasks.TaskSpec{
			AgentUserGroup: &model.AgentUserGroup{},
		},
		archive.Archive{},
	)
	self := system.MustActorOf(actor.Addr("trial"), tr)

	// Pre-scheduled stage.
	require.NoError(t, system.Ask(self, model.ActiveState).Error())
	require.NoError(t, system.Ask(self, TrialSearcherState{
		Create: searcher.Create{RequestID: rID},
		Op: searcher.ValidateAfter{
			RequestID: rID,
			Length:    expconf.NewLengthInBatches(10),
		},
		Complete: false,
		Closed:   true,
	}).Error())
	require.NotNil(t, tr.task)
	require.Contains(t, rmImpl.messages, *tr.task)
	for i := 0; i <= tr.config.MaxRestarts(); i++ {
		// Pre-allocated stage.
		cID := cproto.NewID()
		alloc := &mocks.Allocation{}
		alloc.On("Start", mock.Anything, mock.Anything, mock.Anything).Return()
		alloc.On("Summary").Return(sproto.ContainerSummary{
			TaskID: tr.task.ID,
			ID:     cID,
			Agent:  "agent-1",
		})
		alloc.On("Kill", mock.Anything).Return()
		db.On("AddTrial", mock.Anything).Return(nil)
		db.On("UpdateTrialRunID", 0, i+1).Return(nil)
		db.On("AddTask", mock.Anything, 0, tr.task.ID).Return(nil)
		db.On("StartTaskSession", string(tr.task.ID)).Return("", nil)
		db.On("LatestCheckpointForTrial", 0).Return(&model.Checkpoint{}, nil)
		require.NoError(t, system.Ask(rm, forward{
			to: self,
			msg: sproto.ResourcesAllocated{
				ID:           tr.task.ID,
				ResourcePool: "default",
				Allocations:  []sproto.Allocation{alloc},
			},
		}).Error())
		require.NotEmpty(t, tr.allocations)

		// Pre-ready stage.
		containerStateChanged := sproto.TaskContainerStateChanged{
			Container: cproto.Container{
				Parent:  actor.Address{},
				ID:      cID,
				State:   cproto.Assigned,
				Devices: []device.Device{},
			},
		}
		require.NoError(t, system.Ask(self, containerStateChanged).Error())
		containerStateChanged.Container.State = cproto.Pulling
		require.NoError(t, system.Ask(self, containerStateChanged).Error())
		containerStateChanged.Container.State = cproto.Starting
		require.NoError(t, system.Ask(self, containerStateChanged).Error())
		containerStateChanged.Container.State = cproto.Running
		containerStateChanged.ContainerStarted = &sproto.TaskContainerStarted{
			Addresses: []cproto.Address{
				{
					ContainerIP:   "0.0.0.0",
					ContainerPort: 1734,
					HostIP:        "10.0.0.1",
					HostPort:      1734,
				},
			},
		}
		require.NoError(t, system.Ask(self, containerStateChanged).Error())
		containerStateChanged.ContainerStarted = nil
		require.NoError(t, system.Ask(self, watchRendezvousInfo{
			taskID: tr.task.ID,
			id:     cID,
		}).Error())
		require.True(t, tr.rendezvous.ready())

		// Running stage.

		// Terminating stage.
		db.On("DeleteTaskSessionByTaskID", string(tr.task.ID)).Return(nil)
		db.On("CompleteTask", mock.Anything, 0, tr.task.ID).Return(nil)
		db.On("UpdateTrialRestarts", 0, i+1).Return(nil)
		if i == tr.config.MaxRestarts() {
			db.On("EndTasks", mock.Anything, 0).Return(nil)
			db.On("UpdateTrial", 0, model.ErrorState).Return(nil)
		}
		containerStateChanged.Container.State = cproto.Terminated
		containerStateChanged.ContainerStarted = nil
		exitCode := aproto.ExitCode(137)
		containerStateChanged.ContainerStopped = &sproto.TaskContainerStopped{
			ContainerStopped: aproto.ContainerStopped{Failure: &aproto.ContainerFailure{
				FailureType: aproto.ContainerFailed,
				ErrMsg:      "some bad stuff went down",
				ExitCode:    &exitCode,
			}},
		}
		require.NoError(t, system.Ask(self, containerStateChanged).Error())
	}
	require.True(t, tr.stopped)
	require.NoError(t, self.AwaitTermination())
}

func TestRendezvous(t *testing.T) {
	const operations = 4
	type testCase struct {
		name  string
		order []int
	}

	runTestCase := func(t *testing.T, tc testCase) {
		t.Run(tc.name, func(t *testing.T) {
			// "task" with ranks is started.
			t1 := sproto.NewTaskID()
			c1, c2 := cproto.NewID(), cproto.NewID()
			ranks := map[cproto.ID]int{c1: 0, c2: 1}
			r := newRendezvous(t1, ranks)

			assert.Equal(t, r.rank(c1), 0)
			assert.Equal(t, r.rank(c2), 1)

			var ws []rendezvousWatcher
			watch := func(cID cproto.ID) func() {
				return func() {
					w, err := r.watch(t1, cID)
					assert.NilError(t, err, cID)
					ws = append(ws, w)
				}
			}

			startContainer := func(cID cproto.ID) func() {
				return func() {
					r.containerStarted(cID, addressesFromContainerID(cID))
				}
			}

			ops := []func(){
				watch(c1),
				watch(c2),
				startContainer(c1),
				startContainer(c2),
			}
			for _, i := range tc.order {
				assert.Check(t, !r.ready())
				ops[i]()
			}
			assert.Check(t, r.ready())

			rendezvousArrived := func(w rendezvousWatcher) {
				select {
				case resp := <-w.C:
					assert.NilError(t, resp.err)
					assert.Equal(t, len(resp.info.Addresses), 2)
				default:
					t.Fatal("expected rendezvous on first watcher but found none")
				}
			}
			for _, w := range ws {
				rendezvousArrived(w)
			}

			r.unwatch(c1)
			r.unwatch(c2)
		})
	}

	for _, ordering := range orderings(operations) {
		runTestCase(t, testCase{
			name:  spew.Sdump(ordering),
			order: ordering,
		})
	}
}

func TestRendezvousUninitialized(t *testing.T) {
	// Initialize a nil rendezvous
	var r *rendezvous

	// All API-connected methods (so ones a user could call) should not panic the actor.
	tID := sproto.NewTaskID()
	cID := cproto.NewID()
	_, err := r.watch(tID, cID)
	assert.ErrorContains(t, err, "watch rendezvous not valid without active task")
	r.unwatch(cID)
}

func TestRendezvousValidation(t *testing.T) {
	t1 := sproto.NewTaskID()
	c1 := cproto.NewID()
	r := newRendezvous(t1, map[cproto.ID]int{
		c1: 0,
	})

	_, err := r.watch(t1, cproto.NewID())
	assert.ErrorContains(t, err, "stale container")

	_, err = r.watch(t1, c1)
	assert.NilError(t, err)

	_, err = r.watch(t1, c1)
	assert.ErrorContains(t, err, "rendezvous request from already connected container")
}

func TestTerminationInRendezvous(t *testing.T) {
	t1 := sproto.NewTaskID()
	c1, c2 := cproto.NewID(), cproto.NewID()
	ranks := map[cproto.ID]int{c1: 0, c2: 1}
	r := newRendezvous(t1, ranks)

	r.containerStarted(c1, addressesFromContainerID(c1))
	_, err := r.watch(t1, c1)
	assert.NilError(t, err)
	r.containerTerminated(c1)

	r.containerStarted(c2, addressesFromContainerID(c2))
	_, err = r.watch(t1, c2)
	assert.NilError(t, err)

	assert.Check(t, !r.ready())
}

func TestUnwatchInRendezvous(t *testing.T) {
	t1 := sproto.NewTaskID()
	c1, c2 := cproto.NewID(), cproto.NewID()
	ranks := map[cproto.ID]int{c1: 0, c2: 1}
	r := newRendezvous(t1, ranks)

	r.containerStarted(c1, addressesFromContainerID(c1))
	_, err := r.watch(t1, c1)
	assert.NilError(t, err)
	r.unwatch(c1)

	r.containerStarted(c2, addressesFromContainerID(c2))
	_, err = r.watch(t1, c2)
	assert.NilError(t, err)

	assert.Check(t, !r.ready())
}

func TestRendezvousTimeout(t *testing.T) {
	rendezvousTimeoutDuration = 0

	t1 := sproto.NewTaskID()
	c1, c2 := cproto.NewID(), cproto.NewID()
	ranks := map[cproto.ID]int{c1: 0, c2: 1}
	r := newRendezvous(t1, ranks)

	_, err := r.watch(t1, c1)
	assert.NilError(t, err)
	r.containerStarted(c1, addressesFromContainerID(c1))

	time.Sleep(-1)
	assert.ErrorContains(t, r.checkTimeout(t1), "some containers are taking a long time")
}

func addressesFromContainerID(cID cproto.ID) []cproto.Address {
	return []cproto.Address{
		{
			ContainerIP:   "172.0.1.2",
			ContainerPort: 1734,
			HostIP:        fmt.Sprintf("%s.somehost.io", cID),
			HostPort:      1734,
		},
	}
}

func TestPreemption(t *testing.T) {
	// Initialize a nil preemption.
	var p *preemption

	// Watch nil should not panic and return an error.
	id := uuid.New()
	_, err := p.watch(sproto.NewTaskID(), id)
	assert.ErrorContains(t, err, "no preemption status")

	// All method on nil should not panic.
	p.unwatch(id)
	p.preempt()
	p.close()

	// "task" is allocated.
	t1 := sproto.NewTaskID()
	p = newPreemption(t1)

	// real watcher connects
	id = uuid.New()
	w, err := p.watch(t1, id)
	assert.NilError(t, err)

	// should immediately receive initial status.
	select {
	case <-w.C:
		t.Fatal("received preemption but should not have")
	default:
	}

	// on preemption, it should also receive status.
	p.preempt()

	// should receive updated preemption status.
	select {
	case <-w.C:
	default:
		t.Fatal("did not receive preemption")
	}

	// preempted preemption unwatching should work.
	p.unwatch(id)

	// new post-preemption watch connects
	id = uuid.New()
	w, err = p.watch(t1, id)
	assert.NilError(t, err)

	// should immediately receive initial status and initial status should be preemption.
	select {
	case <-w.C:
	default:
		t.Fatal("preemptionWatcher.C was empty channel (should come with initial status when preempted)")
	}

	// preempted preemption unwatching should work.
	p.unwatch(id)
}

// orderings returns all orders for n operations.
func orderings(n int) [][]int {
	var xs []int
	for i := 0; i < n; i++ {
		xs = append(xs, i)
	}
	return permutations(xs)
}

// https://stackoverflow.com/questions/30226438/generate-all-permutations-in-go
func permutations(arr []int) [][]int {
	var helper func([]int, int)
	res := [][]int{}

	helper = func(arr []int, n int) {
		if n == 1 {
			tmp := make([]int, len(arr))
			copy(tmp, arr)
			res = append(res, tmp)
		} else {
			for i := 0; i < n; i++ {
				helper(arr, n-1)
				if n%2 == 1 {
					tmp := arr[i]
					arr[i] = arr[n-1]
					arr[n-1] = tmp
				} else {
					tmp := arr[0]
					arr[0] = arr[n-1]
					arr[n-1] = tmp
				}
			}
		}
	}

	helper(arr, len(arr))
	return res
}
