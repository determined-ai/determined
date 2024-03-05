//go:build integration
// +build integration

package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/protoutils/protoconverter"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/master/pkg/searcher"
	"github.com/determined-ai/determined/proto/pkg/checkpointv1"
	"github.com/determined-ai/determined/proto/pkg/commonv1"
	"github.com/determined-ai/determined/proto/pkg/trialv1"
)

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
			tr, task := RequireMockTrial(t, db, exp)
			a := RequireMockAllocation(t, db, task.TaskID)

			ckptUUID := uuid.New()
			stepsCompleted := int32(10)
			ckpt := model.CheckpointV2{
				UUID:         ckptUUID,
				TaskID:       task.TaskID,
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
			err := AddCheckpointMetadata(ctx, &ckpt, tr.ID)
			require.NoError(t, err)

			// We added the checkpoint relationship row.
			var res model.RunCheckpoints
			require.NoError(t, Bun().NewSelect().Model(&res).
				Where("run_id = ?", tr.ID).
				Where("checkpoint_id = ?", ckptUUID).
				Scan(ctx, &res))
			require.Equal(t, model.RunCheckpoints{
				RunID:        tr.ID,
				CheckpointID: ckptUUID,
			}, res)

			var m *trialv1.TrialMetrics
			const metricValue = 1.0
			if tt.hasValidation {
				m = &trialv1.TrialMetrics{
					TrialId:        int32(tr.ID),
					StepsCompleted: &stepsCompleted,
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
					require.Equal(t, metricValue, *actual.Training.SearcherMetric)
					require.NotNil(t, actual.Training.ValidationMetrics.AvgMetrics)
				} else {
					require.Nil(t, actual.Training.SearcherMetric)
					require.Nil(t, actual.Training.ValidationMetrics.AvgMetrics)
				}
			}

			retCkpt, err := GetCheckpoint(ctx, ckptUUID.String())
			require.NoError(t, err, "failed to get checkpoint")
			requireCheckpointOk(&ckpt, retCkpt)

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
	require.Len(t, actualNames[model.TrainingMetricGroup], 0)
	require.Len(t, actualNames[model.ValidationMetricGroup], 0)

	user := RequireMockUser(t, db)

	exp := RequireMockExperiment(t, db, user)
	trial1 := RequireMockTrialID(t, db, exp)
	addMetrics(ctx, t, db, trial1, `[{"a":1}, {"b":2}]`, `[{"b":2, "c":3}]`, false)
	trial2 := RequireMockTrialID(t, db, exp)
	addMetrics(ctx, t, db, trial2, `[{"b":1}, {"d":2}]`, `[{"f":"test"}]`, false)

	actualNames, err = db.MetricNames(ctx, []int{exp.ID})
	require.NoError(t, err)
	require.Equal(t, []string{"a", "b", "d"}, actualNames[model.TrainingMetricGroup])
	require.Equal(t, []string{"b", "c", "f"}, actualNames[model.ValidationMetricGroup])

	addMetricCustomTime(ctx, t, trial2, time.Now())
	require.NoError(t, db.withTransaction("add trial summary metrics",
		func(tx *sqlx.Tx) error {
			return db.fullTrialSummaryMetricsRecompute(ctx, tx, trial2)
		}))

	actualNames, err = db.MetricNames(ctx, []int{exp.ID})
	require.NoError(t, err)
	require.Equal(t, []string{"a", "b", "d"}, actualNames[model.TrainingMetricGroup])
	require.Equal(t, []string{"b", "c", "f", "val_loss"}, actualNames[model.ValidationMetricGroup])

	exp = RequireMockExperiment(t, db, user)
	trial1 = RequireMockTrialID(t, db, exp)
	addTestTrialMetrics(ctx, t, db, trial1,
		`{"inference": [{"a":1}, {"b":2}], "golabi": [{"b":2, "c":3}]}`)
	trial2 = RequireMockTrialID(t, db, exp)
	addTestTrialMetrics(ctx, t, db, trial2,
		`{"inference": [{"b":1}, {"d":2}], "golabi": [{"f":"test"}]}`)

	actualNames, err = db.MetricNames(ctx, []int{exp.ID})
	require.NoError(t, err)
	require.Equal(t, []string{"a", "b", "d"}, actualNames[model.MetricGroup("inference")])
	require.Equal(t, []string{"b", "c", "f"}, actualNames[model.MetricGroup("golabi")])
}

