//nolint:exhaustivestruct
package internal

import (
	"crypto/rand"
	"fmt"
	"testing"
	"time"

	"github.com/determined-ai/determined/master/internal/rm/actorrm"

	"github.com/determined-ai/determined/master/pkg/actor/actors"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/rm"

	"github.com/determined-ai/determined/master/internal/task"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/master/internal/mocks"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/etc"
	detLogger "github.com/determined-ai/determined/master/pkg/logger"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/master/pkg/searcher"
	"github.com/determined-ai/determined/master/pkg/tasks"
)

func TestTrial(t *testing.T) {
	system, db, rID, tr, self := setup(t)

	// Pre-scheduled stage.
	require.NoError(t, system.Ask(self,
		model.StateWithReason{State: model.ActiveState}).Error())
	require.NoError(t, system.Ask(self, trialSearcherState{
		Create: searcher.Create{RequestID: rID},
		Op: searcher.ValidateAfter{
			RequestID: rID,
			Length:    10,
		},
		Complete: false,
		Closed:   true,
	}).Error())
	require.NotNil(t, tr.allocation)

	// Pre-allocated stage.
	db.On("AddTrial", mock.Anything).Return(nil)
	db.On("UpdateTrialRunID", 0, 1).Return(nil)
	db.On("LatestCheckpointForTrial", 0).Return(&model.Checkpoint{}, nil)
	require.NoError(t, system.Ask(tr.allocation, actors.ForwardThroughMock{
		To:  self,
		Msg: task.BuildTaskSpec{},
	}).Error())
	require.True(t, db.AssertExpectations(t))

	// Running stage.
	db.On("UpdateTrial", 0, model.StoppingCompletedState).Return(nil)
	require.NoError(t, system.Ask(self, trialSearcherState{
		Create: searcher.Create{RequestID: rID},
		Op: searcher.ValidateAfter{
			RequestID: rID,
			Length:    10,
		},
		Complete: true,
		Closed:   true,
	}).Error())
	require.True(t, db.AssertExpectations(t))

	// Terminating stage.
	db.On("UpdateTrial", 0, model.CompletedState).Return(nil)
	system.Tell(tr.allocation, actors.ForwardThroughMock{
		To:  self,
		Msg: &task.AllocationExited{},
	})
	require.NoError(t, tr.allocation.StopAndAwaitTermination())
	require.NoError(t, self.AwaitTermination())
	require.True(t, model.TerminalStates[tr.state])
	require.True(t, db.AssertExpectations(t))
}

func TestTrialRestarts(t *testing.T) {
	system, db, rID, tr, self := setup(t)

	// Pre-scheduled stage.
	require.NoError(t, system.Ask(self,
		model.StateWithReason{State: model.ActiveState}).Error())
	require.NoError(t, system.Ask(self, trialSearcherState{
		Create: searcher.Create{RequestID: rID},
		Op: searcher.ValidateAfter{
			RequestID: rID,
			Length:    10,
		},
		Complete: false,
		Closed:   true,
	}).Error())

	for i := 0; i <= tr.config.MaxRestarts(); i++ {
		require.NotNil(t, tr.allocation)
		require.Equal(t, i, tr.restarts)

		// Pre-allocated stage.
		if i == 0 {
			db.On("AddTrial", mock.Anything).Return(nil)
		}
		db.On("UpdateTrialRunID", 0, i+1).Return(nil)
		db.On("LatestCheckpointForTrial", 0).Return(&model.Checkpoint{}, nil)
		require.NoError(t, system.Ask(tr.allocation, actors.ForwardThroughMock{
			To:  self,
			Msg: task.BuildTaskSpec{},
		}).Error())
		require.True(t, db.AssertExpectations(t))

		db.On("UpdateTrialRestarts", 0, i+1).Return(nil)
		if i == tr.config.MaxRestarts() {
			db.On("UpdateTrial", 0, model.ErrorState).Return(nil)
		}

		system.Tell(tr.allocation, actors.ForwardThroughMock{
			To:  self,
			Msg: &task.AllocationExited{Err: errors.New("bad stuff went down")},
		})
		require.NoError(t, tr.allocation.StopAndAwaitTermination())
		system.Ask(self, actor.Ping{}).Get() // sync

		if i == tr.config.MaxRestarts() {
			require.True(t, db.AssertExpectations(t))
		}
	}
	require.NoError(t, self.AwaitTermination())
	require.True(t, model.TerminalStates[tr.state])
}

