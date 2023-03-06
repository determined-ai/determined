//go:build integration
// +build integration

package internal

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/checkpointv1"
)

func createVersionOneCheckpoint(
	ctx context.Context, t *testing.T, api *apiServer, curUser model.User,
) string {
	trial := createTestTrial(t, api, curUser)
	checkpointBun := struct {
		bun.BaseModel `bun:"table:checkpoints"`
		TrialID       int
		TrialRunID    int
		TotalBatches  int
		State         model.State
		UUID          string
		EndTime       time.Time
	}{
		TrialID:      trial.ID,
		TrialRunID:   1,
		TotalBatches: 1,
		State:        model.ActiveState,
		UUID:         uuid.New().String(),
		EndTime:      time.Now().UTC().Truncate(time.Millisecond),
	}

	_, err := db.Bun().NewInsert().Model(&checkpointBun).Exec(ctx)
	require.NoError(t, err)

	return checkpointBun.UUID
}

func createVersionTwoCheckpoint(
	ctx context.Context, t *testing.T, api *apiServer, curUser model.User,
) string {
	trial := createTestTrial(t, api, curUser)

	aID := model.AllocationID(string(trial.TaskID) + "-1")
	aIn := &model.Allocation{
		AllocationID: aID,
		TaskID:       trial.TaskID,
		Slots:        1,
		ResourcePool: "somethingelse",
		StartTime:    ptrs.Ptr(time.Now().UTC().Truncate(time.Millisecond)),
	}
	require.NoError(t, api.m.db.AddAllocation(aIn))

	checkpoint := &model.CheckpointV2{
		ID:           0,
		UUID:         uuid.New(),
		TaskID:       trial.TaskID,
		AllocationID: aID,
		ReportTime:   time.Now(),
		State:        model.ActiveState,
		Resources:    nil,
		Metadata: map[string]interface{}{
			"framework":          "tensortorch",
			"determined_version": "1.0.0",
			"steps_completed":    5,
		},
	}
	require.NoError(t, api.m.db.AddCheckpointMetadata(ctx, checkpoint))

	return checkpoint.UUID.String()
}

func TestCheckpointAuthZ(t *testing.T) {
	api, authZExp, _, curUser, ctx := setupExpAuthTest(t, nil)

	cases := []struct {
		DenyFuncName            string
		IDToReqCall             func(id string) error
		UseMultiCheckpointError bool
	}{
		{"CanGetExperimentArtifacts", func(id string) error {
			_, err := api.GetCheckpoint(ctx, &apiv1.GetCheckpointRequest{
				CheckpointUuid: id,
			})
			return err
		}, false},
		{"CanEditExperiment", func(id string) error {
			_, err := api.DeleteCheckpoints(ctx, &apiv1.DeleteCheckpointsRequest{
				CheckpointUuids: []string{id},
			})
			return err
		}, true},
		{"CanEditExperiment", func(id string) error {
			_, err := api.PostCheckpointMetadata(ctx, &apiv1.PostCheckpointMetadataRequest{
				Checkpoint: &checkpointv1.Checkpoint{Uuid: id},
			})
			return err
		}, false},
	}

	for _, checkpointID := range []string{
		createVersionOneCheckpoint(ctx, t, api, curUser),
		createVersionTwoCheckpoint(ctx, t, api, curUser),
	} {
		for _, curCase := range cases {
			notFoundUUID := uuid.New().String()
			if curCase.UseMultiCheckpointError {
				require.Equal(t, errCheckpointsNotFound([]string{notFoundUUID}),
					curCase.IDToReqCall(notFoundUUID))
			} else {
				require.Equal(t, errCheckpointNotFound(notFoundUUID),
					curCase.IDToReqCall(notFoundUUID))
			}

			authZExp.On("CanGetExperiment", mock.Anything, curUser, mock.Anything).
				Return(false, nil).Once()
			if curCase.UseMultiCheckpointError {
				require.Equal(t, errCheckpointsNotFound([]string{checkpointID}),
					curCase.IDToReqCall(checkpointID))
			} else {
				require.Equal(t, errCheckpointNotFound(checkpointID),
					curCase.IDToReqCall(checkpointID))
			}

			expectedErr := fmt.Errorf("canGetExperimentError")
			authZExp.On("CanGetExperiment", mock.Anything, curUser, mock.Anything).
				Return(false, expectedErr).Once()
			require.Equal(t, expectedErr, curCase.IDToReqCall(checkpointID))

			expectedErr = status.Error(codes.PermissionDenied, curCase.DenyFuncName+"Error")
			authZExp.On("CanGetExperiment", mock.Anything, curUser, mock.Anything).
				Return(true, nil).Once()
			authZExp.On(curCase.DenyFuncName, mock.Anything, curUser, mock.Anything).
				Return(fmt.Errorf(curCase.DenyFuncName + "Error")).Once()
			require.Equal(t, expectedErr, curCase.IDToReqCall(checkpointID))
		}
	}
}
