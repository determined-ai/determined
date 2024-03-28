//go:build integration
// +build integration

package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	apiPkg "github.com/determined-ai/determined/master/internal/api"
	authz2 "github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/checkpointv1"
	"github.com/determined-ai/determined/proto/pkg/commonv1"
	"github.com/determined-ai/determined/proto/pkg/modelv1"
	"github.com/determined-ai/determined/proto/pkg/trialv1"
)

func createVersionTwoCheckpoint(
	ctx context.Context, t *testing.T, api *apiServer, curUser model.User, resources map[string]int64,
) string {
	tr, task := createTestTrial(t, api, curUser)

	aID := model.AllocationID(string(task.TaskID) + "-1")
	aIn := &model.Allocation{
		AllocationID: aID,
		TaskID:       task.TaskID,
		Slots:        1,
		ResourcePool: "somethingelse",
		StartTime:    ptrs.Ptr(time.Now().UTC().Truncate(time.Millisecond)),
	}
	require.NoError(t, db.AddAllocation(ctx, aIn))

	checkpoint := &model.CheckpointV2{
		ID:           0,
		UUID:         uuid.New(),
		TaskID:       task.TaskID,
		AllocationID: &aID,
		ReportTime:   time.Now(),
		State:        model.ActiveState,
		Resources:    resources,
		Metadata: map[string]interface{}{
			"framework":          "tensortorch",
			"determined_version": "1.0.0",
			"steps_completed":    5,
		},
	}
	require.NoError(t, db.AddCheckpointMetadata(ctx, checkpoint, tr.ID))

	return checkpoint.UUID.String()
}

// can't use api.GetCheckpoint since we don't include size.
func getCheckpointSizeResourcesState(ctx context.Context, t *testing.T, uuid string) (
	int, map[string]int64, model.State,
) {
	out := struct {
		bun.BaseModel `bun:"table:checkpoints_view"`
		Size          int
		State         model.State
		Resources     map[string]int64
	}{}
	err := db.Bun().NewSelect().Model(&out).Where("uuid = ?", uuid).Scan(ctx)
	require.NoError(t, err)

	return out.Size, out.Resources, out.State
}

// Only returns first trial.
func getTrialSizeFromUUID(ctx context.Context, t *testing.T, uuid string) int {
	out := struct {
		bun.BaseModel  `bun:"table:trials"`
		CheckpointSize int
	}{}
	err := db.Bun().NewSelect().Model(&out).
		Where("uuid = ?", uuid).
		Join("JOIN checkpoints_view ON checkpoints_view.trial_id = trials.id").
		Limit(1).
		Scan(ctx)
	require.NoError(t, err)

	return out.CheckpointSize
}

// Only returns first experiment.
func getExperimentSizeFromUUID(ctx context.Context, t *testing.T, uuid string) int {
	out := struct {
		bun.BaseModel  `bun:"table:experiments"`
		CheckpointSize int
	}{}
	err := db.Bun().NewSelect().Model(&out).
		ColumnExpr("experiments.checkpoint_size AS checkpoint_size").
		Where("uuid = ?", uuid).
		Join("JOIN trials ON experiments.id = trials.experiment_id").
		Join("JOIN checkpoints_view ON checkpoints_view.trial_id = trials.id").
		Limit(1).
		Scan(ctx)
	require.NoError(t, err)

	return out.CheckpointSize
}

