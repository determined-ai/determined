//go:build integration
// +build integration

package db

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/protoutils/protoconverter"
	"github.com/determined-ai/determined/proto/pkg/checkpointv1"
	"github.com/determined-ai/determined/proto/pkg/commonv1"
	"github.com/determined-ai/determined/proto/pkg/modelv1"
	"github.com/determined-ai/determined/proto/pkg/trialv1"
)

func TestExperimentCheckpointsToGCRaw(t *testing.T) {
	require.NoError(t, etc.SetRootPath(RootFromDB))
	db := MustResolveTestPostgres(t)
	MustMigrateTestPostgres(t, db, MigrationsFromDB)

	user := RequireMockUser(t, db)
	exp := RequireMockExperiment(t, db, user)
	tr := RequireMockTrial(t, db, exp)
	a := RequireMockAllocation(t, db, tr.TaskID)
	var expectedCheckpoints []uuid.UUID
	for i := 1; i <= 3; i++ {
		ckptUUID := uuid.New()
		ckpt := MockModelCheckpoint(ckptUUID, tr, a)
		err := db.AddCheckpointMetadata(context.TODO(), &ckpt)
		require.NoError(t, err)
		if i == 2 { // add this checkpoint to the model registry
			err = addCheckpointToModelRegistry(db, ckptUUID, user)
			require.NoError(t, err)
		} else {
			expectedCheckpoints = append(expectedCheckpoints, ckptUUID)
		}
	}

	checkpoints, err := db.ExperimentCheckpointsToGCRaw(
		exp.ID,
		0,
		0,
		0,
	)
	require.NoError(t, err)
	require.Equal(t, expectedCheckpoints, checkpoints)
}

func addCheckpointToModelRegistry(db *PgDB, checkpointUUID uuid.UUID, user model.User) error {
	// Insert a model.
	now := time.Now()
	mdl := model.Model{
		Name:            uuid.NewString(),
		Description:     "some important model",
		CreationTime:    now,
		LastUpdatedTime: now,
		Labels:          []string{"some other label"},
		Username:        user.Username,
		WorkspaceID:     1,
	}
	mdlNotes := "some notes1"
	var pmdl modelv1.Model
	if err := db.QueryProto(
		"insert_model", &pmdl, mdl.Name, mdl.Description, emptyMetadata,
		strings.Join(mdl.Labels, ","), mdlNotes, user.ID, mdl.WorkspaceID,
	); err != nil {
		return fmt.Errorf("inserting a model: %w", err)
	}

	// Register checkpoints
	var retCkpt1 checkpointv1.Checkpoint
	if err := db.QueryProto("get_checkpoint", &retCkpt1, checkpointUUID); err != nil {
		return fmt.Errorf("getting checkpoint: %w", err)
	}

	addmv := modelv1.ModelVersion{
		Model:      &pmdl,
		Checkpoint: &retCkpt1,
		Name:       "checkpoint exp",
		Comment:    "empty",
	}
	var mv modelv1.ModelVersion
	if err := db.QueryProto(
		"insert_model_version", &mv, pmdl.Id, retCkpt1.Uuid, addmv.Name, addmv.Comment,
		emptyMetadata, strings.Join(addmv.Labels, ","), addmv.Notes, user.ID,
	); err != nil {
		return fmt.Errorf("inserting model version: %w", err)
	}

	return nil
}

