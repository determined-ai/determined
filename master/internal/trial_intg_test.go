//go:build integration
// +build integration

//nolint:exhaustivestruct
package internal

import (
	"context"
	"crypto/rand"
	"fmt"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/mocks/allocationmocks"
	"github.com/determined-ai/determined/master/internal/rm/actorrm"
	"github.com/determined-ai/determined/master/internal/task"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/actor/actors"
	"github.com/determined-ai/determined/master/pkg/etc"
	detLogger "github.com/determined-ai/determined/master/pkg/logger"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/master/pkg/searcher"
	"github.com/determined-ai/determined/master/pkg/ssh"
	"github.com/determined-ai/determined/master/pkg/tasks"
)

func TestTrial(t *testing.T) {
	_, _, rID, tr, alloc, done := setup(t)

	// Pre-scheduled stage.
	require.NoError(t, tr.PatchState(
		model.StateWithReason{State: model.ActiveState}))
	require.NoError(t, tr.PatchSearcherState(trialSearcherState{
		Create: searcher.Create{RequestID: rID},
		Op: searcher.ValidateAfter{
			RequestID: rID,
			Length:    10,
		},
		Complete: false,
		Closed:   true,
	}))
	require.True(t, alloc.AssertExpectations(t))
	require.NotNil(t, tr.allocationID)

	// Running stage.
	require.NoError(t, tr.PatchSearcherState(trialSearcherState{
		Create: searcher.Create{RequestID: rID},
		Op: searcher.ValidateAfter{
			RequestID: rID,
			Length:    10,
		},
		Complete: true,
		Closed:   true,
	}))

	dbTrial, err := db.TrialByID(context.TODO(), tr.id)
	require.NoError(t, err)
	require.Equal(t, dbTrial.State, model.StoppingCompletedState)

	// Terminating stage.
	tr.AllocationExitedCallback(&task.AllocationExited{})
	select {
	case <-done: // success
	case <-time.After(5 * time.Second):
		require.Error(t, errors.New("timed out waiting for trial to terminate"))
	}
	require.True(t, model.TerminalStates[tr.state])

	dbTrial, err = db.TrialByID(context.TODO(), tr.id)
	require.NoError(t, err)
	require.Equal(t, dbTrial.State, model.CompletedState)
}

func TestTrialRestarts(t *testing.T) {
	_, pgDB, rID, tr, _, done := setup(t)
	// Pre-scheduled stage.
	require.NoError(t, tr.PatchState(
		model.StateWithReason{State: model.ActiveState}))
	require.NoError(t, tr.PatchSearcherState(trialSearcherState{
		Create: searcher.Create{RequestID: rID},
		Op: searcher.ValidateAfter{
			RequestID: rID,
			Length:    10,
		},
		Complete: false,
		Closed:   true,
	}))

	for i := 0; i <= tr.config.MaxRestarts(); i++ {
		require.NotNil(t, tr.allocationID)
		require.Equal(t, i, tr.restarts)

		tr.AllocationExitedCallback(&task.AllocationExited{Err: errors.New("bad stuff went down")})

		if i == tr.config.MaxRestarts() {
			dbTrial, err := db.TrialByID(context.TODO(), tr.id)
			require.NoError(t, err)
			require.Equal(t, dbTrial.State, model.ErrorState)
		} else {
			// For the next go-around, when we update trial run ID.
			runID, _, err := pgDB.TrialRunIDAndRestarts(tr.id)
			require.NoError(t, err)
			require.Equal(t, i+2, runID)
		}
	}
	select {
	case <-done: // success
	case <-time.After(5 * time.Second):
		require.Error(t, errors.New("timed out waiting for trial to terminate"))
	}
	require.True(t, model.TerminalStates[tr.state])
}

func setup(t *testing.T) (
	*actor.System,
	*db.PgDB,
	model.RequestID,
	*trial,
	*allocationmocks.AllocationService,
	chan bool,
) {
	require.NoError(t, etc.SetRootPath("../static/srv"))
	system := actor.NewSystem("system")

	// mock resource manager.
	rmActor := actors.MockActor{Responses: map[string]*actors.MockResponse{}}
	rmImpl := actorrm.Wrap(system.MustActorOf(actor.Addr("rm"), &rmActor))

	expActor := actors.MockActor{Responses: map[string]*actors.MockResponse{}}
	expRef := system.MustActorOf(actor.Addr("experiment"), &expActor)

	// mock allocation service
	var as allocationmocks.AllocationService
	task.DefaultService = &as
	as.On(
		"StartAllocation", mock.Anything, mock.Anything, mock.Anything,
		mock.Anything, mock.Anything, mock.Anything, mock.Anything,
	).Return(nil)

	a, _, _ := setupAPITest(t, nil)
	j := &model.Job{JobID: model.NewJobID(), JobType: model.JobTypeExperiment}
	require.NoError(t, a.m.db.AddJob(j))

	// instantiate the trial
	rID := model.NewRequestID(rand.Reader)
	taskID := model.TaskID(fmt.Sprintf("%s-%s", model.TaskTypeTrial, rID))
	done := make(chan bool)
	tr, err := newTrial(
		detLogger.Context{},
		taskID,
		j.JobID,
		time.Now(),
		1,
		model.PausedState,
		trialSearcherState{Create: searcher.Create{RequestID: rID}, Complete: true},
		rmImpl,
		a.m.db,
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
		ssh.PrivateAndPublicKeys{},
		false,
		nil, system, expRef, func(ri model.RequestID, reason *model.ExitedReason) {
			require.Equal(t, rID, ri)
			done <- true
			close(done)
		},
	)
	require.NoError(t, err)
	return system, a.m.db, rID, tr, &as, done
}