func TestCheckpointsOnArchivedSteps(t *testing.T) {
	api, curUser, ctx := setupAPITest(t, nil)

	// Create steps and validation so that we have steps / validations on 0 and 1
	// archived and unarchived.
	trialRunID := 0
	trial, task := createTestTrial(t, api, curUser)
	for _, shouldArchive := range []bool{false, true} {
		if shouldArchive {
			_, err := db.Bun().NewUpdate().Table("runs"). // TODO(nick-runs) call runs package.
									Set("restart_id = 1").
									Where("id = ?", trial.ID).
									Exec(ctx)
			require.NoError(t, err)
			trialRunID++
		}

		for i := 0; i < 3; i++ {
			expectedMetrics, err := structpb.NewStruct(map[string]any{
				"expected": fmt.Sprintf("%t-%d", shouldArchive, i),
			})
			require.NoError(t, err)

			step := int32(i)

			for _, group := range []string{
				model.ValidationMetricGroup.ToString(),
				model.TrainingMetricGroup.ToString(),
			} {
				_, err = api.ReportTrialMetrics(ctx, &apiv1.ReportTrialMetricsRequest{
					Metrics: &trialv1.TrialMetrics{
						TrialId:        int32(trial.ID),
						TrialRunId:     int32(trialRunID),
						StepsCompleted: &step,
						Metrics: &commonv1.Metrics{
							AvgMetrics: expectedMetrics,
						},
					},
					Group: group,
				})
				require.NoError(t, err)
			}
		}
	}
	expected := "true-1"

	// Checkpoint on the 1.
	checkpointMeta, err := structpb.NewStruct(map[string]any{
		"steps_completed": 1,
	})
	require.NoError(t, err)
	_, err = api.ReportCheckpoint(ctx, &apiv1.ReportCheckpointRequest{
		Checkpoint: &checkpointv1.Checkpoint{
			TaskId:       string(task.TaskID),
			AllocationId: nil,
			Uuid:         uuid.New().String(),
			ReportTime:   timestamppb.New(time.Now()),
			Resources:    nil,
			Metadata:     checkpointMeta,
			State:        checkpointv1.State_STATE_COMPLETED,
		},
	})
	require.NoError(t, err)

	// We should only have one checkpoint.
	checkpoints, err := api.GetExperimentCheckpoints(ctx, &apiv1.GetExperimentCheckpointsRequest{
		Id: int32(trial.ExperimentID),
	})
	require.NoError(t, err)
	require.Len(t, checkpoints.Checkpoints, 1)

	actual := checkpoints.Checkpoints[0]
	require.Equal(t, map[string]any{"expected": expected},
		actual.Training.TrainingMetrics.AvgMetrics.AsMap())
	require.Equal(t, map[string]any{"expected": expected},
		actual.Training.ValidationMetrics.AvgMetrics.AsMap())
}