func TestExperimentByIDs(t *testing.T) {
	ctx := context.Background()

	require.NoError(t, etc.SetRootPath(RootFromDB))
	db := MustResolveTestPostgres(t)
	MustMigrateTestPostgres(t, db, MigrationsFromDB)
	user := RequireMockUser(t, db)

	externalID := uuid.New().String()
	exp := RequireMockExperimentParams(t, db, user, MockExperimentParams{
		ExternalExperimentID: &externalID,
	}, DefaultProjectID)
	trial, task := RequireMockTrial(t, db, exp)

	for _, c := range []struct {
		funcName string
		f        func(IsNotFoundCase bool) (*model.Experiment, error)
	}{
		{"ExperimentByID", func(isNotFoundCase bool) (*model.Experiment, error) {
			if isNotFoundCase {
				return ExperimentByID(ctx, -1)
			}

			return ExperimentByID(ctx, exp.ID)
		}},
		{"ExperimentByTrialID", func(isNotFoundCase bool) (*model.Experiment, error) {
			if isNotFoundCase {
				return ExperimentByTrialID(ctx, -1)
			}

			return ExperimentByTrialID(ctx, trial.ID)
		}},
		{"ExperimentByTaskID", func(isNotFoundCase bool) (*model.Experiment, error) {
			if isNotFoundCase {
				return ExperimentByTaskID(ctx, model.TaskID(uuid.New().String()))
			}

			return ExperimentByTaskID(ctx, task.TaskID)
		}},
		{"ExperimentByExternalIDTx", func(isNotFoundCase bool) (*model.Experiment, error) {
			if isNotFoundCase {
				return ExperimentByExternalIDTx(ctx, Bun(), uuid.New().String())
			}

			return ExperimentByExternalIDTx(ctx, Bun(), externalID)
		}},
	} {
		t.Run(c.funcName+"-notFound", func(t *testing.T) {
			_, err := c.f(true)
			require.ErrorIs(t, err, ErrNotFound)
		})

		t.Run(c.funcName, func(t *testing.T) {
			actual, err := c.f(false)
			require.NoError(t, err)

			require.Equal(t, exp, actual)
		})
	}
}

func TestTerminateExperimentInRestart(t *testing.T) {
	ctx := context.Background()

	require.NoError(t, etc.SetRootPath(RootFromDB))
	db := MustResolveTestPostgres(t)
	MustMigrateTestPostgres(t, db, MigrationsFromDB)
	user := RequireMockUser(t, db)

	exp := RequireMockExperiment(t, db, user)
	trial0, _ := RequireMockTrial(t, db, exp)
	trial1, _ := RequireMockTrial(t, db, exp)

	require.NoError(t, db.TerminateExperimentInRestart(exp.ID, model.ErrorState))

	actualExp, err := ExperimentByID(ctx, exp.ID)
	require.NoError(t, err)
	require.Equal(t, actualExp.State, model.ErrorState)
	require.NotNil(t, actualExp.EndTime)
	require.Nil(t, actualExp.Progress)

	for _, trialID := range []int{trial0.ID, trial1.ID} {
		actualTrial, err := TrialByID(ctx, trialID)
		require.NoError(t, err)
		require.Equal(t, actualTrial.State, model.ErrorState)
		require.NotNil(t, actualTrial.EndTime)
	}
}

