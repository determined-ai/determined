//go:build integration
// +build integration

//nolint:exhaustruct
package internal

import (
	"context"
	"crypto/rand"
	"fmt"
	"testing"
	"time"

	"github.com/determined-ai/determined/master/pkg/ptrs"

	"github.com/determined-ai/determined/master/pkg/schemas"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	internaldb "github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/experiment"
	"github.com/determined-ai/determined/master/internal/mocks/allocationmocks"
	"github.com/determined-ai/determined/master/internal/task"
	"github.com/determined-ai/determined/master/pkg/etc"
	detLogger "github.com/determined-ai/determined/master/pkg/logger"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/searcher"
	"github.com/determined-ai/determined/master/pkg/ssh"
	"github.com/determined-ai/determined/master/pkg/tasks"
)

func TestTrial(t *testing.T) {
	_, rID, tr, alloc, done := setup(t)
	// xxx: fix this test
	// Pre-scheduled stage.
	require.NoError(t, tr.PatchState(
		model.StateWithReason{State: model.ActiveState}))
	require.NoError(t, tr.PatchSearcherState(experiment.TrialSearcherState{
		Create: searcher.Create{RequestID: rID},
		Op: searcher.ValidateAfter{
			RequestID: rID,
			Length:    10,
		},
		Complete: false,
		Closed:   true,
	}))

	// Running stage.
	require.NoError(t, tr.PatchSearcherState(experiment.TrialSearcherState{
		Create:   searcher.Create{RequestID: rID},
		Complete: true,
		Closed:   true,
	}))
	require.True(t, alloc.AssertExpectations(t))
	require.NotNil(t, tr.allocationID)

	dbTrial, err := internaldb.TrialByID(context.TODO(), tr.id)
	require.NoError(t, err)
	require.Equal(t, model.StoppingCompletedState, dbTrial.State)

	// Terminating stage.
	tr.AllocationExitedCallback(&task.AllocationExited{})
	select {
	case <-done: // success
	case <-time.After(5 * time.Second):
		require.Error(t, fmt.Errorf("timed out waiting for trial to terminate"))
	}
	require.True(t, model.TerminalStates[tr.state])

	dbTrial, err = internaldb.TrialByID(context.TODO(), tr.id)
	require.NoError(t, err)
	require.Equal(t, model.CompletedState, dbTrial.State)
}

func TestTrialRestarts(t *testing.T) {
	pgDB, rID, tr, _, done := setup(t)
	// Pre-scheduled stage.
	require.NoError(t, tr.PatchState(
		model.StateWithReason{State: model.ActiveState}))
	require.NoError(t, tr.PatchSearcherState(experiment.TrialSearcherState{
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

		tr.AllocationExitedCallback(&task.AllocationExited{Err: fmt.Errorf("bad stuff went down")})

		if i == tr.config.MaxRestarts() {
			dbTrial, err := internaldb.TrialByID(context.TODO(), tr.id)
			require.NoError(t, err)
			require.Equal(t, model.ErrorState, dbTrial.State)
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
		require.Error(t, fmt.Errorf("timed out waiting for trial to terminate"))
	}
	require.True(t, model.TerminalStates[tr.state])
}

func setup(t *testing.T) (
	*internaldb.PgDB,
	model.RequestID,
	*trial,
	*allocationmocks.AllocationService,
	chan bool,
) {
	require.NoError(t, etc.SetRootPath("../static/srv"))

	// mock resource manager.
	rmImpl := MockRM()

	// mock allocation service
	var as allocationmocks.AllocationService
	task.DefaultService = &as
	as.On(
		"StartAllocation", mock.Anything, mock.Anything, mock.Anything,
		mock.Anything, mock.Anything, mock.Anything, mock.Anything,
	).Return(nil)
	as.On("Signal", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	a, _, _ := setupAPITest(t, nil)
	j := &model.Job{JobID: model.NewJobID(), JobType: model.JobTypeExperiment}
	require.NoError(t, internaldb.AddJob(j))

	// instantiate the trial
	rID := model.NewRequestID(rand.Reader)
	taskID := model.TaskID(fmt.Sprintf("%s-%s", model.TaskTypeTrial, rID))
	done := make(chan bool)

	// create expconf merged with task container defaults
	expConf := schemas.WithDefaults(expconf.ExperimentConfig{
		RawCheckpointStorage: &expconf.CheckpointStorageConfigV0{
			RawSharedFSConfig: &expconf.SharedFSConfig{
				RawHostPath:      ptrs.Ptr("/tmp"),
				RawContainerPath: ptrs.Ptr("determined-sharedfs"),
			},
		},
	})
	model.DefaultTaskContainerDefaults().MergeIntoExpConfig(&expConf)
	tr, err := newTrial(
		detLogger.Context{},
		taskID,
		j.JobID,
		time.Now(),
		1,
		model.PausedState,
		experiment.TrialSearcherState{Create: searcher.Create{RequestID: rID}, Complete: true},
		rmImpl,
		a.m.db,
		expConf,
		&model.Checkpoint{},
		&tasks.TaskSpec{
			AgentUserGroup: &model.AgentUserGroup{},
			SSHRsaSize:     1024,
			Workspace:      model.DefaultWorkspaceName,
		},
		ssh.PrivateAndPublicKeys{},
		false,
		nil, nil, func(ri model.RequestID, reason *model.ExitedReason) {
			require.Equal(t, rID, ri)
			done <- true
			close(done)
		},
	)
	require.NoError(t, err)
	return a.m.db, rID, tr, &as, done
}