func TestCheckpointReturned(t *testing.T) {
	// This tries to test all places where we will return a checkpointv1.Checkpoint.
	api, curUser, ctx := setupAPITest(t, nil)
	trial, task := createTestTrial(t, api, curUser)

	checkpointStorage, err := structpb.NewStruct(map[string]any{
		"type":        "shared_fs",
		"host_path":   uuid.New().String(),
		"propagation": "private",
	})
	require.NoError(t, err)

	reportResponse, err := api.RunPrepareForReporting(ctx, &apiv1.RunPrepareForReportingRequest{
		RunId: int32(trial.ID),
	})
	require.NoError(t, err)
	require.Nil(t, reportResponse.StorageId)

	reportResponse, err = api.RunPrepareForReporting(ctx, &apiv1.RunPrepareForReportingRequest{
		RunId:             int32(trial.ID),
		CheckpointStorage: checkpointStorage,
	})
	require.NoError(t, err)
	require.NotNil(t, reportResponse.StorageId)
	checkpointMeta, err := structpb.NewStruct(map[string]any{
		"steps_completed": 1,
	})
	require.NoError(t, err)
	checkpointID := uuid.New().String()
	checkpoint := &checkpointv1.Checkpoint{
		TaskId:       string(task.TaskID),
		AllocationId: nil,
		Uuid:         checkpointID,
		ReportTime:   timestamppb.New(time.Now().UTC().Truncate(time.Millisecond)),
		Resources:    map[string]int64{"x": 128, "y/": 0},
		Metadata:     checkpointMeta,
		State:        checkpointv1.State_STATE_COMPLETED,
		StorageId:    reportResponse.StorageId,
	}
	_, err = api.ReportCheckpoint(ctx, &apiv1.ReportCheckpointRequest{
		Checkpoint: checkpoint,
	})
	require.NoError(t, err)

	// Create expected.
	getExperimentRes, err := api.GetExperiment(ctx, &apiv1.GetExperimentRequest{
		ExperimentId: int32(trial.ExperimentID),
	})
	require.NoError(t, err)
	int32TrialID := int32(trial.ID)
	int32ExperimentID := int32(trial.ExperimentID)
	checkpoint.Training = &checkpointv1.CheckpointTrainingMetadata{
		TrialId:           &int32TrialID,
		ExperimentId:      &int32ExperimentID,
		ExperimentConfig:  getExperimentRes.Config,
		TrainingMetrics:   &commonv1.Metrics{},
		ValidationMetrics: &commonv1.Metrics{},
	}
	expected, err := json.MarshalIndent(checkpoint, "", "  ")
	require.NoError(t, err)

	assertCheckpoints := func(ckpts ...*checkpointv1.Checkpoint) {
		require.Len(t, ckpts, 1)

		actual, err := json.MarshalIndent(ckpts[0], "", "  ")
		require.NoError(t, err)

		require.Equal(t, string(expected), string(actual))
	}

	t.Run("GetCheckpoint returns storageID", func(t *testing.T) {
		res, err := api.GetCheckpoint(ctx, &apiv1.GetCheckpointRequest{
			CheckpointUuid: checkpointID,
		})
		require.NoError(t, err)
		assertCheckpoints(res.Checkpoint)
	})

	t.Run("GetExperimentCheckpoints returns storageID", func(t *testing.T) {
		res, err := api.GetExperimentCheckpoints(ctx, &apiv1.GetExperimentCheckpointsRequest{
			Id: int32(trial.ExperimentID),
		})
		require.NoError(t, err)
		assertCheckpoints(res.Checkpoints...)
	})

	t.Run("GetTrialCheckpoints returns storageID", func(t *testing.T) {
		res, err := api.GetTrialCheckpoints(ctx, &apiv1.GetTrialCheckpointsRequest{
			Id: int32(trial.ID),
		})
		require.NoError(t, err)
		assertCheckpoints(res.Checkpoints...)
	})

	t.Run("Model verson APIs", func(t *testing.T) {
		modelName := uuid.New().String()
		_, err := api.PostModel(ctx, &apiv1.PostModelRequest{
			Name: modelName,
		})
		require.NoError(t, err)

		res, err := api.PostModelVersion(ctx, &apiv1.PostModelVersionRequest{
			ModelName:      modelName,
			CheckpointUuid: checkpointID,
		})
		require.NoError(t, err)
		assertCheckpoints(res.ModelVersion.Checkpoint)

		patchRes, err := api.PatchModelVersion(ctx, &apiv1.PatchModelVersionRequest{
			ModelName:       modelName,
			ModelVersionNum: res.ModelVersion.Version,
			ModelVersion: &modelv1.PatchModelVersion{
				Comment: wrapperspb.String("new comment"),
			},
		})
		require.NoError(t, err)
		assertCheckpoints(patchRes.ModelVersion.Checkpoint)

		versionRes, err := api.GetModelVersion(ctx, &apiv1.GetModelVersionRequest{
			ModelName:       modelName,
			ModelVersionNum: res.ModelVersion.Version,
		})
		require.NoError(t, err)
		assertCheckpoints(versionRes.ModelVersion.Checkpoint)

		versionsRes, err := api.GetModelVersions(ctx, &apiv1.GetModelVersionsRequest{
			ModelName: modelName,
		})
		require.NoError(t, err)
		require.Len(t, versionsRes.ModelVersions, 1)
		assertCheckpoints(versionsRes.ModelVersions[0].Checkpoint)
	})
}

func TestCheckpointRemoveFilesPrefixAndEmpty(t *testing.T) {
	api, _, ctx := setupAPITest(t, nil)
	_, err := api.CheckpointsRemoveFiles(ctx, &apiv1.CheckpointsRemoveFilesRequest{
		CheckpointUuids: []string{uuid.New().String()},
		CheckpointGlobs: []string{"../../**"},
	})
	require.Equal(t,
		status.Errorf(codes.InvalidArgument, "glob '../../**' cannot contain '..'"), err)

	_, err = api.CheckpointsRemoveFiles(ctx, &apiv1.CheckpointsRemoveFilesRequest{
		CheckpointUuids: []string{uuid.New().String()},
		CheckpointGlobs: []string{"o", ""},
	})
	require.Equal(t,
		status.Errorf(codes.InvalidArgument, "cannot have empty string glob"), err)
}

