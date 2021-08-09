package internal

import (
	"crypto/rand"
	"fmt"
	"testing"

	"github.com/determined-ai/determined/master/internal/task"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

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

	cproto "github.com/determined-ai/determined/master/pkg/container"
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
	taskID := model.TaskID(fmt.Sprintf("%s-%s", model.TaskTypeTrial, rID))
	tr := newTrial(
		taskID,
		1,
		model.PausedState,
		trialSearcherState{Create: searcher.Create{RequestID: rID}, Complete: true},
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
	require.NoError(t, system.Ask(self, trialSearcherState{
		Create: searcher.Create{RequestID: rID},
		Op: searcher.ValidateAfter{
			RequestID: rID,
			Length:    expconf.NewLengthInBatches(10),
		},
		Complete: false,
		Closed:   true,
	}).Error())
	require.NotNil(t, tr.req)
	require.Contains(t, rmImpl.messages, *tr.req)

	// Pre-allocated stage.
	mockRsvn := func(cID cproto.ID, agentID string) sproto.Reservation {
		rsrv := &mocks.Reservation{}
		rsrv.On("Start", mock.Anything, mock.Anything, mock.Anything).Return().Times(1)
		rsrv.On("Summary").Return(sproto.ContainerSummary{
			AllocationID: tr.req.AllocationID,
			ID:           cID,
			Agent:        agentID,
		})
		rsrv.On("Kill", mock.Anything).Return()
		return rsrv
	}

	reservations := []sproto.Reservation{
		mockRsvn(cproto.NewID(), "agent-1"),
		mockRsvn(cproto.NewID(), "agent-2"),
	}
	db.On("AddTrial", mock.Anything).Return(nil)
	db.On("UpdateTrialRunID", 0, 1).Return(nil)
	db.On("AddAllocation", mock.Anything).Return(nil)
	db.On("StartAllocationSession", tr.req.AllocationID).Return("", nil)
	db.On("LatestCheckpointForTrial", 0).Return(&model.Checkpoint{}, nil)
	require.NoError(t, system.Ask(rm, forward{
		to: self,
		msg: sproto.ResourcesAllocated{
			ID:           tr.req.AllocationID,
			ResourcePool: "default",
			Reservations: reservations,
		},
	}).Error())
	require.NotNil(t, tr.allocation)
	require.True(t, db.AssertExpectations(t))

	// Pre-ready stage.
	for _, a := range reservations {
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
		require.NoError(t, system.Ask(self, task.WatchRendezvousInfo{
			AllocationID: tr.req.AllocationID,
			ContainerID:  a.Summary().ID,
		}).Error())
	}

	// Running stage.
	db.On("UpdateTrial", 0, model.StoppingCompletedState).Return(nil)
	require.NoError(t, system.Ask(self, trialSearcherState{
		Create: searcher.Create{RequestID: rID},
		Op: searcher.ValidateAfter{
			RequestID: rID,
			Length:    expconf.NewLengthInBatches(10),
		},
		Complete: true,
		Closed:   true,
	}).Error())
	require.True(t, db.AssertExpectations(t))

	// Terminating stage.
	db.On("DeleteAllocationSession", tr.req.AllocationID).Return(nil)
	db.On("CompleteAllocation", mock.Anything).Return(nil)
	db.On("UpdateTrial", 0, model.CompletedState).Return(nil)
	for _, a := range reservations {
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
		system.Ask(self, model.TrialLog{}).Get() // Just to sync
	}

	require.True(t, model.TerminalStates[tr.state])
	require.NoError(t, self.AwaitTermination())
	require.True(t, db.AssertExpectations(t))
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
	taskID := model.TaskID(fmt.Sprintf("%s-%s", model.TaskTypeTrial, rID))
	tr := newTrial(
		taskID,
		1,
		model.PausedState,
		trialSearcherState{Create: searcher.Create{RequestID: rID}, Complete: true},
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
	require.NoError(t, system.Ask(self, trialSearcherState{
		Create: searcher.Create{RequestID: rID},
		Op: searcher.ValidateAfter{
			RequestID: rID,
			Length:    expconf.NewLengthInBatches(10),
		},
		Complete: false,
		Closed:   true,
	}).Error())
	require.NotNil(t, tr.req)
	require.Contains(t, rmImpl.messages, *tr.req)

	// Pre-allocated stage.
	mockRsrv := func(cID cproto.ID, agentID string) sproto.Reservation {
		rsrv := &mocks.Reservation{}
		rsrv.On("Start", mock.Anything, mock.Anything, mock.Anything).Return()
		rsrv.On("Summary").Return(sproto.ContainerSummary{
			AllocationID: tr.req.AllocationID,
			ID:           cID,
			Agent:        agentID,
		})
		rsrv.On("Kill", mock.Anything).Return()
		return rsrv
	}

	reservations := []sproto.Reservation{
		mockRsrv(cproto.NewID(), "agent-1"),
		mockRsrv(cproto.NewID(), "agent-2"),
	}
	db.On("AddTrial", mock.Anything).Return(nil)
	db.On("UpdateTrialRunID", 0, 1).Return(nil)
	db.On("AddAllocation", mock.Anything).Return(nil)
	db.On("StartAllocationSession", tr.req.AllocationID).Return("", nil)
	db.On("LatestCheckpointForTrial", 0).Return(&model.Checkpoint{}, nil)
	require.NoError(t, system.Ask(rm, forward{
		to: self,
		msg: sproto.ResourcesAllocated{
			ID:           tr.req.AllocationID,
			ResourcePool: "default",
			Reservations: reservations,
		},
	}).Error())
	require.NotNil(t, tr.allocation)
	require.True(t, db.AssertExpectations(t))

	// Pre-ready stage.
	for _, a := range reservations {
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
		require.NoError(t, system.Ask(self, task.WatchRendezvousInfo{
			AllocationID: tr.req.AllocationID,
			ContainerID:  a.Summary().ID,
		}).Error())
	}

	// Running stage.
	require.NoError(t, system.Ask(self, trialSearcherState{
		Create: searcher.Create{RequestID: rID},
		Op: searcher.ValidateAfter{
			RequestID: rID,
			Length:    expconf.NewLengthInBatches(10),
		},
		Complete: true,
		Closed:   false,
	}).Error())
	require.NoError(t, system.Ask(self, task.AckPreemption{AllocationID: tr.req.AllocationID}).Error())

	// Terminating stage.
	db.On("DeleteAllocationSession", tr.req.AllocationID).Return(nil)
	db.On("CompleteAllocation", mock.Anything).Return(nil)
	db.On("UpdateTrialRunID", 0, 1).Return(nil)
	for _, a := range reservations {
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
	system.Ask(self, model.TrialLog{}).Get() // Just to sync
	require.True(t, db.AssertExpectations(t))

	// Later searcher decides it's done.
	db.On("UpdateTrial", 0, model.StoppingCompletedState).Return(nil)
	db.On("UpdateTrial", 0, model.CompletedState).Return(nil)
	require.NoError(t, system.Ask(self, trialSearcherState{
		Create: searcher.Create{RequestID: rID},
		Op: searcher.ValidateAfter{
			RequestID: rID,
			Length:    expconf.NewLengthInBatches(10),
		},
		Complete: true,
		Closed:   true,
	}).Error())

	require.True(t, model.TerminalStates[tr.state])
	require.NoError(t, self.AwaitTermination())
	require.True(t, db.AssertExpectations(t))
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
	taskID := model.TaskID(fmt.Sprintf("%s-%s", model.TaskTypeTrial, rID))
	tr := newTrial(
		taskID,
		1,
		model.PausedState,
		trialSearcherState{Create: searcher.Create{RequestID: rID}, Complete: true},
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
	require.NoError(t, system.Ask(self, trialSearcherState{
		Create: searcher.Create{RequestID: rID},
		Op: searcher.ValidateAfter{
			RequestID: rID,
			Length:    expconf.NewLengthInBatches(10),
		},
		Complete: false,
		Closed:   true,
	}).Error())
	require.NotNil(t, tr.req)
	require.Contains(t, rmImpl.messages, *tr.req)
	for i := 0; i <= tr.config.MaxRestarts(); i++ {
		// Pre-allocated stage.
		cID := cproto.NewID()
		rsrv := &mocks.Reservation{}
		rsrv.On("Start", mock.Anything, mock.Anything, mock.Anything).Return()
		rsrv.On("Summary").Return(sproto.ContainerSummary{
			AllocationID: tr.req.AllocationID,
			ID:           cID,
			Agent:        "agent-1",
		})
		rsrv.On("Kill", mock.Anything).Return()
		db.On("AddTrial", mock.Anything).Return(nil)
		db.On("UpdateTrialRunID", 0, i+1).Return(nil)
		db.On("AddAllocation", mock.Anything).Return(nil)
		db.On("StartAllocationSession", tr.req.AllocationID).Return("", nil)
		db.On("LatestCheckpointForTrial", 0).Return(&model.Checkpoint{}, nil)
		require.NoError(t, system.Ask(rm, forward{
			to: self,
			msg: sproto.ResourcesAllocated{
				ID:           tr.req.AllocationID,
				ResourcePool: "default",
				Reservations: []sproto.Reservation{rsrv},
			},
		}).Error())
		require.NotNil(t, tr.allocation)
		require.True(t, db.AssertExpectations(t))

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
		require.NoError(t, system.Ask(self, task.WatchRendezvousInfo{
			AllocationID: tr.req.AllocationID,
			ContainerID:  cID,
		}).Error())

		// Running stage.

		// Terminating stage.
		db.On("DeleteAllocationSession", tr.req.AllocationID).Return(nil)
		db.On("CompleteAllocation", mock.Anything).Return(nil)
		db.On("UpdateTrialRestarts", 0, i+1).Return(nil)
		if i == tr.config.MaxRestarts() {
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
		system.Ask(self, model.TrialLog{}).Get() // Just to sync.
		require.True(t, db.AssertExpectations(t))
	}
	require.True(t, model.TerminalStates[tr.state])
	require.NoError(t, self.AwaitTermination())
}
