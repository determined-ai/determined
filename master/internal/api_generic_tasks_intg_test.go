//go:build integration
// +build integration

package internal

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	authz2 "github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

// requireMockGenericTask creates a generic task (with a backing command_state row so that
// command.IdentifyTask can resolve its workspace) and returns its task ID.
func requireMockGenericTask(
	ctx context.Context, t *testing.T, userID model.UserID,
) model.TaskID {
	pgDB := db.SingleDB()

	jID := db.RequireMockJob(t, pgDB, &userID)
	taskID := model.NewTaskID()
	state := model.TaskStateActive
	require.NoError(t, db.AddTask(ctx, &model.Task{
		TaskID:    taskID,
		JobID:     &jID,
		TaskType:  model.TaskTypeGeneric,
		StartTime: time.Now().UTC().Truncate(time.Millisecond),
		State:     &state,
	}))

	alloc := db.RequireMockAllocation(t, pgDB, taskID)

	commandState := struct {
		bun.BaseModel `bun:"table:command_state"`

		TaskID             model.TaskID
		AllocationID       model.AllocationID
		GenericCommandSpec map[string]any
	}{
		TaskID:       taskID,
		AllocationID: alloc.AllocationID,
		GenericCommandSpec: map[string]any{
			"TaskType": model.TaskTypeGeneric,
			"Metadata": map[string]any{
				"workspace_id": 1,
			},
			"Base": map[string]any{
				"Owner": map[string]any{
					"id": userID,
				},
			},
		},
	}
	_, err := db.Bun().NewInsert().Model(&commandState).Exec(ctx)
	require.NoError(t, err)

	return taskID
}

// TestGenericTaskLifecycleAuthZ verifies that KillGenericTask, PauseGenericTask, and
// UnpauseGenericTask enforce authorization before acting on a task. A user that is not
// permitted to view the task's workspace must not be able to kill, pause, or unpause it.
// Regression test for the missing-authorization vulnerability where any authenticated
// user could disrupt another user's generic tasks by supplying the task ID.
func TestGenericTaskLifecycleAuthZ(t *testing.T) {
	api, authZNSC, curUser, ctx := setupNTSCAuthzTest(t)
	taskID := requireMockGenericTask(ctx, t, curUser.ID)

	// Deny the caller access to the task's workspace.
	authZNSC.On("CanGetNSC", mock.Anything, curUser, mock.Anything).
		Return(authz2.PermissionDeniedError{})

	cases := []struct {
		name string
		call func() error
	}{
		{
			name: "kill",
			call: func() error {
				_, err := api.KillGenericTask(ctx, &apiv1.KillGenericTaskRequest{
					TaskId: string(taskID),
				})
				return err
			},
		},
		{
			name: "pause",
			call: func() error {
				_, err := api.PauseGenericTask(ctx, &apiv1.PauseGenericTaskRequest{
					TaskId: string(taskID),
				})
				return err
			},
		},
		{
			name: "unpause",
			call: func() error {
				_, err := api.UnpauseGenericTask(ctx, &apiv1.UnpauseGenericTaskRequest{
					TaskId: string(taskID),
				})
				return err
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.call()
			require.Error(t, err, "expected an authorization error")
			// Permission-denied is surfaced as NotFound so the endpoint does not leak
			// the existence of tasks the caller cannot access.
			require.Equal(t, codes.NotFound, status.Code(err))
		})
	}

	authZNSC.AssertExpectations(t)
}