func TestPatchCheckpoint(t *testing.T) {
	api, curUser, ctx := setupAPITest(t, nil)

	startingResources := map[string]int64{
		"a": 1,
		"b": 2,
		"c": 7,
	}
	uuid := createVersionTwoCheckpoint(ctx, t, api, curUser, startingResources)
	// Don't send an update.
	_, err := api.PatchCheckpoints(ctx, &apiv1.PatchCheckpointsRequest{
		Checkpoints: []*checkpointv1.PatchCheckpoint{
			{
				Uuid:      uuid,
				Resources: nil,
			},
		},
	})
	require.NoError(t, err)
	actualSize, actualResources, actualState := getCheckpointSizeResourcesState(ctx, t, uuid)
	require.Equal(t, 10, actualSize)
	require.Equal(t, startingResources, actualResources)
	require.Equal(t, model.ActiveState, actualState)

	// Send an update with same resources as what we have.
	_, err = api.PatchCheckpoints(ctx, &apiv1.PatchCheckpointsRequest{
		Checkpoints: []*checkpointv1.PatchCheckpoint{
			{
				Uuid: uuid,
				Resources: &checkpointv1.PatchCheckpoint_OptionalResources{
					Resources: startingResources,
				},
			},
		},
	})
	require.NoError(t, err)
	actualSize, actualResources, actualState = getCheckpointSizeResourcesState(ctx, t, uuid)
	require.Equal(t, 10, actualSize)
	require.Equal(t, startingResources, actualResources)
	require.Equal(t, model.ActiveState, actualState)
	require.Equal(t, 10, getTrialSizeFromUUID(ctx, t, uuid))
	require.Equal(t, 10, getExperimentSizeFromUUID(ctx, t, uuid))

	// Partially delete checkpoint
	resources := map[string]int64{
		"a": 1,
	}
	_, err = api.PatchCheckpoints(ctx, &apiv1.PatchCheckpointsRequest{
		Checkpoints: []*checkpointv1.PatchCheckpoint{
			{
				Uuid: uuid,
				Resources: &checkpointv1.PatchCheckpoint_OptionalResources{
					Resources: resources,
				},
			},
		},
	})
	require.NoError(t, err)
	actualSize, actualResources, actualState = getCheckpointSizeResourcesState(ctx, t, uuid)
	require.Equal(t, 1, actualSize)
	require.Equal(t, resources, actualResources)
	require.Equal(t, model.PartiallyDeletedState, actualState)
	require.Equal(t, 1, getTrialSizeFromUUID(ctx, t, uuid))
	require.Equal(t, 1, getExperimentSizeFromUUID(ctx, t, uuid))

	// Full delete checkpoint.
	_, err = api.PatchCheckpoints(ctx, &apiv1.PatchCheckpointsRequest{
		Checkpoints: []*checkpointv1.PatchCheckpoint{
			{
				Uuid: uuid,
				Resources: &checkpointv1.PatchCheckpoint_OptionalResources{
					Resources: nil,
				},
			},
		},
	})
	require.NoError(t, err)
	actualSize, actualResources, actualState = getCheckpointSizeResourcesState(ctx, t, uuid)
	require.Equal(t, 1, actualSize) // Size and resources don't get cleared.
	require.Equal(t, resources, actualResources)
	require.Equal(t, model.DeletedState, actualState)
	require.Equal(t, 0, getTrialSizeFromUUID(ctx, t, uuid))
	require.Equal(t, 0, getExperimentSizeFromUUID(ctx, t, uuid))

	// Test metadata.json special handling.
	startingResources = map[string]int64{
		"test": 1,
	}
	uuid = createVersionTwoCheckpoint(ctx, t, api, curUser, startingResources)
	// Sending extra metadata.json is fine.
	_, err = api.PatchCheckpoints(ctx, &apiv1.PatchCheckpointsRequest{
		Checkpoints: []*checkpointv1.PatchCheckpoint{
			{
				Uuid: uuid,
				Resources: &checkpointv1.PatchCheckpoint_OptionalResources{
					Resources: map[string]int64{"test": 1, "metadata.json": 2},
				},
			},
		},
	})
	require.NoError(t, err)
	actualSize, actualResources, actualState = getCheckpointSizeResourcesState(ctx, t, uuid)
	require.Equal(t, 3, actualSize)
	require.Equal(t, map[string]int64{"test": 1, "metadata.json": 2}, actualResources)
	require.Equal(t, model.ActiveState, actualState)
	require.Equal(t, 3, getTrialSizeFromUUID(ctx, t, uuid))
	require.Equal(t, 3, getExperimentSizeFromUUID(ctx, t, uuid))

	// Now that we have it not sending it causes partial deletion.
	_, err = api.PatchCheckpoints(ctx, &apiv1.PatchCheckpointsRequest{
		Checkpoints: []*checkpointv1.PatchCheckpoint{
			{
				Uuid: uuid,
				Resources: &checkpointv1.PatchCheckpoint_OptionalResources{
					Resources: map[string]int64{"test": 1},
				},
			},
		},
	})
	require.NoError(t, err)
	actualSize, actualResources, actualState = getCheckpointSizeResourcesState(ctx, t, uuid)
	require.Equal(t, 1, actualSize)
	require.Equal(t, map[string]int64{"test": 1}, actualResources)
	require.Equal(t, model.PartiallyDeletedState, actualState)
	require.Equal(t, 1, getTrialSizeFromUUID(ctx, t, uuid))
	require.Equal(t, 1, getExperimentSizeFromUUID(ctx, t, uuid))
}