func TestExperimentsTrialAndTaskIDs(t *testing.T) {
	ctx := context.Background()

	require.NoError(t, etc.SetRootPath(RootFromDB))
	db := MustResolveTestPostgres(t)
	MustMigrateTestPostgres(t, db, MigrationsFromDB)
	user := RequireMockUser(t, db)

	tIDs, taskIDs, err := ExperimentsTrialAndTaskIDs(ctx, Bun(), nil)
	require.NoError(t, err)
	require.Len(t, tIDs, 0)
	require.Len(t, taskIDs, 0)

	tIDs, taskIDs, err = ExperimentsTrialAndTaskIDs(ctx, Bun(), []int{-1})
	require.NoError(t, err)
	require.Len(t, tIDs, 0)
	require.Len(t, taskIDs, 0)

	e0 := RequireMockExperiment(t, db, user)
	e0Trial0, e0Task0 := RequireMockTrial(t, db, e0)
	e0Trial1, e0Task1 := RequireMockTrial(t, db, e0)
	e0Trial2, e0Task2 := RequireMockTrial(t, db, e0)
	e0ExpectedTrials := []int{e0Trial0.ID, e0Trial1.ID, e0Trial2.ID}
	e0ExpectedTasks := []model.TaskID{e0Task0.TaskID, e0Task1.TaskID, e0Task2.TaskID}

	tIDs, taskIDs, err = ExperimentsTrialAndTaskIDs(ctx, Bun(), []int{e0.ID})
	require.NoError(t, err)
	require.ElementsMatch(t, e0ExpectedTrials, tIDs)
	require.ElementsMatch(t, e0ExpectedTasks, taskIDs)

	tIDs, taskIDs, err = ExperimentsTrialAndTaskIDs(ctx, Bun(), []int{e0.ID, -1})
	require.NoError(t, err)
	require.ElementsMatch(t, e0ExpectedTrials, tIDs)
	require.ElementsMatch(t, e0ExpectedTasks, taskIDs)

	e1 := RequireMockExperiment(t, db, user)
	e1Trial0, e1Task0 := RequireMockTrial(t, db, e1)
	e1Trial1, e1Task1 := RequireMockTrial(t, db, e1)
	e1ExpectedTrials := []int{e1Trial0.ID, e1Trial1.ID}
	e1ExpectedTasks := []model.TaskID{e1Task0.TaskID, e1Task1.TaskID}

	tIDs, taskIDs, err = ExperimentsTrialAndTaskIDs(ctx, Bun(), []int{e1.ID})
	require.NoError(t, err)
	require.ElementsMatch(t, e1ExpectedTrials, tIDs)
	require.ElementsMatch(t, e1ExpectedTasks, taskIDs)

	tIDs, taskIDs, err = ExperimentsTrialAndTaskIDs(ctx, Bun(), []int{e0.ID, e1.ID})
	require.NoError(t, err)
	require.ElementsMatch(t, append(e0ExpectedTrials, e1ExpectedTrials...), tIDs)
	require.ElementsMatch(t, append(e0ExpectedTasks, e1ExpectedTasks...), taskIDs)
}

func TestExperimentBestSearcherValidation(t *testing.T) {
	ctx := context.Background()

	require.NoError(t, etc.SetRootPath(RootFromDB))
	db := MustResolveTestPostgres(t)
	MustMigrateTestPostgres(t, db, MigrationsFromDB)
	user := RequireMockUser(t, db)

	// Not found.
	_, err := ExperimentBestSearcherValidation(ctx, -1)
	require.ErrorIs(t, err, ErrNotFound)

	exp := RequireMockExperiment(t, db, user)
	t0 := RequireMockTrialID(t, db, exp)
	addMetrics(ctx, t, db, t0, `[]`,
		fmt.Sprintf(`[{"%[1]s": -5.0}, {"%[1]s": 1.0}]`, defaultSearcherMetric), false)

	t1 := RequireMockTrialID(t, db, exp)
	addMetrics(ctx, t, db, t1, `[]`,
		fmt.Sprintf(`[{"%[1]s": -1.0}, {"%[1]s": 5.0}]`, defaultSearcherMetric), false)

	val, err := ExperimentBestSearcherValidation(ctx, exp.ID)
	require.NoError(t, err)
	require.Equal(t, float32(-5.0), val)

	_, err = Bun().NewUpdate().Table("experiments").
		Set("config = jsonb_set(config, '{searcher,smaller_is_better}', 'false'::jsonb)").
		Where("id = ?", exp.ID).
		Exec(ctx)
	require.NoError(t, err)

	val, err = ExperimentBestSearcherValidation(ctx, exp.ID)
	require.NoError(t, err)
	require.Equal(t, float32(5.0), val)
}

