//go:build integration
// +build integration

package db

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/proto/pkg/checkpointv1"
	"github.com/determined-ai/determined/proto/pkg/trialv1"
)

func TestCheckpointMetadata(t *testing.T) {
	etc.SetRootPath(rootFromDB)
	db := MustResolveTestPostgres(t)
	MustMigrateTestPostgres(t, db, migrationsFromDB)

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
			user := requireMockUser(t, db)
			exp := requireMockExperiment(t, db, user)
			tr := requireMockTrial(t, db, exp)

			ckptUuid := uuid.NewString()
			latestBatch := int32(10)
			ckpt := &trialv1.CheckpointMetadata{
				TrialId:           int32(tr.ID),
				Uuid:              ckptUuid,
				Resources:         map[string]int64{"ok": 1.0},
				Framework:         "some framework",
				Format:            "some format",
				DeterminedVersion: "1.0.0",
				LatestBatch:       latestBatch,
			}
			err := db.AddCheckpointMetadata(context.TODO(), ckpt)
			require.NoError(t, err)

			var m *trialv1.TrialMetrics
			const metricValue = 1.0
			if tt.hasValidation {
				m = &trialv1.TrialMetrics{
					TrialId:     int32(tr.ID),
					LatestBatch: latestBatch,
					Metrics: &structpb.Struct{
						Fields: map[string]*structpb.Value{
							defaultSearcherMetric: {
								Kind: &structpb.Value_NumberValue{
									NumberValue: metricValue,
								},
							},
						},
					},
					BatchMetrics: []*structpb.Struct{},
				}
				err := db.AddValidationMetrics(context.TODO(), m)
				require.NoError(t, err)
			}

			requireCheckpointOk := func(
				expected *trialv1.CheckpointMetadata,
				actual checkpointv1.Checkpoint,
			) {
				require.Equal(t, expected, &trialv1.CheckpointMetadata{
					TrialId:           actual.TrialId,
					Uuid:              actual.Uuid,
					Resources:         actual.Resources,
					Framework:         actual.Framework,
					Format:            actual.Format,
					DeterminedVersion: actual.DeterminedVersion,
					LatestBatch:       actual.BatchNumber,
				})
				if tt.hasValidation {
					require.Equal(t, metricValue, actual.SearcherMetric.Value)
					require.Equal(t, checkpointv1.State_STATE_COMPLETED, actual.ValidationState)
					require.NotNil(t, actual.Metrics)
				} else {
					require.Nil(t, actual.SearcherMetric)
					require.Equal(t, checkpointv1.State_STATE_UNSPECIFIED, actual.ValidationState)
					require.Nil(t, actual.Metrics)
				}
			}

			var retCkpt checkpointv1.Checkpoint
			err = db.QueryProto("get_checkpoint", &retCkpt, ckptUuid)
			require.NoError(t, err, "failed to get checkpoint")
			requireCheckpointOk(ckpt, retCkpt)

			var retCkpts []*checkpointv1.Checkpoint
			err = db.QueryProto("get_checkpoints_for_trial", &retCkpts, tr.ID)
			require.NoError(t, err)
			require.Len(t, retCkpts, 1)
			requireCheckpointOk(ckpt, *retCkpts[0])

			retCkpts = nil
			err = db.QueryProto("get_checkpoints_for_experiment", &retCkpts, exp.ID)
			require.NoError(t, err)
			require.Len(t, retCkpts, 1)
			requireCheckpointOk(ckpt, *retCkpts[0])
		})
	}
}
