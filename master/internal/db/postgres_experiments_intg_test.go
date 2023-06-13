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
	ctx := context.Background()
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
		err := AddCheckpointMetadata(ctx, &ckpt)
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
	ctx := context.Background()
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
				AllocationID: &a.AllocationID,
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
			err := AddCheckpointMetadata(ctx, &ckpt)
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
				err = db.AddValidationMetrics(ctx, m)
				require.NoError(t, err)
			}

			requireCheckpointOk := func(
				expected *model.CheckpointV2, actual *checkpointv1.Checkpoint,
			) {
				conv := protoconverter.ProtoConverter{}
				require.Equal(t, expected.TaskID, model.TaskID(actual.TaskId))
				require.Equal(t, *expected.AllocationID, model.AllocationID(*actual.AllocationId))
				require.NoError(t, conv.Error())
				require.Equal(t, expected.UUID, conv.ToUUID(actual.Uuid))
				require.Equal(t, expected.ReportTime.Truncate(time.Millisecond),
					actual.ReportTime.AsTime().Truncate(time.Millisecond))
				require.Equal(t, expected.Resources, actual.Resources)
				require.Equal(t, expected.Metadata, actual.Metadata.AsMap())
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

func TestMetricNames(t *testing.T) {
	ctx := context.Background()

	require.NoError(t, etc.SetRootPath(RootFromDB))
	db := MustResolveTestPostgres(t)
	MustMigrateTestPostgres(t, db, MigrationsFromDB)

	actualNames, err := db.MetricNames(ctx, []int{-1})
	require.NoError(t, err)
	require.Len(t, actualNames[model.TrainingMetricType], 0)
	require.Len(t, actualNames[model.ValidationMetricType], 0)

	user := RequireMockUser(t, db)

	exp := RequireMockExperiment(t, db, user)
	trial1 := RequireMockTrial(t, db, exp).ID
	addMetrics(ctx, t, db, trial1, `[{"a":1}, {"b":2}]`, `[{"b":2, "c":3}]`, false)
	trial2 := RequireMockTrial(t, db, exp).ID
	addMetrics(ctx, t, db, trial2, `[{"b":1}, {"d":2}]`, `[{"f":"test"}]`, false)

	actualNames, err = db.MetricNames(ctx, []int{exp.ID})
	require.NoError(t, err)
	require.Equal(t, []string{"a", "b", "d"}, actualNames[model.TrainingMetricType])
	require.Equal(t, []string{"b", "c", "f"}, actualNames[model.ValidationMetricType])

	addMetricCustomTime(ctx, t, trial2, time.Now())
	runSummaryMigration(t)

	actualNames, err = db.MetricNames(ctx, []int{exp.ID})
	require.NoError(t, err)
	require.Equal(t, []string{"a", "b", "d"}, actualNames[model.TrainingMetricType])
	require.Equal(t, []string{"b", "c", "f", "val_loss"}, actualNames[model.ValidationMetricType])

	exp = RequireMockExperiment(t, db, user)
	trial1 = RequireMockTrial(t, db, exp).ID
	addTestTrialMetrics(ctx, t, db, trial1,
		`{"inference": [{"a":1}, {"b":2}], "golabi": [{"b":2, "c":3}]}`)
	trial2 = RequireMockTrial(t, db, exp).ID
	addTestTrialMetrics(ctx, t, db, trial2,
		`{"inference": [{"b":1}, {"d":2}], "golabi": [{"f":"test"}]}`)

	actualNames, err = db.MetricNames(ctx, []int{exp.ID})
	require.NoError(t, err)
	require.Equal(t, []string{"a", "b", "d"}, actualNames[model.MetricType("inference")])
	require.Equal(t, []string{"b", "c", "f"}, actualNames[model.MetricType("golabi")])
}

func TestMetricBatchesMilestones(t *testing.T) {
	ctx := context.Background()
	require.NoError(t, etc.SetRootPath(RootFromDB))
	db := MustResolveTestPostgres(t)
	MustMigrateTestPostgres(t, db, MigrationsFromDB)
	user := RequireMockUser(t, db)
	exp := RequireMockExperiment(t, db, user)

	startTime := time.Time{}

	trial1 := RequireMockTrial(t, db, exp).ID
	addTestTrialMetrics(ctx, t, db, trial1,
		`{"inference": [{"a":1}, {"b":2}], "golabi": [{"b":2, "c":3}]}`)
	trial2 := RequireMockTrial(t, db, exp).ID
	addTestTrialMetrics(ctx, t, db, trial2,
		`{"inference": [{"b":1}, {"d":2}], "golabi": [{"f":"test"}]}`)

	batches, _, err := MetricBatches(exp.ID, "a", startTime, model.MetricType("inference"))
	require.NoError(t, err)
	require.Len(t, batches, 1)
	require.Equal(t, batches[0], int32(1))

	batches, _, err = MetricBatches(exp.ID, "b", startTime, model.MetricType("inference"))
	require.NoError(t, err)
	require.Len(t, batches, 2, "should have 2 batches", batches, trial1, trial2)
	require.Equal(t, batches[0], int32(1))
	require.Equal(t, batches[1], int32(2))
}

func TestTopTrialsByMetric(t *testing.T) {
	ctx := context.Background()

	require.NoError(t, etc.SetRootPath(RootFromDB))
	db := MustResolveTestPostgres(t)
	MustMigrateTestPostgres(t, db, MigrationsFromDB)

	user := RequireMockUser(t, db)

	res, err := TopTrialsByMetric(ctx, -1, 1, "metric", true)
	require.NoError(t, err)
	require.Len(t, res, 0)

	exp := RequireMockExperiment(t, db, user)
	trial1 := RequireMockTrial(t, db, exp).ID
	addMetrics(ctx, t, db, trial1,
		`[{"a":-10.0}]`, // Only care about validation.
		`[{"a":1.5, "b":"NaN", "c":"-Infinity", "d":1.5}, {"d":"nonumeric", "e":1.0}]`, false)
	trial2 := RequireMockTrial(t, db, exp).ID
	addMetrics(ctx, t, db, trial2,
		`[{"a":10.5}]`,
		`[{"a":-1.5, "b":1.0, "c":"Infinity"}]`, false)

	const (
		more             = false
		less             = true
		noError          = true
		error            = false
		orderExpected    = true
		orderNotRequired = false
	)

	tests := []struct {
		name                  string
		metric                string
		lessIsBetter          bool
		limit                 int
		expectNoError         bool
		expected              []int
		expectedOrderRequired bool
	}{
		{"'a' limit 1 less", "a", less, 1, noError, []int{trial2}, orderExpected},
		{"'a' limit 1 more", "a", more, 1, noError, []int{trial1}, orderExpected},

		{"'a' limit 2 less", "a", less, 2, noError, []int{trial2, trial1}, orderExpected},
		{"'a' limit 2 more", "a", more, 2, noError, []int{trial1, trial2}, orderExpected},

		{
			"NaNs are bigger than everything less", "b", less, 2, noError,
			[]int{trial2, trial1},
			orderExpected,
		},
		{
			"NaNs are bigger than everything more", "b", more, 2, noError,
			[]int{trial1, trial2},
			orderExpected,
		},

		{
			"Infinity works as expected less", "c", less, 2, noError,
			[]int{trial1, trial2},
			orderExpected,
		},
		{
			"Infinity works as expected more", "c", more, 2, noError,
			[]int{trial2, trial1},
			orderExpected,
		},

		{"Non numeric metrics error less", "d", less, 2, error, nil, orderExpected},
		{"Non numeric metrics error more", "d", more, 2, error, nil, orderExpected},

		{
			"Metrics only reported in one trial appear first less", "e", less, 2, noError,
			[]int{trial1, trial2},
			orderExpected,
		},
		{
			"Metrics only reported in one trial appear first more", "e", more, 2, noError,
			[]int{trial1, trial2},
			orderExpected,
		},

		{
			"Metric doesn't exist order doesn't matter less", "z", less, 2, noError,
			[]int{trial1, trial2},
			orderNotRequired,
		},
		{
			"Metric doesn't exist order doesn't matter more", "z", more, 2, noError,
			[]int{trial1, trial2},
			orderNotRequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := TopTrialsByMetric(ctx, exp.ID, tt.limit, tt.metric, tt.lessIsBetter)
			if tt.expectNoError {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}

			var i32s []int32
			for _, i := range tt.expected {
				i32s = append(i32s, int32(i))
			}

			if tt.expectedOrderRequired {
				require.Equal(t, i32s, res)
			} else {
				require.ElementsMatch(t, i32s, res)
			}
		})
	}
}