func TestProjectHyperparameters(t *testing.T) {
	ctx := context.Background()

	require.NoError(t, etc.SetRootPath(RootFromDB))
	db := MustResolveTestPostgres(t)
	MustMigrateTestPostgres(t, db, MigrationsFromDB)
	user := RequireMockUser(t, db)

	workspaceID, _ := RequireMockWorkspaceID(t, db)
	projectID, _ := RequireMockProjectID(t, db, workspaceID, false)
	exp0 := RequireMockExperimentParams(t, db, user, MockExperimentParams{
		HParamNames: &[]string{"a", "b", "c"},
		ProjectID:   &projectID,
	}, DefaultProjectID)
	exp1 := RequireMockExperimentParams(t, db, user, MockExperimentParams{
		HParamNames: &[]string{"b", "c", "d"},
		ProjectID:   &projectID,
	}, DefaultProjectID)

	require.ElementsMatch(t, []string{"a", "b", "c", "d"},
		RequireGetProjectHParams(t, db, projectID))

	require.NoError(t,
		RemoveProjectHyperparameters(ctx, nil, []int32{int32(exp0.ID), int32(exp1.ID)}))
	require.Len(t, RequireGetProjectHParams(t, db, projectID), 0)

	require.NoError(t,
		RemoveProjectHyperparameters(ctx, nil, []int32{int32(exp0.ID), int32(exp1.ID)}))
	require.Len(t, RequireGetProjectHParams(t, db, projectID), 0)

	require.NoError(t,
		AddProjectHyperparameters(ctx, nil, int32(projectID), []int32{int32(exp0.ID)}))
	require.ElementsMatch(t, []string{"a", "b", "c"},
		RequireGetProjectHParams(t, db, projectID))

	require.NoError(t,
		AddProjectHyperparameters(ctx, nil, int32(projectID), []int32{int32(exp0.ID)}))
	require.ElementsMatch(t, []string{"a", "b", "c"},
		RequireGetProjectHParams(t, db, projectID))

	require.NoError(t,
		AddProjectHyperparameters(ctx, nil, int32(projectID), []int32{int32(exp1.ID)}))
	require.ElementsMatch(t, []string{"a", "b", "c", "d"},
		RequireGetProjectHParams(t, db, projectID))

	require.NoError(t,
		RemoveProjectHyperparameters(ctx, nil, []int32{int32(exp1.ID)}))
	require.ElementsMatch(t, []string{}, // TODO(!!!) this is a bug in the query.
		RequireGetProjectHParams(t, db, projectID))
}

func TestActiveLogPatternPolicies(t *testing.T) {
	ctx := context.Background()
	require.NoError(t, etc.SetRootPath(RootFromDB))

	db := MustResolveTestPostgres(t)
	MustMigrateTestPostgres(t, db, MigrationsFromDB)

	_, err := ActiveLogPolicies(ctx, -1)
	require.ErrorIs(t, err, sql.ErrNoRows)

	user := RequireMockUser(t, db)
	exp := RequireMockExperiment(t, db, user)

	policies, err := ActiveLogPolicies(ctx, exp.ID)
	require.NoError(t, err)
	require.Len(t, policies, 0)

	activeConfig, err := db.ActiveExperimentConfig(exp.ID)
	require.NoError(t, err)
	activeConfig.RawLogPolicies = expconf.LogPoliciesConfig{
		expconf.LogPolicy{RawPattern: "sub", RawAction: expconf.LogAction{
			RawCancelRetries: &expconf.LogActionCancelRetries{},
		}},
		expconf.LogPolicy{RawPattern: `\d{5}$`, RawAction: expconf.LogAction{
			RawExcludeNode: &expconf.LogActionExcludeNode{},
		}},
	}

	v, err := json.Marshal(activeConfig)
	require.NoError(t, err)
	var m map[string]any
	require.NoError(t, json.Unmarshal(v, &m))
	_, err = Bun().NewUpdate().Table("experiments").
		Where("id = ?", exp.ID).
		Set("config = ?", m).
		Exec(ctx)
	require.NoError(t, err)

	policies, err = ActiveLogPolicies(ctx, exp.ID)
	require.NoError(t, err)
	require.Equal(t, activeConfig.RawLogPolicies, policies)
}