func TestCheckpointMetadata(t *testing.T) {
	require.NoError(t, etc.SetRootPath(RootFromDB))
	db := MustResolveTestPostgres(t)
	MustMigrateTestPostgres(t, db, MigrationsFromDB)

	tests := []struct {
		name          string
		hasValidation bool
	}{
		{
			name:          "checkpoints associated validations",
			hasValidation: true,
		},
		{
			name:          "checkpoints not associated validations",
			hasValidation: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := RequireMockUser(t, db)
			exp := RequireMockExperiment(t, db, user)
			tr := RequireMockTrial(t, db, exp)
			a := RequireMockAllocation(t, db, tr.TaskID)

			ckptUUID := uuid.New()
			stepsCompleted := int32(10)
			ckpt := model.CheckpointV2{
				UUID:         ckptUUID,
				TaskID:       tr.TaskID,
				AllocationID: a.AllocationID,
				ReportTime:   time.Now().UTC(),
				State:        model.CompletedState,
				Resources: map[string]int64{
					"ok": 1.0,
				},
				Metadata: map[string]interface{}{
					"framework":          "some framework",
					"determined_version": "1.0.0",
					"steps_completed":    float64(stepsCompleted),
				},
			}
			err := db.AddCheckpointMetadata(context.TODO(), &ckpt)
			require.NoError(t, err)

			var m *trialv1.TrialMetrics
			const metricValue = 1.0
			if tt.hasValidation {
				m = &trialv1.TrialMetrics{
					TrialId:        int32(tr.ID),
					StepsCompleted: stepsCompleted,
					Metrics: &commonv1.Metrics{
						AvgMetrics: &structpb.Struct{
							Fields: map[string]*structpb.Value{
								defaultSearcherMetric: {
									Kind: &structpb.Value_NumberValue{
										NumberValue: metricValue,
									},
								},
							},
						},
						BatchMetrics: []*structpb.Struct{},
					},
				}
				err = db.AddValidationMetrics(context.TODO(), m)
				require.NoError(t, err)
			}

			requireCheckpointOk := func(
				expected *model.CheckpointV2, actual *checkpointv1.Checkpoint,
			) {
				conv := protoconverter.ProtoConverter{}
				require.Equal(t, expected.TaskID, model.TaskID(actual.TaskId))
				require.Equal(t, expected.AllocationID, model.AllocationID(actual.AllocationId))
				require.NoError(t, conv.Error())
				require.Equal(t, expected.UUID, conv.ToUUID(actual.Uuid))
				require.Equal(t, expected.ReportTime.Truncate(time.Millisecond),
					actual.ReportTime.AsTime().Truncate(time.Millisecond))
				require.Equal(t, expected.Resources, actual.Resources)
				require.Equal(t, expected.Metadata, model.JSONObj(actual.Metadata.AsMap()))
				require.NoError(t, conv.Error())
				require.Equal(t, expected.State, conv.ToCheckpointState(actual.State))
				if tt.hasValidation {
					require.Equal(t, metricValue, actual.Training.SearcherMetric.Value)
					require.NotNil(t, actual.Training.ValidationMetrics.AvgMetrics)
				} else {
					require.Nil(t, actual.Training.SearcherMetric)
					require.Nil(t, actual.Training.ValidationMetrics.AvgMetrics)
				}
			}

			var retCkpt checkpointv1.Checkpoint
			err = db.QueryProto("get_checkpoint", &retCkpt, ckptUUID)
			require.NoError(t, err, "failed to get checkpoint")
			requireCheckpointOk(&ckpt, &retCkpt)

			var retCkpts []*checkpointv1.Checkpoint
			err = db.QueryProto("get_checkpoints_for_trial", &retCkpts, tr.ID)
			require.NoError(t, err)
			require.Len(t, retCkpts, 1)
			requireCheckpointOk(&ckpt, retCkpts[0])

			retCkpts = nil
			err = db.QueryProto("get_checkpoints_for_experiment", &retCkpts, exp.ID)
			require.NoError(t, err)
			require.Len(t, retCkpts, 1)
			requireCheckpointOk(&ckpt, retCkpts[0])

			latestCkpt, err := db.LatestCheckpointForTrial(tr.ID)
			require.NoError(t, err, "failed to obtain latest checkpoint")
			require.NotNil(t, latestCkpt, "checkpoint is nil")
			require.Equal(t, latestCkpt.TrialID, tr.ID)
		})
	}
}