func TestCheckpointAuthZ(t *testing.T) {
	api, authZExp, _, curUser, ctx := setupExpAuthTest(t, nil)
	authZModel := getMockModelAuth()

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
		{"CanGetExperimentArtifacts", func(id string) error {
			_, err := api.GetTrialMetricsByCheckpoint(ctx,
				&apiv1.GetTrialMetricsByCheckpointRequest{CheckpointUuid: id})
			return err
		}, false},
	}

	checkpointID := createVersionTwoCheckpoint(ctx, t, api, curUser, nil)
	for _, curCase := range cases {
		notFoundUUID := uuid.New().String()
		if curCase.UseMultiCheckpointError {
			require.Equal(t, errCheckpointsNotFound([]string{notFoundUUID}),
				curCase.IDToReqCall(notFoundUUID))
		} else {
			require.Equal(t, apiPkg.NotFoundErrs("checkpoint", notFoundUUID, true),
				curCase.IDToReqCall(notFoundUUID))
		}

		authZExp.On("CanGetExperiment", mock.Anything, curUser, mock.Anything).
			Return(authz2.PermissionDeniedError{}).Once()
		if curCase.UseMultiCheckpointError {
			require.Equal(t, errCheckpointsNotFound([]string{checkpointID}),
				curCase.IDToReqCall(checkpointID))
		} else {
			require.Equal(t, apiPkg.NotFoundErrs("checkpoint", checkpointID, true),
				curCase.IDToReqCall(checkpointID))
		}

		expectedErr := fmt.Errorf("canGetExperimentError")
		authZExp.On("CanGetExperiment", mock.Anything, curUser, mock.Anything).
			Return(expectedErr).Once()
		authZModel.On("CanGetModel", mock.Anything, mock.Anything,
			mock.Anything, mock.Anything).Return(authz2.PermissionDeniedError{}).Once()
		require.Equal(t, expectedErr, curCase.IDToReqCall(checkpointID))

		expectedErr = status.Error(codes.PermissionDenied, curCase.DenyFuncName+"Error")
		authZExp.On("CanGetExperiment", mock.Anything, curUser, mock.Anything).
			Return(nil).Once()
		authZModel.On("CanGetModel", mock.Anything, mock.Anything,
			mock.Anything, mock.Anything).Return(authz2.PermissionDeniedError{}).Once()
		authZExp.On(curCase.DenyFuncName, mock.Anything, curUser, mock.Anything).
			Return(fmt.Errorf(curCase.DenyFuncName + "Error")).Once()
		require.Equal(t, expectedErr, curCase.IDToReqCall(checkpointID))
	}
}
