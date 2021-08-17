package internal

import (
	"crypto/rand"
	"fmt"
	"testing"

	"github.com/determined-ai/determined/master/pkg/actor/actors"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/db"

	"github.com/determined-ai/determined/master/internal/task"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/master/internal/mocks"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/archive"
	"github.com/determined-ai/determined/master/pkg/etc"
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
	require.NotNil(t, tr.allocation)

	// Pre-allocated stage.
	db.On("AddTrial", mock.Anything).Return(nil)
	db.On("UpdateTrialRunID", 0, 0).Return(nil)
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
			Length:    expconf.NewLengthInBatches(10),
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

	for i := 0; i <= tr.config.MaxRestarts(); i++ {
		require.NotNil(t, tr.allocation)
		require.Equal(t, i, tr.restarts)

		// Pre-allocated stage.
		if i == 0 {
			db.On("AddTrial", mock.Anything).Return(nil)
		}
		db.On("UpdateTrialRunID", 0, i).Return(nil)
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
		system.Ask(self, model.TrialLog{}).Get() // sync

		if i == tr.config.MaxRestarts() {
			require.True(t, db.AssertExpectations(t))
		}
	}
	require.NoError(t, self.AwaitTermination())
	require.True(t, model.TerminalStates[tr.state])
}

func setup(t *testing.T) (*actor.System, *mocks.DB, model.RequestID, *trial, *actor.Ref) {
	require.NoError(t, etc.SetRootPath("../static/srv"))
	system := actor.NewSystem("system")

	// mock resource manager.
	rmImpl := actors.MockActor{Responses: map[string]*actors.MockResponse{}}
	rm := system.MustActorOf(actor.Addr("rm"), &rmImpl)

	// mock allocation
	allocImpl := actors.MockActor{Responses: map[string]*actors.MockResponse{}}
	taskAllocator = func(req sproto.AllocateRequest, db db.DB, rm *actor.Ref) actor.Actor {
		return &allocImpl
	}

	// mock logger.
	loggerImpl := actors.MockActor{Responses: map[string]*actors.MockResponse{}}
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
	return system, db, rID, tr, self
}