func TestTrialSimultaneousCancelAndAllocation(t *testing.T) {
	system, db, rID, tr, self := setup(t)

	// Pre-scheduled stage.
	require.NoError(t, system.Ask(self,
		model.StateWithReason{State: model.ActiveState}).Error())
	require.NoError(t, system.Ask(self, trialSearcherState{
		Create: searcher.Create{RequestID: rID},
		Op: searcher.ValidateAfter{
			RequestID: rID,
			Length:    10,
		},
		Complete: false,
		Closed:   true,
	}).Error())
	require.NotNil(t, tr.allocation)

	// Send the trial a termination, but don't setup our mock allocation to handle it, as if it
	// is busy handling receiving resources.
	require.NoError(t, system.Ask(self, model.StateWithReason{
		State: model.StoppingCanceledState,
	}).Error())

	// Now the allocation checks in to get what to launch while we're canceled.
	require.Error(t, system.Ask(tr.allocation, actors.ForwardThroughMock{
		To:  self,
		Msg: task.BuildTaskSpec{},
	}).Error())
	require.True(t, db.AssertNotCalled(t, "AddTrial", mock.Anything),
		"trial should not save itself when canceled before ready")
	require.True(t, db.AssertExpectations(t))

	// After the allocation exits, we should error.
	system.Tell(tr.allocation, actors.ForwardThroughMock{
		To:  self,
		Msg: &task.AllocationExited{},
	})
	require.NoError(t, tr.allocation.StopAndAwaitTermination())
	require.NoError(t, self.AwaitTermination())
	require.True(t, db.AssertNotCalled(t, "UpdateTrial", mock.Anything),
		"trial was not saved so no update should happen")
	require.True(t, db.AssertExpectations(t))

	// But the actor itself should have the state recorded.
	require.True(t, model.TerminalStates[tr.state])
}

func setup(t *testing.T) (*actor.System, *mocks.DB, model.RequestID, *trial, *actor.Ref) {
	require.NoError(t, etc.SetRootPath("../static/srv"))
	system := actor.NewSystem("system")

	// mock resource manager.
	rmActor := actors.MockActor{Responses: map[string]*actors.MockResponse{}}
	rmImpl := actorrm.Wrap(system.MustActorOf(actor.Addr("rm"), &rmActor))

	// mock logger.
	loggerImpl := actors.MockActor{Responses: map[string]*actors.MockResponse{}}
	loggerActor := system.MustActorOf(actor.Addr("logger"), &loggerImpl)
	logger := task.NewCustomLogger(loggerActor)

	// mock allocation
	allocImpl := actors.MockActor{Responses: map[string]*actors.MockResponse{}}
	taskAllocator = func(
		logCtx detLogger.Context, req sproto.AllocateRequest, db db.DB, rm rm.ResourceManager,
		l *task.Logger,
	) actor.Actor {
		return &allocImpl
	}

	// mock db.
	db := &mocks.DB{}
	db.On("AddTask", mock.Anything).Return(nil)

	// instantiate the trial
	rID := model.NewRequestID(rand.Reader)
	taskID := model.TaskID(fmt.Sprintf("%s-%s", model.TaskTypeTrial, rID))
	tr := newTrial(
		detLogger.Context{},
		taskID,
		model.JobID("1"),
		time.Now(),
		1,
		model.PausedState,
		trialSearcherState{Create: searcher.Create{RequestID: rID}, Complete: true},
		logger,
		rmImpl,
		db,
		schemas.WithDefaults(expconf.ExperimentConfig{
			RawCheckpointStorage: &expconf.CheckpointStorageConfigV0{
				RawSharedFSConfig: &expconf.SharedFSConfig{
					RawHostPath:      ptrs.Ptr("/tmp"),
					RawContainerPath: ptrs.Ptr("determined-sharedfs"),
				},
			},
		}),
		&model.Checkpoint{},
		&tasks.TaskSpec{
			AgentUserGroup: &model.AgentUserGroup{},
			SSHRsaSize:     1024,
		},
		false,
	)
	self := system.MustActorOf(actor.Addr("trial"), tr)
	return system, db, rID, tr, self
}