func TestGetNonTerminalExperimentCount(t *testing.T) {
	ctx := context.Background()
	require.NoError(t, etc.SetRootPath(RootFromDB))
	db := MustResolveTestPostgres(t)
	MustMigrateTestPostgres(t, db, MigrationsFromDB)
	user := RequireMockUser(t, db)

	c, err := GetNonTerminalExperimentCount(ctx, nil)
	require.NoError(t, err)
	require.Equal(t, 0, c)
	c, err = GetNonTerminalExperimentCount(ctx, []int32{})
	require.NoError(t, err)
	require.Equal(t, 0, c)

	e0 := RequireMockExperimentParams(t, db, user, MockExperimentParams{
		State: ptrs.Ptr(model.ActiveState),
	}, DefaultProjectID)
	c, err = GetNonTerminalExperimentCount(ctx, []int32{int32(e0.ID)})
	require.NoError(t, err)
	require.Equal(t, 1, c)

	e1 := RequireMockExperimentParams(t, db, user, MockExperimentParams{
		State: ptrs.Ptr(model.CompletedState),
	}, DefaultProjectID)
	c, err = GetNonTerminalExperimentCount(ctx, []int32{int32(e1.ID)})
	require.NoError(t, err)
	require.Equal(t, 0, c)

	e2 := RequireMockExperimentParams(t, db, user, MockExperimentParams{
		State: ptrs.Ptr(model.PausedState),
	}, DefaultProjectID)
	c, err = GetNonTerminalExperimentCount(ctx, []int32{int32(e2.ID)})
	require.NoError(t, err)
	require.Equal(t, 1, c)

	c, err = GetNonTerminalExperimentCount(ctx, []int32{int32(e0.ID), int32(e1.ID)})
	require.NoError(t, err)
	require.Equal(t, 1, c)

	c, err = GetNonTerminalExperimentCount(ctx, []int32{int32(e0.ID), int32(e1.ID), int32(e2.ID)})
	require.NoError(t, err)
	require.Equal(t, 2, c)
}

func TestMetricBatchesMilestones(t *testing.T) {
	ctx := context.Background()
	require.NoError(t, etc.SetRootPath(RootFromDB))
	db := MustResolveTestPostgres(t)
	MustMigrateTestPostgres(t, db, MigrationsFromDB)
	user := RequireMockUser(t, db)
	exp := RequireMockExperiment(t, db, user)

	startTime := time.Time{}

	trial1 := RequireMockTrialID(t, db, exp)
	addTestTrialMetrics(ctx, t, db, trial1,
		`{"inference": [{"a":1}, {"b":2}], "golabi": [{"b":2, "c":3}]}`)
	trial2 := RequireMockTrialID(t, db, exp)
	addTestTrialMetrics(ctx, t, db, trial2,
		`{"inference": [{"b":1}, {"d":2}], "golabi": [{"f":"test"}]}`)

	batches, _, err := MetricBatches(exp.ID, "a", startTime, model.MetricGroup("inference"))
	require.NoError(t, err)
	require.Len(t, batches, 1)
	require.Equal(t, batches[0], int32(1))

	batches, _, err = MetricBatches(exp.ID, "b", startTime, model.MetricGroup("inference"))
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
	trial1 := RequireMockTrialID(t, db, exp)
	addMetrics(ctx, t, db, trial1,
		`[{"a":-10.0}]`, // Only care about validation.
		`[{"a":1.5, "b":"NaN", "c":"-Infinity", "d":1.5}, {"d":"nonumeric", "e":1.0}]`, false)
	trial2 := RequireMockTrialID(t, db, exp)
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

func TestDeleteExperiments(t *testing.T) {
	ctx := context.Background()

	require.NoError(t, etc.SetRootPath(RootFromDB))
	db, cleanup := MustResolveNewPostgresDatabase(t)
	defer cleanup()
	MustMigrateTestPostgres(t, db, MigrationsFromDB)
	user := RequireMockUser(t, db)

	var experimentIDs,
		trialIDs,
		checkpointIDs []int

	// Sets containing IDs of elements removed from database.
	var (
		removedExperimentIDs = make(map[int]int)
		removedTrialIDs      = make(map[int]int)
		removedCheckpointIDs = make(map[int]int)
	)

	var (
		numExpts   = 4
		numTrs     = 2 // Trials per experiment
		numChkpts  = 2 // Checkpoints per trial
		numMtrsRaw = 2 // Training metrics per trial
		numMtrsVal = 1 // Validation metrics per trial
		numMtrsGen = 2 // Generic metrics per trial
		numExptSns = 1 // Experiment snapshots per experiment
	)

	createMetric := func(sc int32, mv float64, trID int) *trialv1.TrialMetrics {
		m := &trialv1.TrialMetrics{
			TrialId:        int32(trID),
			StepsCompleted: &sc,
			Metrics: &commonv1.Metrics{
				AvgMetrics: &structpb.Struct{
					Fields: map[string]*structpb.Value{
						defaultSearcherMetric: {
							Kind: &structpb.Value_NumberValue{
								NumberValue: mv,
							},
						},
					},
				},
				BatchMetrics: []*structpb.Struct{},
			},
		}
		return m
	}

	checkPointIndex := 1
	for i := 0; i < numExpts; i++ { // Create experiments
		exp := RequireMockExperiment(t, db, user)
		experimentIDs = append(experimentIDs, exp.ID)

		for j := 0; j < numTrs; j++ { // Create trials
			tr, task := RequireMockTrial(t, db, exp)
			allocation := RequireMockAllocation(t, db, task.TaskID)
			trialIDs = append(trialIDs, tr.ID)

			for k := 0; k < numChkpts; k++ { // Create checkpoints
				ckpt := uuid.New()
				checkpoint := MockModelCheckpoint(ckpt, allocation)

				// Set checkpoint state to 'DELETED' (indicating that they cease to exist in
				// storage) so that their deletion isn't blocked by the on_checkpoint_deletion
				// trigger.
				checkpoint.State = model.DeletedState
				err := AddCheckpointMetadata(ctx, &checkpoint, tr.ID)
				require.NoError(t, err)
				checkpointIDs = append(checkpointIDs, checkpoint.ID)
				checkPointIndex++
			}

			// training metrics (raw_steps)
			mRaw1 := createMetric(10, 0.5, tr.ID)
			err := db.AddTrainingMetrics(ctx, mRaw1)
			require.NoError(t, err)
			mRaw2 := createMetric(11, 0.9, tr.ID)
			err = db.AddTrainingMetrics(ctx, mRaw2)
			require.NoError(t, err)

			//  validation metrics (raw_validations)
			mValidation := createMetric(12, 0.95, tr.ID)
			err = db.AddValidationMetrics(ctx, mValidation)
			require.NoError(t, err)

			// generic metrics (generic_metrics)
			mGeneric1 := createMetric(5, 0.8, tr.ID)
			err = db.AddTrialMetrics(ctx, mGeneric1, model.MetricGroup(GenericMetric))
			require.NoError(t, err)

			mGeneric2 := createMetric(2, 0.9, tr.ID)
			err = db.AddTrialMetrics(ctx, mGeneric2, model.MetricGroup(GenericMetric))
			require.NoError(t, err)
		}

		// Create experiment snapshot
		//nolint:exhaustruct
		config := expconf.SearcherConfig{
			RawCustomConfig: &expconf.CustomConfig{},
		}
		searcher1 := searcher.NewSearcher(3, searcher.NewSearchMethod(config), nil)
		_, err := searcher1.InitialOperations()
		require.NoError(t, err)
		_, err = searcher1.TrialExitedEarly(model.RequestID(uuid.New()), model.Errored)
		require.NoError(t, err)

		snapshot, err := searcher1.Snapshot()
		require.NoError(t, err)
		err = db.SaveSnapshot(exp.ID, 2, snapshot)
		require.NoError(t, err)
	}

	type expected struct {
		numExperiments         int
		numTrials              int
		numCheckpoints         int
		numMetricsRaw          int
		numMetricsValidation   int
		numMetricsGeneric      int
		numExperimentSnapshots int
	}

	// verifyNumsAndElems verifies correct number of rows (num) exist given column in table
	// and that no retrieved database elements are in removed.
	verifyNumAndElems := func(table string, column string, removed map[int]int, num int) {
		var ids []int
		err := Bun().NewSelect().Table(table).Column(column).Scan(context.Background(), &ids)
		require.NoError(t, err)
		require.Equalf(t, num, len(ids),
			"table=%s column=%s removed=%+v num=%d", table, column, removed, num)

		for _, id := range ids {
			_, inRm := removed[id]
			require.Equal(t, false, inRm)
		}
	}

	verifyData := func(e expected) {
		verifyNumAndElems("experiments", "id", removedExperimentIDs, e.numExperiments)
		verifyNumAndElems("trials", "id", removedTrialIDs, e.numTrials)
		verifyNumAndElems("checkpoints_v2", "id", removedCheckpointIDs, e.numCheckpoints)
		verifyNumAndElems("raw_steps", "trial_id", removedTrialIDs, e.numMetricsRaw)
		verifyNumAndElems("raw_validations", "trial_id", removedTrialIDs, e.numMetricsValidation)
		verifyNumAndElems("generic_metrics", "trial_id", removedTrialIDs, e.numMetricsGeneric)
		verifyNumAndElems("experiment_snapshots", "experiment_id", removedExperimentIDs,
			e.numExperimentSnapshots)
	}

	// addToMap adds elements from indexes st to end of ids to rmIDs.
	addToMap := func(st, end int, rmIds map[int]int, ids []int) map[int]int {
		var i int
		for i = st; i < end; i++ {
			rmIds[ids[i]] = 0
		}
		return rmIds
	}

	subtractRows := func(e expected, amt int) expected {
		e.numExperiments -= amt
		e.numTrials -= amt * numTrs
		e.numCheckpoints -= amt * numChkpts * numTrs
		e.numMetricsRaw -= amt * numMtrsRaw * numTrs
		e.numMetricsValidation -= amt * numMtrsVal * numTrs
		e.numMetricsGeneric -= amt * numMtrsGen * numTrs
		e.numExperimentSnapshots -= amt * numExptSns
		return e
	}

	// Capture current state of database tables that will be altered by experiment deletion.
	currExpts, err := Bun().NewSelect().Table("experiments").Count(ctx)
	require.NoError(t, err)

	currTrials, err := Bun().NewSelect().Table("trials").Count(ctx)
	require.NoError(t, err)

	currChkpts, err := Bun().NewSelect().Table("checkpoints_v2").Count(ctx)
	require.NoError(t, err)

	currMetricsRaw, err := Bun().NewSelect().Table("raw_steps").Count(ctx)
	require.NoError(t, err)

	currMetricsVal, err := Bun().NewSelect().Table("raw_validations").Count(ctx)
	require.NoError(t, err)

	currMetricsGen, err := Bun().NewSelect().Table("generic_metrics").Count(ctx)
	require.NoError(t, err)

	currExptSns, err := Bun().NewSelect().Table("experiment_snapshots").Count(ctx)
	require.NoError(t, err)

	oldAllocs, err := Bun().NewSelect().Table("allocations").Count(ctx)
	require.NoError(t, err)

	e := expected{
		numExperiments: currExpts, numTrials: currTrials, numCheckpoints: currChkpts,
		numMetricsRaw: currMetricsRaw, numMetricsValidation: currMetricsVal,
		numMetricsGeneric: currMetricsGen, numExperimentSnapshots: currExptSns,
	}

	verifyData(e)

	require.NoError(t, db.DeleteExperiments(ctx, experimentIDs[:1]))
	removedExperimentIDs[experimentIDs[0]] = 0
	removedTrialIDs = addToMap(0, 2, removedTrialIDs, trialIDs)
	removedCheckpointIDs = addToMap(0, 4, removedCheckpointIDs, checkpointIDs)
	e = subtractRows(e, 1)

	verifyData(e)

	require.NoError(t, db.DeleteExperiments(ctx, experimentIDs[1:3]))
	removedExperimentIDs = addToMap(1, 3, removedExperimentIDs, experimentIDs)
	removedTrialIDs = addToMap(2, 6, removedTrialIDs, trialIDs)
	removedCheckpointIDs = addToMap(4, 12, removedCheckpointIDs, checkpointIDs)
	e = subtractRows(e, 2)

	verifyData(e)

	require.NoError(t, db.DeleteExperiments(ctx, experimentIDs[3:]))
	removedExperimentIDs[experimentIDs[3]] = 0
	removedTrialIDs = addToMap(6, 8, removedTrialIDs, trialIDs)
	removedCheckpointIDs = addToMap(12, 16, removedCheckpointIDs, checkpointIDs)
	e = subtractRows(e, 1)

	verifyData(e)

	// Verify that no allocations were removed
	newAllocs, err := Bun().NewSelect().Table("allocations").Count(ctx)
	require.NoError(t, err)

	require.Equal(t, oldAllocs, newAllocs)
}

func TestProjectExperiments(t *testing.T) {
	require.NoError(t, etc.SetRootPath(RootFromDB))
	db := MustResolveTestPostgres(t)
	MustMigrateTestPostgres(t, db, MigrationsFromDB)

	// add a workspace, and project
	workspaceID, _ := RequireMockWorkspaceID(t, db)
	projectID, _ := RequireMockProjectID(t, db, workspaceID, false)
	RequireMockUser(t, db)

	t.Run("valid project with zero experiments", func(t *testing.T) {
		actualExps, err := ProjectExperiments(context.Background(), projectID)
		require.NoError(t, err)
		assert.Equal(t, len(actualExps), 0)
	})

	t.Run("valid project with experiments", func(t *testing.T) {
		// add 3 experiments
		user := RequireMockUser(t, db)
		expectedExps := make([]*model.Experiment, 0)
		for i := 0; i < 3; i++ {
			exp := RequireMockExperimentProject(t, db, user, projectID)
			expectedExps = append(expectedExps, exp)
		}
		actualExps, err := ProjectExperiments(context.Background(), projectID)
		require.NoError(t, err)
		assert.Equal(t, len(expectedExps), len(actualExps))
		validateExperimentMatch(t, expectedExps, actualExps)
	})

	t.Run("invalid project", func(t *testing.T) {
		actualExps, err := ProjectExperiments(context.Background(), -11)
		require.NoError(t, err)
		assert.Equal(t, 0, len(actualExps))
	})

	t.Run("archived project", func(t *testing.T) {
		// add archvied project to workspace
		archivedProjectID, _ := RequireMockProjectID(t, db, workspaceID, true)
		// add 3 experiments
		user := RequireMockUser(t, db)
		expectedExps := make([]*model.Experiment, 0)
		for i := 0; i < 3; i++ {
			exp := RequireMockExperimentProject(t, db, user, archivedProjectID)
			expectedExps = append(expectedExps, exp)
		}
		actualExps, err := ProjectExperiments(context.Background(), archivedProjectID)
		require.NoError(t, err)
		assert.Equal(t, len(expectedExps), len(actualExps))
		validateExperimentMatch(t, expectedExps, actualExps)
	})
}

func validateExperimentMatch(t *testing.T, expected []*model.Experiment, actual []*model.Experiment) {
	// validate that both sets of experiments are the same using jobID
	// add all expected jobIDs to a map
	m := make(map[string]*model.Experiment)
	for _, exp := range expected {
		m[string(exp.JobID)] = exp
	}

	// remove all actual jobIDs from map
	for _, actual := range actual {
		expected, ok := m[string(actual.JobID)]
		require.True(t, ok)
		require.Equal(t, expected.ProjectID, actual.ProjectID)
		require.Equal(t, expected.Username, actual.Username)
		require.Equal(t, expected.OwnerID, actual.OwnerID)
		delete(m, string(actual.JobID))
	}

	// map should be empty
	require.Equal(t, len(m), 0)
}
