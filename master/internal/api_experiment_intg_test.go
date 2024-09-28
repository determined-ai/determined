//go:build integration
// +build integration

package internal

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"sync"
	"testing"
	"time"
	"unsafe"

	"github.com/determined-ai/determined/proto/pkg/checkpointv1"

	"github.com/uptrace/bun"

	"github.com/ghodss/yaml"
	"github.com/golang/protobuf/ptypes/wrappers"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"

	apiPkg "github.com/determined-ai/determined/master/internal/api"
	authz2 "github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/internal/db"
	expauth "github.com/determined-ai/determined/master/internal/experiment"
	"github.com/determined-ai/determined/master/internal/mocks"
	modelauth "github.com/determined-ai/determined/master/internal/model"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/master/test/olddata"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/commonv1"
	"github.com/determined-ai/determined/proto/pkg/experimentv1"
	"github.com/determined-ai/determined/proto/pkg/rbacv1"
	"github.com/determined-ai/determined/proto/pkg/userv1"
	"github.com/determined-ai/determined/proto/pkg/utilv1"
	"github.com/determined-ai/determined/proto/pkg/workspacev1"
)

//nolint:containedctx
type mockStream[T any] struct {
	mu   sync.Mutex
	ctx  context.Context
	data []T
}

func (m *mockStream[T]) Send(resp T) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.data = append(m.data, resp)
	return nil
}
func (m *mockStream[T]) SetHeader(metadata.MD) error   { return nil }
func (m *mockStream[T]) SendHeader(metadata.MD) error  { return nil }
func (m *mockStream[T]) SetTrailer(metadata.MD)        {}
func (m *mockStream[T]) Context() context.Context      { return m.ctx }
func (m *mockStream[T]) SendMsg(mes interface{}) error { return nil }
func (m *mockStream[T]) RecvMsg(mes interface{}) error { return nil }

func (m *mockStream[T]) getData() []T {
	m.mu.Lock()
	defer m.mu.Unlock()

	out := make([]T, len(m.data))
	copy(out, m.data)

	return out
}

var (
	authZExp   *mocks.ExperimentAuthZ
	authzModel *mocks.ModelAuthZ
)

// pgdb can be nil to use the singleton database for testing.
func setupExpAuthTest(t *testing.T, pgdb *db.PgDB,
	altMockRM ...*mocks.ResourceManager,
) (
	*apiServer, *mocks.ExperimentAuthZ, *mocks.ProjectAuthZ, model.User, context.Context,
) {
	api, projectAuthZ, _, user, ctx := setupProjectAuthZTest(t, pgdb, altMockRM...)
	if authZExp == nil {
		authZExp = &mocks.ExperimentAuthZ{}
		expauth.AuthZProvider.Register("mock", authZExp)
	}
	return api, authZExp, projectAuthZ, user, ctx
}

func getMockModelAuth() *mocks.ModelAuthZ {
	if authzModel == nil {
		authzModel = &mocks.ModelAuthZ{}
		modelauth.AuthZProvider.Register("mock", authzModel)
	}

	return authzModel
}

func createTestExp(
	t *testing.T, api *apiServer, curUser model.User, labels ...string,
) *model.Experiment {
	return createTestExpWithProjectID(t, api, curUser, 1, labels...)
}

func minExpConfToYaml(t *testing.T) string {
	bytes, err := yaml.Marshal(minExpConfig)
	require.NoError(t, err)
	return string(bytes)
}

// nolint: exhaustruct
var minExpConfig = expconf.ExperimentConfig{
	RawResources: &expconf.ResourcesConfig{
		RawResourcePool: ptrs.Ptr("kubernetes"),
	},
	RawEntrypoint: &expconf.EntrypointV0{RawEntrypoint: "test"},
	RawCheckpointStorage: &expconf.CheckpointStorageConfig{
		RawSharedFSConfig: &expconf.SharedFSConfig{
			RawHostPath: ptrs.Ptr("/"),
		},
	},
	RawHyperparameters: expconf.Hyperparameters{},
	RawReproducibility: &expconf.ReproducibilityConfig{RawExperimentSeed: ptrs.Ptr(uint32(42))},
	RawSearcher: &expconf.SearcherConfig{
		RawMetric: ptrs.Ptr("loss"),
		RawSingleConfig: &expconf.SingleConfig{
			RawMaxLength: &expconf.Length{Units: 10, Unit: "batches"},
		},
	},
}

func TestGetExperimentLabels(t *testing.T) {
	api, curUser, ctx := setupAPITest(t, nil)
	_, p0 := createProjectAndWorkspace(ctx, t, api)
	_, p1 := createProjectAndWorkspace(ctx, t, api)

	var labels []string
	for i := 0; i <= 3; i++ {
		labels = append(labels, uuid.New().String())
	}

	// Labels returned in sorted order by frequency.
	createTestExpWithProjectID(t, api, curUser, p0, labels[0], labels[1])
	createTestExpWithProjectID(t, api, curUser, p0, labels[0])
	resp, err := api.GetExperimentLabels(ctx,
		&apiv1.GetExperimentLabelsRequest{ProjectId: int32(p0)})
	require.NoError(t, err)
	require.Equal(t, labels[:2], resp.Labels)

	// Exact label arrays don't count multiple times
	// (behavior is kinda weird since Postgres can save
	// ["a", "b"] either as ["b", "a"] or ["a", "b"] breaking this distinct).
	createTestExpWithProjectID(t, api, curUser, p0, labels[2])
	createTestExpWithProjectID(t, api, curUser, p0, labels[2])
	createTestExpWithProjectID(t, api, curUser, p0, labels[2])
	resp, err = api.GetExperimentLabels(ctx,
		&apiv1.GetExperimentLabelsRequest{ProjectId: int32(p0)})
	require.NoError(t, err)
	require.Equal(t, labels[0], resp.Labels[0])

	// Second project.
	createTestExpWithProjectID(t, api, curUser, p1, labels[3])
	resp, err = api.GetExperimentLabels(ctx,
		&apiv1.GetExperimentLabelsRequest{ProjectId: int32(p1)})
	require.NoError(t, err)
	require.Equal(t, []string{labels[3]}, resp.Labels)

	// No project specified returns at least all of our labels from both projects.
	resp, err = api.GetExperimentLabels(ctx, &apiv1.GetExperimentLabelsRequest{})
	require.NoError(t, err)
	require.Subset(t, resp.Labels, labels)
}

func TestGetExperimentConfig(t *testing.T) {
	api, curUser, ctx := setupAPITest(t, nil)

	exp := createTestExp(t, api, curUser)
	expectedBytes, err := db.SingleDB().ExperimentConfigRaw(exp.ID)
	require.NoError(t, err)
	expected := make(map[string]any)
	require.NoError(t, json.Unmarshal(expectedBytes, &expected))

	resp, err := api.GetExperiment(ctx, &apiv1.GetExperimentRequest{
		ExperimentId: int32(exp.ID),
	})
	require.NoError(t, err)

	cases := []struct {
		name   string
		config *structpb.Struct
	}{
		{"GetExperimentResponse.Config", resp.Config},
		{"GetExperimentResponse.Experiment.Config", resp.Experiment.Config}, //nolint:staticcheck
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			require.Equal(t, expected, c.config.AsMap())
		})
	}
}

func TestGetExperimentHyperparameters(t *testing.T) {
	api, curUser, ctx := setupAPITest(t, nil)
	exp := createTestExp(t, api, curUser)
	expectedBytes, err := db.SingleDB().ExperimentConfigRaw(exp.ID)
	require.NoError(t, err)
	expected := make(map[string]any)
	require.NoError(t, json.Unmarshal(expectedBytes, &expected))
	expectedHyperparameter := expected["hyperparameters"].(map[string]any)
	resp, err := api.GetExperiments(ctx, &apiv1.GetExperimentsRequest{
		ExperimentIdFilter: &commonv1.Int32FieldFilter{
			Incl: []int32{int32(exp.ID)},
		},
	})
	require.NoError(t, err)
	require.Len(t, resp.Experiments, 1)
	require.Equal(t, int32(exp.ID), resp.Experiments[0].Id)

	if _, ok := resp.Experiments[0].Config.Fields["hyperparameters"]; !ok { //nolint:staticcheck
		t.Errorf("`hyperparameters` is not in config")
	}

	cases := []struct {
		name            string
		hyperparameters map[string]any
	}{
		{"GetExperimentResponse.Experiments.Config.Hyperparameters", resp.Experiments[0].Config. //nolint:staticcheck
														Fields["hyperparameters"].
														AsInterface().(map[string]any)},
		{"GetExperimentResponse.Experiments.Hyperparameters", resp.Experiments[0].Hyperparameters.AsMap()},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			require.Equal(t, expectedHyperparameter, c.hyperparameters)
		})
	}
}

func TestGetTaskContextDirectoryExperiment(t *testing.T) {
	api, curUser, ctx := setupAPITest(t, nil)

	trial, task := createTestTrial(t, api, curUser)

	expected, err := api.GetModelDef(ctx, &apiv1.GetModelDefRequest{
		ExperimentId: int32(trial.ExperimentID),
	})
	require.NoError(t, err)

	actual, err := api.GetTaskContextDirectory(ctx, &apiv1.GetTaskContextDirectoryRequest{
		TaskId: string(task.TaskID),
	})
	require.NoError(t, err)
	require.Equal(t, expected.B64Tgz, actual.B64Tgz)
}

func TestGetTaskContextDirectoryTask(t *testing.T) {
	api, _, ctx := setupAPITest(t, nil)
	task := &model.Task{TaskType: model.TaskTypeNotebook, TaskID: model.NewTaskID()}
	require.NoError(t, db.AddTask(ctx, task))

	expectedContextDirectory := []byte("expectedContextDirectory")
	_, err := db.Bun().NewInsert().Model(&model.TaskContextDirectory{
		TaskID:           task.TaskID,
		ContextDirectory: expectedContextDirectory,
	}).Exec(context.TODO())
	require.NoError(t, err)

	actual, err := api.GetTaskContextDirectory(ctx, &apiv1.GetTaskContextDirectoryRequest{
		TaskId: string(task.TaskID),
	})
	require.NoError(t, err)

	actualString, err := base64.StdEncoding.DecodeString(actual.B64Tgz)
	require.NoError(t, err)
	require.Equal(t, string(expectedContextDirectory), string(actualString))
}

func TestMoveExperimentRunID(t *testing.T) {
	api, curUser, ctx := setupAPITest(t, nil)
	_, projectID := createProjectAndWorkspace(ctx, t, api)

	t.Run("trial less move", func(t *testing.T) {
		exp := createTestExp(t, api, curUser)
		_, err := api.MoveExperiment(ctx, &apiv1.MoveExperimentRequest{
			ExperimentId:         int32(exp.ID),
			DestinationProjectId: int32(projectID),
		})
		require.NoError(t, err)
	})

	t.Run("move with trials", func(t *testing.T) {
		trial, _ := createTestTrial(t, api, curUser)
		_, err := api.MoveExperiment(ctx, &apiv1.MoveExperimentRequest{
			ExperimentId:         int32(trial.ExperimentID),
			DestinationProjectId: int32(projectID),
		})
		require.NoError(t, err)

		e, err := api.GetExperiment(ctx, &apiv1.GetExperimentRequest{
			ExperimentId: int32(trial.ExperimentID),
		})
		require.NoError(t, err)
		require.Equal(t, projectID, int(e.Experiment.ProjectId))

		var r model.Run
		require.NoError(t, db.Bun().NewSelect().Model(&r).
			Where("id = ?", trial.ID).
			Scan(ctx, &r))
		require.Equal(t, projectID, r.ProjectID)
	})
}

func TestMoveExperiments(t *testing.T) {
	api, curUser, ctx := setupAPITest(t, nil)
	_, projectID := createProjectAndWorkspace(ctx, t, api)

	t.Run("Move an experiment without filters", func(t *testing.T) {
		exp := createTestExp(t, api, curUser)
		result, err := api.MoveExperiments(ctx, &apiv1.MoveExperimentsRequest{
			ProjectId:            int32(exp.ProjectID),
			ExperimentIds:        []int32{int32(exp.ID)},
			DestinationProjectId: int32(projectID),
			Filters:              nil,
		})
		for _, v := range result.Results {
			require.Empty(t, v.Error)
		}
		require.Len(t, result.Results, 1)
		require.NoError(t, err)
	})

	t.Run("Move multiple experiments without filters", func(t *testing.T) {
		exp1 := createTestExp(t, api, curUser)
		exp2 := createTestExp(t, api, curUser)
		result, err := api.MoveExperiments(ctx, &apiv1.MoveExperimentsRequest{
			ProjectId:            int32(exp1.ProjectID),
			ExperimentIds:        []int32{int32(exp1.ID), int32(exp2.ID)},
			DestinationProjectId: int32(projectID),
			Filters:              nil,
		})
		for _, v := range result.Results {
			require.Empty(t, v.Error)
		}
		require.Len(t, result.Results, 2)
		require.NoError(t, err)
	})
	t.Run("Move no experiments with filters (no filter match)", func(t *testing.T) {
		notExsistentLable := uuid.New().String()
		createTestExp(t, api, curUser)

		result, err := api.MoveExperiments(ctx, &apiv1.MoveExperimentsRequest{
			ProjectId:            int32(projectID),
			ExperimentIds:        []int32{},
			DestinationProjectId: int32(projectID),
			Filters: &apiv1.BulkExperimentFilters{
				Labels:   []string{notExsistentLable},
				Archived: &wrappers.BoolValue{Value: true},
			},
		})
		require.Empty(t, result.Results)
		require.NoError(t, err)
	})

	t.Run("Move multiple experiments to non-existent project", func(t *testing.T) {
		exp1 := createTestExp(t, api, curUser)
		exp2 := createTestExp(t, api, curUser)
		result, err := api.MoveExperiments(ctx, &apiv1.MoveExperimentsRequest{
			ProjectId:            int32(exp1.ProjectID),
			ExperimentIds:        []int32{int32(exp1.ID), int32(exp2.ID)},
			DestinationProjectId: 0,
			Filters:              nil,
		})
		require.Nil(t, result)
		require.Error(t, err)
	})

	t.Run("Move 0 experiments", func(t *testing.T) {
		result, err := api.MoveExperiments(ctx, &apiv1.MoveExperimentsRequest{
			ProjectId:            0,
			ExperimentIds:        []int32{},
			DestinationProjectId: int32(projectID),
			Filters:              nil,
		})
		require.Empty(t, result.Results)
		require.NoError(t, err)
	})

	t.Run("Move a non-existent experiement", func(t *testing.T) {
		result, err := api.MoveExperiments(ctx, &apiv1.MoveExperimentsRequest{
			ProjectId:            0,
			ExperimentIds:        []int32{-1, 0},
			DestinationProjectId: int32(projectID),
			Filters:              nil,
		})
		require.Len(t, result.Results, 2)
		require.NoError(t, err)
	})

	t.Run("Move mixed of existent and non-existent experiments", func(t *testing.T) {
		exp := createTestExp(t, api, curUser)
		expIds := []int32{-1, 0, int32(exp.ID)}
		result, err := api.MoveExperiments(ctx, &apiv1.MoveExperimentsRequest{
			ProjectId:            int32(exp.ProjectID),
			ExperimentIds:        expIds,
			DestinationProjectId: int32(projectID),
			Filters:              nil,
		})
		successIDList := make([]int32, 0)
		errorIDList := make([]int32, 0)
		for _, v := range result.Results {
			if v.Error == "" {
				require.Empty(t, v.Error)
				successIDList = append(successIDList, v.Id)
			} else {
				require.Equal(
					t,
					v.Error,
					fmt.Sprintf("rpc error: code = NotFound desc = experiment '%d' not found", v.Id),
				)
				errorIDList = append(errorIDList, v.Id)
			}
		}
		require.Len(t, successIDList, 1)
		require.Len(t, errorIDList, 2)
		require.Equal(t, successIDList[0], int32(exp.ID))
		require.Len(t, result.Results, len(expIds))
		require.NoError(t, err)
	})
}

func TestDeleteExperimentWithoutCheckpoints(t *testing.T) {
	api, curUser, ctx := setupAPITest(t, nil)
	exp := createTestExp(t, api, curUser)
	_, err := db.Bun().NewUpdate().Table("experiments").
		Set("state = ?", model.CompletedState).
		Where("id = ?", exp.ID).Exec(ctx)
	require.NoError(t, err)

	_, err = api.DeleteExperiment(ctx, &apiv1.DeleteExperimentRequest{ExperimentId: int32(exp.ID)})
	require.NoError(t, err)

	// Delete is async so we need to retry until it completes.
	for i := 0; i < 60; i++ {
		e, err := api.GetExperiment(ctx, &apiv1.GetExperimentRequest{ExperimentId: int32(exp.ID)})
		if err != nil {
			require.Equal(t, apiPkg.NotFoundErrs("experiment", strconv.Itoa(exp.ID), true), err)
			return
		}
		require.NotEqual(t, experimentv1.State_STATE_DELETE_FAILED, e.Experiment.State)
		time.Sleep(1 * time.Second)
	}
	t.Error("expected experiment to delete after 1 minute and it did not")
}

func TestHPSearchContinueProvideConfigError(t *testing.T) {
	api, curUser, ctx := setupAPITest(t, nil)
	trial, _ := createTestTrial(t, api, curUser)

	_, err := db.Bun().NewUpdate().Table("experiments").
		Set("state = ?", model.CompletedState).
		Set("config = jsonb_set(config, '{searcher}', "+
			`'{"name": "random", "metric": "loss", "max_trials": 5, "max_length": 5}', false)`).
		Where("id = ?", trial.ExperimentID).
		Exec(ctx)
	require.NoError(t, err)

	_, err = api.ContinueExperiment(ctx, &apiv1.ContinueExperimentRequest{
		Id:             int32(trial.ExperimentID),
		OverrideConfig: `name: "new"`,
	})
	require.ErrorContains(t, err, "override config is provided and experiment")
}

func TestHPSearchContinueCompletedError(t *testing.T) {
	api, curUser, ctx := setupAPITest(t, nil)
	trial, _ := createTestTrial(t, api, curUser)

	_, err := db.Bun().NewUpdate().Table("experiments").
		Set("state = ?", model.CompletedState).
		Set("config = jsonb_set(config, '{searcher}', "+
			`'{"name": "random", "metric": "loss", "max_trials": 5, "max_length": 5}', false)`).
		Where("id = ?", trial.ExperimentID).
		Exec(ctx)
	require.NoError(t, err)
	_, err = db.Bun().NewUpdate().Table("runs").
		Set("state = ?", model.CompletedState).
		Where("id = ?", trial.ID).
		Exec(ctx)
	require.NoError(t, err)

	_, err = api.ContinueExperiment(ctx, &apiv1.ContinueExperimentRequest{
		Id: int32(trial.ExperimentID),
	})
	require.ErrorIs(t, err, errContinueHPSearchCompleted)
}

func TestPutExperimentRetainLogs(t *testing.T) {
	api, _, ctx := setupAPITest(t, nil)
	exp, trialIDs, _ := CreateTestRetentionExperiment(ctx, t, api, logRetentionConfigForever, 5)

	err := CompleteExpAndTrials(ctx, exp.Id, trialIDs)
	require.NoError(t, err)

	numDays := -1
	res, err := api.PutExperimentRetainLogs(ctx, &apiv1.PutExperimentRetainLogsRequest{
		ExperimentId: exp.Id,
		NumDays:      int32(numDays),
	})
	require.NoError(t, err)
	require.NotNil(t, res)

	newLogRetentionDays := []int32{-1, -1, -1, -1, -1}
	updatedLogRetentionDays, err := getLogRetentionDays(ctx, trialIDs)
	require.NoError(t, err)
	require.Equal(t, updatedLogRetentionDays, newLogRetentionDays)
}

func TestPutExperimentsRetainLogs(t *testing.T) {
	api, curUser, ctx := setupAPITest(t, nil)
	var expIDs []int32
	var intExpIDS []int
	var trialIDs []int
	for i := 0; i <= 5; i++ {
		trial, _ := createTestTrial(t, api, curUser)
		expIDs = append(expIDs, int32(trial.ExperimentID))
		intExpIDS = append(intExpIDS, trial.ExperimentID)
		trialIDs = append(trialIDs, trial.ID)
	}

	_, err := db.Bun().NewUpdate().Table("experiments").
		Set("state = ?", model.CompletedState).
		Where("id IN (?)", bun.In(expIDs)).
		Exec(ctx)
	require.NoError(t, err)
	_, err = db.Bun().NewUpdate().Table("runs").
		Set("state = ?", model.CompletedState).
		Where("id IN (?)", bun.In(trialIDs)).
		Exec(ctx)
	require.NoError(t, err)

	numDays := 10
	res, err := api.PutExperimentsRetainLogs(ctx, &apiv1.PutExperimentsRetainLogsRequest{
		ProjectId:     1,
		ExperimentIds: expIDs,
		NumDays:       int32(numDays),
	})
	require.NoError(t, err)
	require.NotNil(t, res)

	_, _, err = db.ExperimentsTrialAndTaskIDs(ctx, db.Bun(), intExpIDS)
	require.NoError(t, err)

	var trialLogRetentionDays []int
	err = db.Bun().NewSelect().Table("runs").
		Column("log_retention_days").
		Where("id IN (?)", bun.In(trialIDs)).
		Scan(ctx, &trialLogRetentionDays)
	require.NoError(t, err)

	for _, v := range trialLogRetentionDays {
		require.Equal(t, v, numDays)
	}
}

func TestParseAndMergeContinueConfig(t *testing.T) {
	// Blank config.
	api, curUser, ctx := setupAPITest(t, nil)
	exp := createTestExp(t, api, curUser)

	_, _, err := api.parseAndMergeContinueConfig(exp.ID, ``)
	require.NoError(t, err)

	_, _, err = api.parseAndMergeContinueConfig(exp.ID, `{}`)
	require.NoError(t, err)

	_, _, err = api.parseAndMergeContinueConfig(exp.ID, `
project: test
`)
	require.ErrorContains(t, err, "'project' in override config cannot be specified")

	_, _, err = api.parseAndMergeContinueConfig(exp.ID, `
workspace: test
`)
	require.ErrorContains(t, err, "'workspace' in override config cannot be specified")

	_, _, err = api.parseAndMergeContinueConfig(exp.ID, `
searcher:
    max_length:
        batches: 10
`)
	require.ErrorContains(t, err, "you might also need to specify searcher.name=single")

	_, _, err = api.parseAndMergeContinueConfig(exp.ID, `
searcher:
  max_length:
    batches: 10
  name: single
`)
	require.NoError(t, err)

	_, _, err = api.parseAndMergeContinueConfig(exp.ID, `
searcher:
  name: random
  metric: accuracy
  max_trials: 5
  max_length:
    batches: 1000
`)
	require.ErrorContains(t, err,
		"override config must have single searcher type got 'random' instead")

	conf := `
entrypoint: test
checkpoint_storage:
  type: shared_fs
  host_path: /tmp
  storage_path: determined-integration-checkpoints
searcher:
  metric: validation_error
  smaller_is_better: true
  name: random
  max_trials: 3
  max_length:
    batches: 10
resources:
  resource_pool: kubernetes`
	createReq := &apiv1.CreateExperimentRequest{
		ModelDefinition: []*utilv1.File{{Content: []byte{1}}},
		Config:          conf,
		ParentId:        0,
		Activate:        false,
		ProjectId:       1,
	}

	// No checkpoint specified anywhere.
	resp, err := api.CreateExperiment(ctx, createReq)
	require.NoError(t, err)
	_, _, err = api.parseAndMergeContinueConfig(int(resp.Experiment.Id), `{}`)
	require.NoError(t, err)

	_, _, err = api.parseAndMergeContinueConfig(int(resp.Experiment.Id), `
searcher:
  name: random
  metric: accuracy
  max_trials: 5
  max_length:
    batches: 1000
`)
	require.ErrorContains(t, err,
		"override config is provided and experiment is not single searcher, got 'random' instead")
}

// nolint: exhaustruct
func TestCreateExperimentCheckpointStorage(t *testing.T) {
	api, _, ctx := setupAPITest(t, nil)
	api.m.config.CheckpointStorage = expconf.CheckpointStorageConfig{}
	defer func() {
		api.m.config.CheckpointStorage = expconf.CheckpointStorageConfig{}
	}()

	conf := `
entrypoint: test
searcher:
  metric: loss
  name: single
  max_length: 10
resources:
  resource_pool: kubernetes`
	createReq := &apiv1.CreateExperimentRequest{
		ModelDefinition: []*utilv1.File{{Content: []byte{1}}},
		Config:          conf,
		ParentId:        0,
		Activate:        false,
		ProjectId:       1,
	}

	// No checkpoint specified anywhere.
	_, err := api.CreateExperiment(ctx, createReq)
	require.ErrorContains(t, err, "checkpoint_storage: type is a required property")

	// Checkpoint specified in workspace.
	workspaceLevelKey := "secretz"
	workspaceID, projectID := createProjectAndWorkspace(ctx, t, api)
	_, err = api.PatchWorkspace(ctx, &apiv1.PatchWorkspaceRequest{
		Id: int32(workspaceID),
		Workspace: &workspacev1.PatchWorkspace{
			CheckpointStorageConfig: newProtoStruct(t, map[string]any{
				"type":       "s3",
				"bucket":     "bucketz",
				"secret_key": workspaceLevelKey,
			}),
		},
	})
	require.NoError(t, err)

	createReq.ProjectId = int32(projectID)
	resp, err := api.CreateExperiment(ctx, createReq)
	require.NoError(t, err)

	expected := map[string]any{
		"type":                 "s3",
		"bucket":               "bucketz",
		"secret_key":           workspaceLevelKey, // Key doesn't get censored.
		"access_key":           nil,
		"endpoint_url":         nil,
		"prefix":               nil,
		"save_experiment_best": 0.0, // These get filled in from some default.
		"save_trial_best":      1.0, // Not sure why they are floats.
		"save_trial_latest":    1.0,
	}
	require.Equal(t, expected, resp.Config.AsMap()["checkpoint_storage"])

	// Checkpoint specified in master config.
	api.m.config.CheckpointStorage = expconf.CheckpointStorageConfig{
		RawS3Config: &expconf.S3Config{
			RawBucket:    ptrs.Ptr("masterbucket"),
			RawSecretKey: ptrs.Ptr("mastersecret"),
		},
	}

	createReq.ProjectId = 1

	resp, err = api.CreateExperiment(ctx, createReq)
	require.NoError(t, err)

	expected["bucket"] = "masterbucket"
	expected["secret_key"] = "mastersecret"
	require.Equal(t, expected, resp.Config.AsMap()["checkpoint_storage"])

	// Checkpoint specified in master config and workspace gives workspace config.
	createReq.ProjectId = int32(projectID)
	resp, err = api.CreateExperiment(ctx, createReq)
	require.NoError(t, err)

	expected["bucket"] = "bucketz"
	expected["secret_key"] = workspaceLevelKey
	require.Equal(t, expected, resp.Config.AsMap()["checkpoint_storage"])

	// Checkpoint specified in master config, expconf, and workspace gives expconf.
	createReq.Config += `
checkpoint_storage:
  type: s3
  bucket: "expconfbucket"
  `
	resp, err = api.CreateExperiment(ctx, createReq)
	require.NoError(t, err)

	expected["bucket"] = "expconfbucket"
	expected["secret_key"] = workspaceLevelKey
	require.Equal(t, expected, resp.Config.AsMap()["checkpoint_storage"])
}

func TestGetExperimentsShowTrialData(t *testing.T) {
	api, curUser, ctx := setupAPITest(t, nil)

	exp0 := createTestExp(t, api, curUser)
	trial, _ := createTestTrialWithMetrics(ctx, t, api, curUser, true)

	resp, err := api.GetExperiments(ctx, &apiv1.GetExperimentsRequest{
		ShowTrialData: true,
		ExperimentIdFilter: &commonv1.Int32FieldFilter{
			Incl: []int32{int32(exp0.ID), int32(trial.ExperimentID)},
		},
		SortBy: apiv1.GetExperimentsRequest_SORT_BY_SEARCHER_METRIC_VAL,
	})
	require.NoError(t, err)
	require.Len(t, resp.Experiments, 2)
	require.NotNil(t, resp.Experiments[0].BestTrialSearcherMetric)
	require.Zero(t, *resp.Experiments[0].BestTrialSearcherMetric) // 0.0 is the best metric.
	require.Nil(t, resp.Experiments[1].BestTrialSearcherMetric)
}

// nolint: exhaustruct
func TestGetExperiments(t *testing.T) {
	// Setup.
	api, _, ctx := setupAPITest(t, nil)

	workResp, err := api.PostWorkspace(ctx, &apiv1.PostWorkspaceRequest{
		Name: uuid.New().String(),
	})
	require.NoError(t, err)
	projResp, err := api.PostProject(ctx, &apiv1.PostProjectRequest{
		WorkspaceId: workResp.Workspace.Id,
		Name:        uuid.New().String(),
	})
	require.NoError(t, err)
	pid := projResp.Project.Id
	_, err = api.ArchiveWorkspace(ctx, &apiv1.ArchiveWorkspaceRequest{
		Id: workResp.Workspace.Id,
	})
	require.NoError(t, err)
	userResp, err := api.PostUser(ctx, &apiv1.PostUserRequest{
		User: &userv1.User{
			Username:    uuid.New().String(),
			DisplayName: uuid.New().String(),
			Active:      true,
		},
	})
	require.NoError(t, err)

	// Create experiments to test with.
	startTime := time.Unix(123123123, int64(1329012309*time.Nanosecond))
	endTime := time.Unix(423123123, int64(999813239*time.Nanosecond))

	require.WithinDuration(t,
		endTime, timestamppb.New(endTime).AsTime(), time.Millisecond)

	job0ID := uuid.New().String()
	activeConfig0 := schemas.Merge(minExpConfig, expconf.ExperimentConfig{
		RawDescription: ptrs.Ptr("12345"),
		RawName:        expconf.Name{RawString: ptrs.Ptr("name")},
		RawLabels:      expconf.Labels{"l0": true, "l1": true},
	})
	activeConfig0 = schemas.WithDefaults(activeConfig0)
	exp0 := &model.Experiment{
		StartTime: startTime,
		EndTime:   &endTime,
		JobID:     model.JobID(job0ID),
		Archived:  false,
		State:     model.PausedState,
		Notes:     "notes",
		Config:    activeConfig0.AsLegacy(),
		OwnerID:   ptrs.Ptr(model.UserID(1)),
		ProjectID: int(pid),
	}
	require.NoError(t, api.m.db.AddExperiment(exp0, []byte{1, 2, 3}, activeConfig0))
	for i := 0; i < 3; i++ {
		task := &model.Task{TaskType: model.TaskTypeTrial, TaskID: model.NewTaskID()}
		require.NoError(t, db.AddTask(ctx, task))
		require.NoError(t, db.AddTrial(ctx, &model.Trial{
			State:        model.PausedState,
			ExperimentID: exp0.ID,
		}, task.TaskID))
	}
	exp0Expected := &experimentv1.Experiment{
		Id:             int32(exp0.ID),
		Description:    *activeConfig0.RawDescription,
		Labels:         []string{"l0", "l1"},
		State:          experimentv1.State_STATE_PAUSED,
		StartTime:      timestamppb.New(startTime),
		EndTime:        timestamppb.New(endTime),
		Duration:       ptrs.Ptr(int32(300000000)),
		Archived:       false,
		NumTrials:      3,
		DisplayName:    "admin",
		UserId:         1,
		Username:       "admin",
		SearcherType:   "single",
		SearcherMetric: "loss",
		Name:           "name",
		Notes:          "omitted", // Notes get omitted when non null.
		JobId:          job0ID,
		Progress:       &wrappers.DoubleValue{Value: 0},
		ProjectName:    projResp.Project.Name,
		WorkspaceId:    workResp.Workspace.Id,
		WorkspaceName:  workResp.Workspace.Name,
		ParentArchived: true,
		ResourcePool:   "kubernetes",
		ProjectId:      pid,
		ProjectOwnerId: projResp.Project.UserId,
	}

	secondStartTime := time.Now()
	job1ID := uuid.New().String()
	activeConfig1 := schemas.Merge(minExpConfig, expconf.ExperimentConfig{
		RawDescription: ptrs.Ptr("234"),
		RawName:        expconf.Name{RawString: ptrs.Ptr("longername")},
		RawLabels:      expconf.Labels{"l0": true},
	})
	activeConfig1 = schemas.WithDefaults(activeConfig1)
	exp1 := &model.Experiment{
		StartTime: secondStartTime,
		JobID:     model.JobID(job1ID),
		Archived:  true,
		State:     model.ErrorState,
		ParentID:  ptrs.Ptr(exp0.ID),
		Config:    activeConfig1.AsLegacy(),
		OwnerID:   ptrs.Ptr(model.UserID(userResp.User.Id)),
		ProjectID: int(pid),
	}
	require.NoError(t, api.m.db.AddExperiment(exp1, []byte{1, 2, 3}, activeConfig1))
	exp1Expected := &experimentv1.Experiment{
		StartTime:      timestamppb.New(secondStartTime),
		Duration:       ptrs.Ptr(int32(0)),
		Id:             int32(exp1.ID),
		Description:    *activeConfig1.RawDescription,
		Labels:         []string{"l0"},
		State:          experimentv1.State_STATE_ERROR,
		Archived:       true,
		NumTrials:      0,
		DisplayName:    userResp.User.DisplayName,
		UserId:         userResp.User.Id,
		Username:       userResp.User.Username,
		SearcherType:   "single",
		SearcherMetric: "loss",
		Name:           "longername",
		JobId:          job1ID,
		ProjectId:      pid,
		Progress:       &wrappers.DoubleValue{Value: 0},
		ForkedFrom:     &wrappers.Int32Value{Value: int32(exp0.ID)},
		ProjectName:    projResp.Project.Name,
		WorkspaceId:    workResp.Workspace.Id,
		WorkspaceName:  workResp.Workspace.Name,
		ParentArchived: true,
		ResourcePool:   "kubernetes",
		ProjectOwnerId: projResp.Project.UserId,
	}

	// Filtering tests.
	testGetExperiments(ctx, t, api, pid, &apiv1.GetExperimentsRequest{}, exp0Expected, exp1Expected)

	testGetExperiments(ctx, t, api, pid,
		&apiv1.GetExperimentsRequest{Description: "12345"}, exp0Expected)
	testGetExperiments(ctx, t, api, pid,
		&apiv1.GetExperimentsRequest{Description: "234"}, exp0Expected, exp1Expected)
	testGetExperiments(ctx, t, api, pid,
		&apiv1.GetExperimentsRequest{Description: "123456"})

	testGetExperiments(ctx, t, api, pid,
		&apiv1.GetExperimentsRequest{Name: "longername"}, exp1Expected)
	testGetExperiments(ctx, t, api, pid,
		&apiv1.GetExperimentsRequest{Name: "name"}, exp0Expected, exp1Expected)
	testGetExperiments(ctx, t, api, pid,
		&apiv1.GetExperimentsRequest{Name: "longlongername"})

	testGetExperiments(ctx, t, api, pid,
		&apiv1.GetExperimentsRequest{Labels: []string{"l0", "l1"}}, exp0Expected)
	testGetExperiments(ctx, t, api, pid,
		&apiv1.GetExperimentsRequest{Labels: []string{"l0"}}, exp0Expected, exp1Expected)
	testGetExperiments(ctx, t, api, pid,
		&apiv1.GetExperimentsRequest{Labels: []string{"l0", "l1", "l3"}})

	testGetExperiments(ctx, t, api, pid,
		&apiv1.GetExperimentsRequest{Archived: wrapperspb.Bool(false)}, exp0Expected)
	testGetExperiments(ctx, t, api, pid,
		&apiv1.GetExperimentsRequest{Archived: wrapperspb.Bool(true)}, exp1Expected)

	testGetExperiments(ctx, t, api, pid,
		&apiv1.GetExperimentsRequest{
			States: []experimentv1.State{experimentv1.State_STATE_PAUSED},
		}, exp0Expected)
	testGetExperiments(ctx, t, api, pid,
		&apiv1.GetExperimentsRequest{
			States: []experimentv1.State{experimentv1.State_STATE_ERROR},
		}, exp1Expected)
	testGetExperiments(ctx, t, api, pid,
		&apiv1.GetExperimentsRequest{
			States: []experimentv1.State{
				experimentv1.State_STATE_PAUSED,
				experimentv1.State_STATE_ERROR,
			},
		}, exp0Expected, exp1Expected)
	testGetExperiments(ctx, t, api, pid,
		&apiv1.GetExperimentsRequest{
			States: []experimentv1.State{experimentv1.State_STATE_CANCELED},
		})

	testGetExperiments(ctx, t, api, pid,
		&apiv1.GetExperimentsRequest{Users: []string{"admin"}}, exp0Expected)
	testGetExperiments(ctx, t, api, pid,
		&apiv1.GetExperimentsRequest{Users: []string{userResp.User.Username}}, exp1Expected)
	testGetExperiments(ctx, t, api, pid,
		&apiv1.GetExperimentsRequest{Users: []string{"admin", userResp.User.Username}},
		exp0Expected, exp1Expected)
	testGetExperiments(ctx, t, api, pid,
		&apiv1.GetExperimentsRequest{Users: []string{"notarealuser"}})

	testGetExperiments(ctx, t, api, pid,
		&apiv1.GetExperimentsRequest{UserIds: []int32{1}}, exp0Expected)
	testGetExperiments(ctx, t, api, pid,
		&apiv1.GetExperimentsRequest{UserIds: []int32{userResp.User.Id}}, exp1Expected)
	testGetExperiments(ctx, t, api, pid,
		&apiv1.GetExperimentsRequest{UserIds: []int32{1, userResp.User.Id}},
		exp0Expected, exp1Expected)
	testGetExperiments(ctx, t, api, pid, &apiv1.GetExperimentsRequest{UserIds: []int32{-999}})

	// Sort and order by tests.
	testGetExperiments(ctx, t, api, pid,
		&apiv1.GetExperimentsRequest{
			SortBy: apiv1.GetExperimentsRequest_SORT_BY_NUM_TRIALS,
		}, exp1Expected, exp0Expected)
	testGetExperiments(ctx, t, api, pid,
		&apiv1.GetExperimentsRequest{
			SortBy:  apiv1.GetExperimentsRequest_SORT_BY_NUM_TRIALS,
			OrderBy: apiv1.OrderBy_ORDER_BY_ASC,
		}, exp1Expected, exp0Expected)
	testGetExperiments(ctx, t, api, pid,
		&apiv1.GetExperimentsRequest{
			SortBy:  apiv1.GetExperimentsRequest_SORT_BY_NUM_TRIALS,
			OrderBy: apiv1.OrderBy_ORDER_BY_DESC,
		}, exp0Expected, exp1Expected)

	// Pagination tests.
	// No experiments should be returned for Limit -2.
	testGetExperiments(ctx, t, api, pid, &apiv1.GetExperimentsRequest{Limit: -2})
	testGetExperimentsPage(ctx, t, api, pid, &apiv1.GetExperimentsRequest{Offset: 1},
		&apiv1.Pagination{Offset: 1, Limit: 0, StartIndex: 1, EndIndex: 2, Total: 2})
	testGetExperimentsPage(ctx, t, api, pid, &apiv1.GetExperimentsRequest{Limit: 1},
		&apiv1.Pagination{Offset: 0, Limit: 1, StartIndex: 0, EndIndex: 1, Total: 2})
	testGetExperimentsPage(ctx, t, api, pid, &apiv1.GetExperimentsRequest{Limit: 1, Offset: 1},
		&apiv1.Pagination{Offset: 1, Limit: 1, StartIndex: 1, EndIndex: 2, Total: 2})
	testGetExperimentsPage(ctx, t, api, pid, &apiv1.GetExperimentsRequest{Offset: 2},
		&apiv1.Pagination{Offset: 2, Limit: 0, StartIndex: 2, EndIndex: 2, Total: 2})

	testGetExperimentsPage(ctx, t, api, pid, &apiv1.GetExperimentsRequest{Limit: -1},
		&apiv1.Pagination{Offset: 0, Limit: -1, StartIndex: 0, EndIndex: 2, Total: 2})
}

func testGetExperimentsPage(ctx context.Context, t *testing.T, api *apiServer, pid int32,
	req *apiv1.GetExperimentsRequest, expected *apiv1.Pagination,
) {
	req.ProjectId = pid
	res, err := api.GetExperiments(ctx, req)
	require.NoError(t, err)
	proto.Equal(expected, res.Pagination)
	require.Equal(t, expected, res.Pagination)
}

func testGetExperiments(ctx context.Context, t *testing.T, api *apiServer, pid int32,
	req *apiv1.GetExperimentsRequest, expected ...*experimentv1.Experiment,
) {
	req.ProjectId = pid
	res, err := api.GetExperiments(ctx, req)
	require.NoError(t, err)
	require.Len(t, res.Experiments, len(expected),
		fmt.Sprintf("wrong length of result set with request %+v", req))

	for i := range expected {
		sort.Strings(expected[i].Labels)
		sort.Strings(res.Experiments[i].Labels)

		// Don't compare config.
		res.Experiments[i].Config = nil //nolint:staticcheck

		// Don't compare hyperparameters.
		res.Experiments[i].Hyperparameters = nil

		// Compare time seperatly due to millisecond precision in postgres.
		require.WithinDuration(t,
			expected[i].StartTime.AsTime(), res.Experiments[i].StartTime.AsTime(), time.Millisecond)
		if expected[i].EndTime == nil {
			require.Equal(t, expected[i].EndTime, res.Experiments[i].EndTime)
		} else {
			require.WithinDuration(t,
				expected[i].EndTime.AsTime(), res.Experiments[i].EndTime.AsTime(), time.Millisecond)
		}

		res.Experiments[i].StartTime = expected[i].StartTime
		res.Experiments[i].EndTime = expected[i].EndTime

		proto.Equal(expected[i], res.Experiments[i]) // Allows require.Equal to compare properly?
		require.Equal(t, expected[i], res.Experiments[i],
			fmt.Sprintf("wrong result request %+v", req))
	}
}

func TestSearchExperiments(t *testing.T) {
	api, curUser, ctx := setupAPITest(t, nil)
	_, projectIDInt := createProjectAndWorkspace(ctx, t, api)
	projectID := int32(projectIDInt)

	// Empty response causes no errors.
	req := &apiv1.SearchExperimentsRequest{
		ProjectId: &projectID,
		Sort:      ptrs.Ptr("id=asc"),
	}
	resp, err := api.SearchExperiments(ctx, req)
	require.NoError(t, err)
	require.Empty(t, resp.Experiments)

	// No trial doesn't cause errors.
	exp := createTestExpWithProjectID(t, api, curUser, projectIDInt)

	resp, err = api.SearchExperiments(ctx, req)
	require.NoError(t, err)
	require.Len(t, resp.Experiments, 1)
	require.Nil(t, resp.Experiments[0].BestTrial)
	require.Equal(t, int32(exp.ID), resp.Experiments[0].Experiment.Id)

	// Trial without validations doesn't cause issues.
	noValidationsExp := createTestExpWithProjectID(t, api, curUser, projectIDInt)
	task := &model.Task{TaskType: model.TaskTypeTrial, TaskID: model.NewTaskID()}
	require.NoError(t, db.AddTask(ctx, task))
	require.NoError(t, db.AddTrial(ctx, &model.Trial{
		State:        model.PausedState,
		ExperimentID: noValidationsExp.ID,
	}, task.TaskID))

	resp, err = api.SearchExperiments(ctx, req)
	require.NoError(t, err)
	require.Len(t, resp.Experiments, 2)

	require.Nil(t, resp.Experiments[0].BestTrial)
	require.Equal(t, int32(exp.ID), resp.Experiments[0].Experiment.Id)

	require.NotNil(t, resp.Experiments[1].BestTrial) // Now has a best trial, since it's the only one.
	require.Equal(t, int32(noValidationsExp.ID), resp.Experiments[1].Experiment.Id)

	// Validations returned properly.
	metricTrial, metrics := createTestTrialWithMetrics(ctx, t, api, curUser, true)
	valMetrics := metrics[model.ValidationMetricGroup]

	// Move experiment to our project.
	_, err = db.Bun().NewUpdate().Table("experiments").
		Set("project_id = ?", projectID).
		Where("id = ?", metricTrial.ExperimentID).
		Exec(ctx)
	require.NoError(t, err)
	// Set restarts super high so it gets reduced to config number.
	_, err = db.Bun().NewUpdate().Table("runs").
		Set("restarts = ?", 31415).
		Where("id = ?", metricTrial.ID).
		Exec(ctx)
	require.NoError(t, err)

	// Sort by external trial ID.
	resp, err = api.SearchExperiments(ctx, &apiv1.SearchExperimentsRequest{
		ProjectId: req.ProjectId,
		Sort:      ptrs.Ptr("externalTrialId=asc"),
	})
	require.NoError(t, err)
	require.Len(t, resp.Experiments, 3)

	resp, err = api.SearchExperiments(ctx, req)
	require.NoError(t, err)
	require.Len(t, resp.Experiments, 3)

	require.Nil(t, resp.Experiments[0].BestTrial)
	require.Equal(t, int32(exp.ID), resp.Experiments[0].Experiment.Id)

	require.NotNil(t, resp.Experiments[1].BestTrial)
	require.Equal(t, int32(noValidationsExp.ID), resp.Experiments[1].Experiment.Id)

	require.NotNil(t, resp.Experiments[2].BestTrial)

	require.Equal(t, int32(9), resp.Experiments[2].BestTrial.TotalBatchesProcessed)

	require.Equal(t, int32(0), resp.Experiments[2].BestTrial.BestValidation.TotalBatches)
	bestActual, err := json.Marshal(resp.Experiments[2].BestTrial.BestValidation.Metrics)
	require.NoError(t, err)
	bestExpected, err := json.Marshal(valMetrics[0])
	require.NoError(t, err)
	require.Equal(t, string(bestExpected), string(bestActual))

	require.Equal(t, int32(9), resp.Experiments[2].BestTrial.LatestValidation.TotalBatches)
	latestActual, err := json.Marshal(resp.Experiments[2].BestTrial.LatestValidation.Metrics)
	require.NoError(t, err)
	latestExpected, err := json.Marshal(valMetrics[len(valMetrics)-1])
	require.NoError(t, err)
	require.Equal(t, string(latestExpected), string(latestActual))

	require.Equal(t, int32(5), resp.Experiments[2].BestTrial.Restarts)

	// when single-trial experiment with no validation metrics completes
	// ensure that the only trial is the best trial.
	_, err = db.Bun().NewUpdate().Table("experiments").
		Set("state = ?", model.CompletedState).
		Where("id = ?", noValidationsExp.ID).
		Exec(ctx)
	require.NoError(t, err)
	resp, err = api.SearchExperiments(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, resp.Experiments[1].BestTrial)
	require.Equal(t, int32(noValidationsExp.ID), resp.Experiments[1].Experiment.Id)
}

func TestSearchExperimentsFilters(t *testing.T) {
	api, curUser, ctx := setupAPITest(t, nil)
	_, projectIDInt := createProjectAndWorkspace(ctx, t, api)
	projectID := int32(projectIDInt)

	// RequireMockExperimentParams expects to be run in the db folder.
	err := os.Chdir("./db")
	require.NoError(t, err)
	paramNames := []string{"foo"}
	db.RequireMockExperimentParams(t, api.m.db, curUser, db.MockExperimentParams{
		HParamNames: &paramNames,
	}, projectIDInt)
	err = os.Chdir("./..")
	require.NoError(t, err)

	tests := map[string]struct {
		expectedNumExperiments int
		filter                 string
	}{
		"ExpHPNotContains": {
			expectedNumExperiments: 0,
			filter: `{"filterGroup":{"children":[{"columnName":"hp.foo","kind":"field",` +
				`"location":"LOCATION_TYPE_HYPERPARAMETERS","operator":"notContains","type":"COLUMN_TYPE_NUMBER","value":1}],` +
				`"conjunction":"and","kind":"group"},"showArchived":false}`,
		},
	}

	for testCase, testVars := range tests {
		t.Run(testCase, func(t *testing.T) {
			resp, requestError := api.SearchExperiments(ctx, &apiv1.SearchExperimentsRequest{
				ProjectId: &projectID,
				Filter:    ptrs.Ptr(testVars.filter),
			})

			require.NoError(t, requestError)
			require.Len(t, resp.Experiments, testVars.expectedNumExperiments)
		})
	}
}

func TestSearchExperimentsMalformed(t *testing.T) {
	api, curUser, ctx := setupAPITest(t, nil)
	_, projectIDInt := createProjectAndWorkspace(ctx, t, api)
	projectID := int32(projectIDInt)

	// No trial doesn't cause errors.
	exp := createTestExpWithProjectID(t, api, curUser, projectIDInt)

	req := &apiv1.SearchExperimentsRequest{
		ProjectId: &projectID,
		Sort:      ptrs.Ptr("wow.it's.mean=asc"),
	}
	resp, err := api.SearchExperiments(ctx, req)
	require.NoError(t, err)
	require.Len(t, resp.Experiments, 1)
	require.Nil(t, resp.Experiments[0].BestTrial)
	require.Equal(t, int32(exp.ID), resp.Experiments[0].Experiment.Id)

	req = &apiv1.SearchExperimentsRequest{
		ProjectId: &projectID,
		Sort:      ptrs.Ptr("hp.tests'=asc"),
	}
	resp, err = api.SearchExperiments(ctx, req)
	require.NoError(t, err)
	require.Len(t, resp.Experiments, 1)
	require.Nil(t, resp.Experiments[0].BestTrial)
	require.Equal(t, int32(exp.ID), resp.Experiments[0].Experiment.Id)
}

func TestSearchExperimentsNoDeleting(t *testing.T) {
	api, curUser, ctx := setupAPITest(t, nil)
	_, projectIDInt := createProjectAndWorkspace(ctx, t, api)

	shownExp := createTestExpWithProjectID(t, api, curUser, projectIDInt)

	// Add deleting experiment
	experimentConfig := expconf.ExperimentConfig{
		RawDescription: ptrs.Ptr("desc"),
		RawName:        expconf.Name{RawString: ptrs.Ptr("name")},
	}

	activeConfig := schemas.WithDefaults(schemas.Merge(minExpConfig, experimentConfig))

	hiddenExp := &model.Experiment{
		JobID:     model.JobID(uuid.New().String()),
		State:     model.DeletingState,
		OwnerID:   &curUser.ID,
		ProjectID: projectIDInt,
		StartTime: time.Now(),
		Config:    activeConfig.AsLegacy(),
	}
	require.NoError(t, api.m.db.AddExperiment(hiddenExp, []byte{10, 11, 12}, activeConfig))

	projectID := int32(projectIDInt)

	// Empty response causes no errors.
	req := &apiv1.SearchExperimentsRequest{
		ProjectId: &projectID,
		Sort:      ptrs.Ptr("id=asc"),
	}

	resp, err := api.SearchExperiments(ctx, req)
	require.NoError(t, err)
	// Deleting experiment should not be in the response
	require.Len(t, resp.Experiments, 1)
	require.Nil(t, resp.Experiments[0].BestTrial)
	require.Equal(t, int32(shownExp.ID), resp.Experiments[0].Experiment.Id)
}

// Test that endpoints don't puke when running against old experiments.
func TestLegacyExperiments(t *testing.T) {
	err := etc.SetRootPath("../static/srv")
	require.NoError(t, err)

	pgDB, cleanup := db.MustResolveNewPostgresDatabase(t)
	defer cleanup()

	prse := olddata.PreRemoveStepsExperiments()
	prse.MustMigrate(t, pgDB, "file://../static/migrations")

	api, _, ctx := setupAPITest(t, pgDB)

	t.Run("GetExperimentCheckpoints", func(t *testing.T) {
		req := &apiv1.GetExperimentCheckpointsRequest{
			Id: prse.CompletedPBTExpID,
			SortBy: &apiv1.GetExperimentCheckpointsRequest_SortByAttr{
				SortByAttr: checkpointv1.SortBy_SORT_BY_SEARCHER_METRIC,
			},
		}
		_, err = api.GetExperimentCheckpoints(ctx, req)
		require.NoError(t, err)
	})

	t.Run("ExpMetricNames", func(t *testing.T) {
		req := &apiv1.ExpMetricNamesRequest{
			Ids: []int32{prse.CompletedPBTExpID},
		}
		err = api.ExpMetricNames(req, &mockStream[*apiv1.ExpMetricNamesResponse]{ctx: ctx})
		require.NoError(t, err)
	})

	t.Run("TrialsSample", func(t *testing.T) {
		req := &apiv1.TrialsSampleRequest{
			ExperimentId: prse.CompletedAdaptiveSimpleExpID,
			MetricName:   "loss",
			MetricType:   apiv1.MetricType_METRIC_TYPE_TRAINING,
		}
		err = api.TrialsSample(req, &mockStream[*apiv1.TrialsSampleResponse]{ctx: ctx})
		require.NoError(t, err)
	})

	t.Run("GetBestSearcherValidationMetric", func(t *testing.T) {
		req := &apiv1.GetBestSearcherValidationMetricRequest{
			ExperimentId: prse.CompletedPBTExpID,
		}
		_, err = api.GetBestSearcherValidationMetric(ctx, req)
		require.NoError(t, err)
	})
}

var res *apiv1.GetExperimentsResponse // Avoid compiler optimizing res out.

// nolint: exhaustruct
func benchmarkGetExperiments(b *testing.B, n int) {
	// This should be fine as long as no error happens. For some
	// reason passing nil gives an error. In addition this
	// benchmark won't run when integration tests run
	// (since it needs the -bench flag) so if this breaks in the
	// future it won't cause any issues.
	api, _, ctx := setupAPITest((*testing.T)(unsafe.Pointer(b)), nil) //nolint: gosec

	// Create n records in the database from the new user we created.
	userResp, err := api.PostUser(ctx, &apiv1.PostUserRequest{
		User: &userv1.User{
			Username:    uuid.New().String(),
			DisplayName: uuid.New().String(),
			Active:      true,
		},
	})
	if err != nil {
		b.Fatal(err)
	}
	defer func() {
		// Delete user and all experiments.
		if _, err := db.Bun().NewDelete().Table("experiments").
			Where("owner_id = ?", userResp.User.Id).Exec(ctx); err != nil {
			b.Fatal(err)
		}
		if _, err := db.Bun().NewDelete().Table("jobs").
			Where("owner_id = ?", userResp.User.Id).Exec(ctx); err != nil {
			b.Fatal(err)
		}
		if _, err := db.Bun().NewDelete().Table("users").
			Where("id = ?", userResp.User.Id).Exec(ctx); err != nil {
			b.Fatal(err)
		}
	}()

	activeConfig := schemas.Merge(minExpConfig, expconf.ExperimentConfig{
		RawDescription: ptrs.Ptr("desc"),
		RawName:        expconf.Name{RawString: ptrs.Ptr("name")},
	})
	activeConfig = schemas.WithDefaults(activeConfig)
	exp := &model.Experiment{
		State:     model.PausedState,
		Config:    activeConfig.AsLegacy(),
		OwnerID:   ptrs.Ptr(model.UserID(userResp.User.Id)),
		ProjectID: 1,
	}
	for i := 0; i < n; i++ {
		jobID := uuid.New().String()
		exp.ID = 0
		exp.JobID = model.JobID(jobID)

		if err := api.m.db.AddExperiment(exp, []byte{1, 2, 3}, activeConfig); err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var err error
		res, err = api.GetExperiments(ctx, &apiv1.GetExperimentsRequest{
			Limit: -1, UserIds: []int32{userResp.User.Id},
		})
		if err != nil {
			b.Fatal(err)
		}
	}
	b.StopTimer()
}

func BenchmarkGetExeriments50(b *testing.B) { benchmarkGetExperiments(b, 50) }

func BenchmarkGetExeriments250(b *testing.B) { benchmarkGetExperiments(b, 250) }

func BenchmarkGetExeriments500(b *testing.B) { benchmarkGetExperiments(b, 500) }

func BenchmarkGetExeriments2500(b *testing.B) { benchmarkGetExperiments(b, 2500) }

func createTestExpWithActiveConfig(
	t *testing.T,
	api *apiServer,
	curUser model.User,
	projectID int,
	activeConfig expconf.ExperimentConfig,
) (exp *model.Experiment) {
	exp = &model.Experiment{
		JobID:     model.JobID(uuid.New().String()),
		State:     model.PausedState,
		OwnerID:   &curUser.ID,
		ProjectID: projectID,
		StartTime: time.Now(),
		Config:    activeConfig.AsLegacy(),
	}
	require.NoError(t, api.m.db.AddExperiment(exp, []byte{10, 11, 12}, activeConfig))

	// Get experiment as our API mostly will to make it easier to mock.
	exp, err := db.ExperimentByID(context.TODO(), exp.ID)
	require.NoError(t, err)
	return exp
}

// nolint: exhaustruct
func createTestExpWithProjectID(
	t *testing.T, api *apiServer, curUser model.User, projectID int, labels ...string,
) *model.Experiment {
	labelMap := make(map[string]bool)
	for _, l := range labels {
		labelMap[l] = true
	}

	experimentConfig := expconf.ExperimentConfig{
		RawLabels:      labelMap,
		RawDescription: ptrs.Ptr("desc"),
		RawName:        expconf.Name{RawString: ptrs.Ptr("name")},
	}

	activeConfig := schemas.WithDefaults(schemas.Merge(minExpConfig, experimentConfig))
	return createTestExpWithActiveConfig(t, api, curUser, projectID, activeConfig)
}

func TestAuthZGetExperiment(t *testing.T) {
	api, authZExp, _, curUser, ctx := setupExpAuthTest(t, nil)
	exp := createTestExp(t, api, curUser)

	mockUserArg := mock.MatchedBy(func(u model.User) bool {
		return u.ID == curUser.ID
	})

	// Not found returns same as permission denied.
	_, err := api.GetExperiment(ctx, &apiv1.GetExperimentRequest{ExperimentId: -999})
	require.Equal(t, apiPkg.NotFoundErrs("experiment", "-999", true).Error(), err.Error())

	authZExp.On("CanGetExperiment", mock.Anything, mockUserArg, mock.Anything).
		Return(authz2.PermissionDeniedError{}).Once()
	_, err = api.GetExperiment(ctx, &apiv1.GetExperimentRequest{ExperimentId: int32(exp.ID)})
	require.Equal(t, apiPkg.NotFoundErrs("experiment", strconv.Itoa(exp.ID), true).Error(),
		err.Error())

	// Error returns error unmodified.
	expectedErr := fmt.Errorf("canGetExperimentError")
	authZExp.On("CanGetExperiment", mock.Anything, mockUserArg, mock.Anything).
		Return(expectedErr).Once()
	_, err = api.GetExperiment(ctx, &apiv1.GetExperimentRequest{ExperimentId: int32(exp.ID)})
	require.Equal(t, expectedErr, err)

	authZExp.On("CanGetExperiment", mock.Anything, mockUserArg, mock.Anything).Return(nil).Once()
	res, err := api.GetExperiment(ctx, &apiv1.GetExperimentRequest{ExperimentId: int32(exp.ID)})
	require.NoError(t, err)
	require.Equal(t, int32(exp.ID), res.Experiment.Id)
}

func TestAuthZGetExperiments(t *testing.T) {
	api, authZExp, authZProject, curUser, ctx := setupExpAuthTest(t, nil)
	_, projectID := createProjectAndWorkspace(ctx, t, api)
	exp0 := createTestExpWithProjectID(t, api, curUser, projectID)
	createTestExpWithProjectID(t, api, curUser, projectID, uuid.New().String())

	mockUserArg := mock.MatchedBy(func(u model.User) bool {
		return u.ID == curUser.ID
	})

	// Can't view project gets a 404.
	authZProject.On("CanGetProject", mock.Anything, mockUserArg,
		mock.Anything).Return(authz2.PermissionDeniedError{}).Once()
	_, err := api.GetExperiments(ctx, &apiv1.GetExperimentsRequest{ProjectId: int32(projectID)})
	require.Equal(t, apiPkg.NotFoundErrs("project", strconv.Itoa(projectID), true).Error(),
		err.Error())

	// Error from FilterExperimentsQuery passes through.
	authZProject.On("CanGetProject", mock.Anything, mockUserArg, mock.Anything).
		Return(nil).Once()
	expectedErr := fmt.Errorf("filterExperimentsQueryError")
	authZExp.On("FilterExperimentsQuery", mock.Anything, mockUserArg, mock.Anything, mock.Anything,
		[]rbacv1.PermissionType{rbacv1.PermissionType_PERMISSION_TYPE_VIEW_EXPERIMENT_METADATA}).
		Return(nil, expectedErr).Once()
	_, err = api.GetExperiments(ctx, &apiv1.GetExperimentsRequest{ProjectId: int32(projectID)})
	require.Equal(t, expectedErr, err)

	// Filter only to only one experiment ID.
	resQuery := &bun.SelectQuery{}
	authZExp.On("FilterExperimentsQuery", mock.Anything, mockUserArg, mock.Anything, mock.Anything,
		mock.Anything).
		Return(resQuery, nil).Once().Run(func(args mock.Arguments) {
		q := args.Get(3).(*bun.SelectQuery)
		*resQuery = *q.Where("e.id = ?", exp0.ID)
	})
	res, err := api.GetExperiments(ctx, &apiv1.GetExperimentsRequest{})
	require.NoError(t, err)
	require.Equal(t, 1, int(res.Pagination.Total))
	require.Len(t, res.Experiments, 1)
	require.Equal(t, exp0.ID, int(res.Experiments[0].Id))
}

func TestAuthZPreviewHPSearch(t *testing.T) {
	api, authZExp, _, curUser, ctx := setupExpAuthTest(t, nil)

	// Can't preview hp search returns error with PermissionDenied
	expectedErr := status.Errorf(codes.PermissionDenied, "canPreviewHPSearchError")
	authZExp.On("CanPreviewHPSearch", mock.Anything, curUser).
		Return(fmt.Errorf("canPreviewHPSearchError")).Once()
	_, err := api.PreviewHPSearch(ctx, &apiv1.PreviewHPSearchRequest{})
	require.Equal(t, expectedErr.Error(), err.Error())
}

func TestAuthZGetExperimentLabels(t *testing.T) {
	api, authZExp, authZProject, curUser, ctx := setupExpAuthTest(t, nil)
	_, projectID := createProjectAndWorkspace(ctx, t, api)
	exp0Label := uuid.New().String()
	exp0 := createTestExpWithProjectID(t, api, curUser, projectID, exp0Label)
	createTestExpWithProjectID(t, api, curUser, projectID, uuid.New().String())

	// Can't view project gets a 404.
	authZProject.On("CanGetProject", mock.Anything, curUser,
		mock.Anything).Return(authz2.PermissionDeniedError{}).Once()
	_, err := api.GetExperimentLabels(ctx, &apiv1.GetExperimentLabelsRequest{
		ProjectId: int32(projectID),
	})
	require.Equal(t, apiPkg.NotFoundErrs("project", strconv.Itoa(projectID), true).Error(),
		err.Error())

	// Error from FilterExperimentsLabelsQuery passes through.
	authZProject.On("CanGetProject", mock.Anything, curUser,
		mock.Anything).Return(nil).Once()
	expectedErr := fmt.Errorf("filterExperimentLabelsQueryError")
	authZExp.On("FilterExperimentLabelsQuery", mock.Anything, curUser, mock.Anything, mock.Anything).
		Return(nil, expectedErr).Once()
	_, err = api.GetExperimentLabels(ctx, &apiv1.GetExperimentLabelsRequest{
		ProjectId: int32(projectID),
	})
	require.Equal(t, expectedErr, err)

	// Filter only to only one experiment ID.
	resQuery := &bun.SelectQuery{}
	authZExp.On("FilterExperimentLabelsQuery", mock.Anything, curUser, mock.Anything, mock.Anything).
		Return(resQuery, nil).Once().Run(func(args mock.Arguments) {
		q := args.Get(3).(*bun.SelectQuery)
		*resQuery = *q.Where("id = ?", exp0.ID)
	})
	res, err := api.GetExperimentLabels(ctx, &apiv1.GetExperimentLabelsRequest{})
	require.NoError(t, err)
	require.Equal(t, []string{exp0Label}, res.Labels)
}

func TestAuthZCreateExperiment(t *testing.T) {
	api, authZExp, _, curUser, ctx := setupExpAuthTest(t, nil)
	forkFrom := createTestExp(t, api, curUser)
	workspaceID, projectID := createProjectAndWorkspace(ctx, t, api)

	// user is logged in again during experiment creation, meaning args don't match
	mockUserArg := mock.MatchedBy(func(u model.User) bool {
		return u.ID == curUser.ID
	})

	t.Run("can't view forked experiment", func(t *testing.T) {
		authZExp.On("CanGetExperiment", mock.Anything, mockUserArg,
			mock.Anything).Return(authz2.PermissionDeniedError{}).Once()
		_, err := api.CreateExperiment(ctx, &apiv1.CreateExperimentRequest{
			ParentId: int32(forkFrom.ID),
		})
		require.Equal(t, apiPkg.NotFoundErrs("experiment",
			strconv.Itoa(forkFrom.ID), true), err)
	})

	t.Run("can't fork from experiment", func(t *testing.T) {
		expectedErr := status.Errorf(codes.PermissionDenied, "canForkExperimentError")
		authZExp.On("CanGetExperiment", mock.Anything, mockUserArg, mock.Anything).Return(nil).Once()
		authZExp.On("CanForkFromExperiment", mock.Anything, mockUserArg, mock.Anything).
			Return(fmt.Errorf("canForkExperimentError")).Once()
		_, err := api.CreateExperiment(ctx, &apiv1.CreateExperimentRequest{
			ParentId: int32(forkFrom.ID),
		})
		require.Equal(t, expectedErr, err)
	})

	t.Run("can't view project passed in", func(t *testing.T) {
		pAuthZ.On("CanGetProject", mock.Anything, mockUserArg,
			mock.Anything).Return(authz2.PermissionDeniedError{}).Once()
		_, err := api.CreateExperiment(ctx, &apiv1.CreateExperimentRequest{
			ProjectId: int32(projectID),
			Config:    minExpConfToYaml(t),
		})
		require.Equal(t, apiPkg.NotFoundErrs("project", strconv.Itoa(projectID), true), err)
	})

	t.Run("can't view project passed in from config", func(t *testing.T) {
		pAuthZ.On("CanGetProject", mock.Anything, mockUserArg,
			mock.Anything).Return(authz2.PermissionDeniedError{}).Once()
		_, err := api.CreateExperiment(ctx, &apiv1.CreateExperimentRequest{
			Config: minExpConfToYaml(t) + "project: Uncategorized\nworkspace: Uncategorized",
		})
		require.Equal(t,
			apiPkg.NotFoundErrs("workspace/project", "Uncategorized/Uncategorized", true), err)
	})

	t.Run("can't view project; workspace does not exist", func(t *testing.T) {
		_, err := api.CreateExperiment(ctx, &apiv1.CreateExperimentRequest{
			Config: minExpConfToYaml(t) + "project: doesntexist123\nworkspace: doesntexist123",
		})
		require.Equal(t,
			apiPkg.NotFoundErrs("workspace/project", "doesntexist123/doesntexist123", true), err)
	})

	t.Run("can't view project; project does not exist", func(t *testing.T) {
		var workspaceName string
		err := db.Bun().NewSelect().
			Table("workspaces").
			Column("name").
			Where("id = ?", workspaceID).
			Scan(ctx, &workspaceName)
		require.NoError(t, err)
		_, err = api.CreateExperiment(ctx, &apiv1.CreateExperimentRequest{
			Config: minExpConfToYaml(t) + "project: doesntexist123\nworkspace: " + workspaceName,
		})
		require.Equal(t,
			apiPkg.NotFoundErrs("workspace/project", workspaceName+"/doesntexist123", true), err)
	})

	t.Run("can't create experiment deny", func(t *testing.T) {
		expectedErr := status.Errorf(codes.PermissionDenied, "canCreateExperimentError")
		pAuthZ.On("CanGetProject", mock.Anything, mockUserArg, mock.Anything).Return(nil).Once()
		authZExp.On("CanCreateExperiment", mock.Anything, mockUserArg, mock.Anything).
			Return(fmt.Errorf("canCreateExperimentError")).Once()
		_, err := api.CreateExperiment(ctx, &apiv1.CreateExperimentRequest{
			ProjectId:       int32(projectID),
			Config:          minExpConfToYaml(t),
			ModelDefinition: []*utilv1.File{{Content: []byte{1, 2, 3}}},
		})
		require.Equal(t, expectedErr, err)
	})

	t.Run("can't activate experiment deny", func(t *testing.T) {
		expectedErr := status.Errorf(codes.PermissionDenied, "canActivateExperimentError")
		pAuthZ.On("CanGetProject", mock.Anything, mockUserArg, mock.Anything).Return(nil).Once()
		authZExp.On("CanCreateExperiment", mock.Anything, mockUserArg, mock.Anything).
			Return(nil).Once()
		authZExp.On("CanEditExperiment", mock.Anything, mockUserArg, mock.Anything, mock.Anything).Return(
			fmt.Errorf("canActivateExperimentError")).Once()
		_, err := api.CreateExperiment(ctx, &apiv1.CreateExperimentRequest{
			Activate:        true,
			Config:          minExpConfToYaml(t),
			ModelDefinition: []*utilv1.File{{Content: []byte{1, 2, 3}}},
		})
		require.Equal(t, expectedErr, err)
	})
}

func TestAuthZGetExperimentAndCanDoActions(t *testing.T) {
	api, authZExp, _, curUser, ctx := setupExpAuthTest(t, nil)
	authZNSC := setupNSCAuthZ()
	workspaceAuthZ := setupWorkspaceAuthZ()
	exp := createTestExp(t, api, curUser)

	// put/patch
	mockUserArg := mock.MatchedBy(func(u model.User) bool {
		return u.ID == curUser.ID
	})

	caseIndividualCalls := []struct {
		DenyFuncName string
		IDToReqCall  func(id int) error
	}{
		{"CanEditExperiment", func(id int) error {
			_, err := api.ActivateExperiment(ctx, &apiv1.ActivateExperimentRequest{
				Id: int32(id),
			})
			return err
		}},
		{"CanEditExperiment", func(id int) error {
			_, err := api.ContinueExperiment(ctx, &apiv1.ContinueExperimentRequest{
				Id: int32(id),
			})
			return err
		}},
		{"CanDeleteExperiment", func(id int) error {
			_, err := api.DeleteExperiment(ctx, &apiv1.DeleteExperimentRequest{
				ExperimentId: int32(id),
			})
			return err
		}},
		{"CanGetExperimentArtifacts", func(id int) error {
			_, err := api.GetExperimentValidationHistory(ctx,
				&apiv1.GetExperimentValidationHistoryRequest{ExperimentId: int32(id)})
			return err
		}},
		{"CanEditExperimentsMetadata", func(id int) error {
			_, err := api.PatchExperiment(ctx, &apiv1.PatchExperimentRequest{
				Experiment: &experimentv1.PatchExperiment{
					Id:   int32(id),
					Name: wrapperspb.String("toname"),
				},
			})
			return err
		}},
		{"CanEditExperimentsMetadata", func(id int) error {
			_, err := api.PatchExperiment(ctx, &apiv1.PatchExperimentRequest{
				Experiment: &experimentv1.PatchExperiment{
					Id:    int32(id),
					Notes: wrapperspb.String("tonotes"),
				},
			})
			return err
		}},
		{"CanEditExperimentsMetadata", func(id int) error {
			_, err := api.PatchExperiment(ctx, &apiv1.PatchExperimentRequest{
				Experiment: &experimentv1.PatchExperiment{
					Id:          int32(id),
					Description: wrapperspb.String("todesc"),
				},
			})
			return err
		}},
		{"CanEditExperimentsMetadata", func(id int) error {
			l, err := structpb.NewList([]any{"l1", "l2"})
			require.NoError(t, err)
			_, err = api.PatchExperiment(ctx, &apiv1.PatchExperimentRequest{
				Experiment: &experimentv1.PatchExperiment{
					Id:     int32(id),
					Labels: l,
				},
			})
			return err
		}},
		{"CanGetExperimentArtifacts", func(id int) error {
			_, err := api.GetExperimentCheckpoints(ctx, &apiv1.GetExperimentCheckpointsRequest{
				Id: int32(id),
			})
			return err
		}},
		{"CanGetExperimentArtifacts", func(id int) error {
			return api.ExpMetricNames(&apiv1.ExpMetricNamesRequest{
				Ids: []int32{int32(id)},
			}, &mockStream[*apiv1.ExpMetricNamesResponse]{ctx: ctx})
		}},
		{"CanGetExperimentArtifacts", func(id int) error {
			return api.MetricBatches(&apiv1.MetricBatchesRequest{
				ExperimentId: int32(id),
				MetricName:   "name",
				MetricType:   apiv1.MetricType_METRIC_TYPE_TRAINING,
			}, &mockStream[*apiv1.MetricBatchesResponse]{ctx: ctx})
		}},
		{"CanGetExperimentArtifacts", func(id int) error {
			return api.TrialsSnapshot(&apiv1.TrialsSnapshotRequest{
				ExperimentId: int32(id),
				MetricName:   "name",
				MetricType:   apiv1.MetricType_METRIC_TYPE_TRAINING,
			}, &mockStream[*apiv1.TrialsSnapshotResponse]{ctx: ctx})
		}},
		{"CanGetExperimentArtifacts", func(id int) error {
			return api.TrialsSample(&apiv1.TrialsSampleRequest{
				ExperimentId: int32(id),
				MetricName:   "name",
				MetricType:   apiv1.MetricType_METRIC_TYPE_TRAINING,
			}, &mockStream[*apiv1.TrialsSampleResponse]{ctx: ctx})
		}},
		{"CanGetExperimentArtifacts", func(id int) error {
			_, err := api.GetBestSearcherValidationMetric(ctx,
				&apiv1.GetBestSearcherValidationMetricRequest{ExperimentId: int32(id)})
			return err
		}},
		{"CanGetExperimentArtifacts", func(id int) error {
			_, err := api.GetModelDef(ctx, &apiv1.GetModelDefRequest{
				ExperimentId: int32(id),
			})
			return err
		}},
		{"CanGetExperimentArtifacts", func(id int) error {
			_, err := api.GetModelDefTree(ctx, &apiv1.GetModelDefTreeRequest{
				ExperimentId: int32(id),
			})
			return err
		}},
		{"CanGetExperimentArtifacts", func(id int) error {
			_, err := api.GetModelDefFile(ctx, &apiv1.GetModelDefFileRequest{
				ExperimentId: int32(id),
			})
			return err
		}},
		{"CanGetExperimentArtifacts", func(id int) error {
			_, err := api.GetExperimentTrials(ctx, &apiv1.GetExperimentTrialsRequest{
				ExperimentId: int32(id),
			})
			return err
		}},
		{"CanGetExperimentArtifacts", func(id int) error {
			authZNSC.On("CanGetTensorboard", mock.Anything, mockUserArg, mock.Anything, mock.Anything,
				mock.Anything).Return(nil).Once()
			workspaceAuthZ.On("CanGetWorkspace", mock.Anything, mock.Anything, mock.Anything).
				Return(nil).Once()
			mockRM := MockRM()
			mockRM.On("SmallerValueIsHigherPriority", mock.Anything).Return(true, nil)
			api.m.rm = mockRM
			_, err := api.LaunchTensorboard(ctx, &apiv1.LaunchTensorboardRequest{
				ExperimentIds: []int32{int32(id)},
			})
			return err
		}},
	}

	for _, curCase := range caseIndividualCalls {
		// Not found returns same as permission denied.
		require.Equal(t, apiPkg.NotFoundErrs("experiment", "-999", true), curCase.IDToReqCall(-999))

		authZExp.On("CanGetExperiment", mock.Anything, mockUserArg, mock.Anything).
			Return(authz2.PermissionDeniedError{}).Once()
		require.Equal(t, apiPkg.NotFoundErrs("experiment", strconv.Itoa(exp.ID), true),
			curCase.IDToReqCall(exp.ID))

		// CanGetExperiment error returns unmodified.
		expectedErr := fmt.Errorf("canGetExperimentError")
		authZExp.On("CanGetExperiment", mock.Anything, mockUserArg, mock.Anything).
			Return(expectedErr).Once()
		require.Equal(t, expectedErr, curCase.IDToReqCall(exp.ID))

		// Deny returns error with PermissionDenied.
		expectedErr = status.Errorf(codes.PermissionDenied, curCase.DenyFuncName+"Error")
		authZExp.On("CanGetExperiment", mock.Anything, mockUserArg, mock.Anything).
			Return(nil).Once()
		authZExp.On(curCase.DenyFuncName, mock.Anything, mockUserArg, mock.Anything).
			Return(fmt.Errorf(curCase.DenyFuncName + "Error")).Once()
		require.Equal(t, expectedErr.Error(), curCase.IDToReqCall(exp.ID).Error())
	}

	resQuery := &bun.SelectQuery{}
	caseIndividualThroughBulkCalls := []struct {
		DenyFuncName string
		IDToReqCall  func(id int) error
	}{
		{"CanEditExperiment", func(id int) error {
			_, err := api.PauseExperiment(ctx, &apiv1.PauseExperimentRequest{
				Id: int32(id),
			})
			return err
		}},
		{"CanEditExperiment", func(id int) error {
			_, err := api.CancelExperiment(ctx, &apiv1.CancelExperimentRequest{
				Id: int32(id),
			})
			return err
		}},
		{"CanEditExperiment", func(id int) error {
			_, err := api.KillExperiment(ctx, &apiv1.KillExperimentRequest{
				Id: int32(id),
			})
			return err
		}},
		{"CanEditExperimentsMetadata", func(id int) error {
			_, err := api.ArchiveExperiment(ctx, &apiv1.ArchiveExperimentRequest{
				Id: int32(id),
			})
			return err
		}},
		{"CanEditExperimentsMetadata", func(id int) error {
			_, err := api.UnarchiveExperiment(ctx, &apiv1.UnarchiveExperimentRequest{
				Id: int32(id),
			})
			return err
		}},
	}
	for _, curCase := range caseIndividualThroughBulkCalls {
		// Return no results, get not-found error.
		// permissions could be UpdateExperiment or UpdateExperimentMetadata.
		authZExp.On("FilterExperimentsQuery", mock.Anything, mock.Anything, mock.Anything, mock.Anything,
			mock.Anything).
			Return(resQuery, nil).Once().Run(func(args mock.Arguments) {
			q := args.Get(3).(*bun.SelectQuery).Where("0 = 1")
			*resQuery = *q
		})
		require.Equal(t, apiPkg.NotFoundErrs("experiment", strconv.Itoa(exp.ID), true),
			curCase.IDToReqCall(exp.ID))

		// FilterExperimentsQuery error returned unmodified.
		expectedErr := fmt.Errorf("canGetExperimentError")
		authZExp.On("FilterExperimentsQuery", mock.Anything, mock.Anything, mock.Anything, mock.Anything,
			mock.Anything).
			Return(resQuery, expectedErr).Once().Run(func(args mock.Arguments) {
			q := args.Get(3).(*bun.SelectQuery).Where("0 = 1")
			*resQuery = *q
		})
		require.Equal(t, expectedErr, curCase.IDToReqCall(exp.ID))
	}

	caseBulkCalls := []struct {
		DenyFuncName string
		IDToReqCall  func(id int) ([]*apiv1.ExperimentActionResult, error)
	}{
		{"CanEditExperiment", func(id int) ([]*apiv1.ExperimentActionResult, error) {
			res, err := api.ActivateExperiments(ctx, &apiv1.ActivateExperimentsRequest{
				ProjectId:     1,
				ExperimentIds: []int32{int32(id)},
			})
			if err != nil {
				return nil, err
			}
			return res.Results, err
		}},
		{"CanEditExperiment", func(id int) ([]*apiv1.ExperimentActionResult, error) {
			res, err := api.PauseExperiments(ctx, &apiv1.PauseExperimentsRequest{
				ProjectId:     1,
				ExperimentIds: []int32{int32(id)},
			})
			if err != nil {
				return nil, err
			}
			return res.Results, err
		}},
		{"CanEditExperiment", func(id int) ([]*apiv1.ExperimentActionResult, error) {
			res, err := api.CancelExperiments(ctx, &apiv1.CancelExperimentsRequest{
				ProjectId:     1,
				ExperimentIds: []int32{int32(id)},
			})
			if err != nil {
				return nil, err
			}
			return res.Results, err
		}},
		{"CanEditExperiment", func(id int) ([]*apiv1.ExperimentActionResult, error) {
			res, err := api.KillExperiments(ctx, &apiv1.KillExperimentsRequest{
				ProjectId:     1,
				ExperimentIds: []int32{int32(id)},
			})
			if err != nil {
				return nil, err
			}
			return res.Results, err
		}},
		{"CanEditExperimentsMetadata", func(id int) ([]*apiv1.ExperimentActionResult, error) {
			res, err := api.ArchiveExperiments(ctx, &apiv1.ArchiveExperimentsRequest{
				ProjectId:     1,
				ExperimentIds: []int32{int32(id)},
			})
			if err != nil {
				return nil, err
			}
			return res.Results, err
		}},
		{"CanEditExperimentsMetadata", func(id int) ([]*apiv1.ExperimentActionResult, error) {
			res, err := api.UnarchiveExperiments(ctx, &apiv1.UnarchiveExperimentsRequest{
				ProjectId:     1,
				ExperimentIds: []int32{int32(id)},
			})
			if err != nil {
				return nil, err
			}
			return res.Results, err
		}},
	}
	for _, curCase := range caseBulkCalls {
		// Return no results, get not-found error in results.
		// permissions could be UpdateExperiment or UpdateExperimentMetadata.
		authZExp.On("FilterExperimentsQuery", mock.Anything, mock.Anything, mock.Anything, mock.Anything,
			mock.Anything).
			Return(resQuery, nil).Once().Run(func(args mock.Arguments) {
			q := args.Get(3).(*bun.SelectQuery).Where("0 = 1")
			*resQuery = *q
		})
		results, _ := curCase.IDToReqCall(exp.ID)
		require.Equal(t, apiPkg.NotFoundErrs("experiment",
			strconv.Itoa(exp.ID), true).Error(), results[0].Error)

		// FilterExperimentsQuery error returned unmodified.
		expectedErr := fmt.Errorf("canGetExperimentError")
		authZExp.On("FilterExperimentsQuery", mock.Anything, mock.Anything, mock.Anything, mock.Anything,
			mock.Anything).
			Return(resQuery, expectedErr).Once().Run(func(args mock.Arguments) {
			q := args.Get(3).(*bun.SelectQuery).Where("0 = 1")
			*resQuery = *q
		})
		_, apiPkg := curCase.IDToReqCall(exp.ID)
		require.Equal(t, expectedErr, apiPkg)
	}
}

// nolint: lll
func TestExperimentSearchApiFilterParsing(t *testing.T) {
	setupAPITest(t, nil)
	invalidTestCases := []string{
		// No operator specified in field
		`{"filterGroup":{"children":[{"columnName":"resourcePool","kind":"field","value":"default"}],"conjunction":"and","kind":"group"},"showArchived":false}`,

		// No conjunction in group
		`{"filterGroup":{"children":[{"columnName":"resourcePool","kind":"field","operator":"=","value":"default"}],"kind":"group"},"showArchived":false}`,

		// invalid group conjunction
		`{"filterGroup":{"children":[{"columnName":"resourcePool","kind":"field","operator":"=","value":"default"}],"conjunction":"invalid","kind":"group"},"showArchived":false}`,

		// invalid operator
		`{"filterGroup":{"children":[{"columnName":"resourcePool","kind":"field","operator":"invalid","value":"default"}],"conjunction":"and","kind":"group"},"showArchived":false}`,

		//  Invalid experiment field
		`{"filterGroup":{"children":[{"location":"LOCATION_TYPE_EXPERIMENT","columnName":"notValid","kind":"field","value":"default"}],"conjunction":"and","kind":"group"},"showArchived":false}`,
	}
	for _, c := range invalidTestCases {
		q := db.Bun().NewSelect()
		var efr experimentFilterRoot
		err := json.Unmarshal([]byte(c), &efr)
		require.NoError(t, err)
		_, err = efr.toSQL(q)
		require.Error(t, err)
	}
	validTestCases := [][2]string{
		{`{"filterGroup":{"children":[{"columnName":"id","kind":"field","operator":"=","value":1}],"conjunction":"and","kind":"group"},"showArchived":false}`, `(((e.id = 1))) AND ((e.archived = false))`},
		{`{"filterGroup":{"children":[{"children":[{"columnName":"id","kind":"field","operator":"=","value":1},{"columnName":"id","kind":"field","operator":"=","value":2}],"conjunction":"and","kind":"group"},{"children":[{"columnName":"id","kind":"field","operator":"=","value":3},{"columnName":"id","kind":"field","operator":"=","value":4}],"conjunction":"and","kind":"group"},{"children":[{"columnName":"id","kind":"field","operator":"=","value":5},{"columnName":"id","kind":"field","operator":"=","value":6}],"conjunction":"and","kind":"group"}],"conjunction":"or","kind":"group"},"showArchived":false}`, `((((e.id = 1)) AND ((e.id = 2))) OR (((e.id = 3)) AND ((e.id = 4))) OR (((e.id = 5)) AND ((e.id = 6)))) AND ((e.archived = false))`},
		{`{"filterGroup":{"children":[{"columnName":"id","kind":"field","operator":"=","value":1},{"columnName":"id","kind":"field","operator":"=","value":2}],"conjunction":"and","kind":"group"},"showArchived":false}`, `(((e.id = 1)) AND ((e.id = 2))) AND ((e.archived = false))`},
		{`{"filterGroup":{"children":[{"columnName":"id","kind":"field","operator":"=","value":1},{"columnName":"id","kind":"field","operator":"=","value":2}],"conjunction":"or","kind":"group"},"showArchived":false}`, `(((e.id = 1)) OR ((e.id = 2))) AND ((e.archived = false))`},
		{`{"filterGroup":{"children":[{"columnName":"id","kind":"field","operator":"=","value":1},{"columnName":"id","kind":"field","operator":"=","value":2},{"children":[{"columnName":"id","kind":"field","operator":"=","value":3},{"columnName":"id","kind":"field","operator":"=","value":4}],"conjunction":"and","kind":"group"},{"children":[{"columnName":"id","kind":"field","operator":"=","value":5},{"children":[{"columnName":"id","kind":"field","operator":"=","value":6},{"columnName":"id","kind":"field","operator":"=","value":7}],"conjunction":"and","kind":"group"}],"conjunction":"or","kind":"group"}],"conjunction":"or","kind":"group"},"showArchived":false}`, `(((e.id = 1)) OR ((e.id = 2)) OR (((e.id = 3)) AND ((e.id = 4))) OR (((e.id = 5)) OR (((e.id = 6)) AND ((e.id = 7))))) AND ((e.archived = false))`},
		{`{"filterGroup":{"children":[{"children":[{"columnName":"id","kind":"field","operator":"=","value":1},{"columnName":"id","kind":"field","operator":"=","value":2}],"conjunction":"and","kind":"group"},{"children":[{"columnName":"id","kind":"field","operator":"=","value":3},{"columnName":"id","kind":"field","operator":"=","value":4}],"conjunction":"and","kind":"group"},{"children":[{"columnName":"id","kind":"field","operator":"=","value":5},{"columnName":"id","kind":"field","operator":"=","value":6}],"conjunction":"and","kind":"group"}],"conjunction":"or","kind":"group"},"showArchived":false}`, `((((e.id = 1)) AND ((e.id = 2))) OR (((e.id = 3)) AND ((e.id = 4))) OR (((e.id = 5)) AND ((e.id = 6)))) AND ((e.archived = false))`},
		{`{"filterGroup":{"children":[{"children":[],"conjunction":"and","kind":"group"},{"children":[],"conjunction":"and","kind":"group"},{"children":[],"conjunction":"and","kind":"group"}],"conjunction":"or","kind":"group"},"showArchived":false}`, `(((true)) OR ((true)) OR ((true))) AND ((e.archived = false))`},
		{`{"filterGroup":{"children":[{"children":[],"conjunction":"and","kind":"group"},{"children":[],"conjunction":"and","kind":"group"},{"children":[{"columnName":"description","kind":"field","operator":"notEmpty","value":null}],"conjunction":"and","kind":"group"}],"conjunction":"and","kind":"group"},"showArchived":true}`, `(((true)) AND ((true)) AND (((e.config->>'description' IS NOT NULL AND e.config->>'description' != '' AND e.config->>'description' != '[]'))))`},
		{`{"filterGroup":{"children":[{"columnName":"numTrials","kind":"field","operator":">","value":0},{"columnName":"id","kind":"field","operator":"!=","value":0},{"columnName":"forkedFrom","kind":"field","operator":"!=","value":1}],"conjunction":"and","kind":"group"},"showArchived":true}`, `((((SELECT COUNT(*) FROM trials t WHERE e.id = t.experiment_id) > 0)) AND ((e.id != 0)) AND ((e.parent_id != 1)))`},
		{`{"filterGroup":{"children":[{"columnName":"description","kind":"field","operator":"contains","value":"t\\set"}],"conjunction":"and","kind":"group"},"showArchived":false}`, `(((e.config->>'description' ILIKE '%t\set%'))) AND ((e.archived = false))`},
		{`{"filterGroup":{"children":[{"columnName":"resourcePool","kind":"field","operator":"contains","value":"default"}],"conjunction":"and","kind":"group"},"showArchived":true}`, `(((e.config->'resources'->>'resource_pool' ILIKE '%default%')))`},
		{`{"filterGroup":{"children":[{"columnName":"id","kind":"field","operator":"=","value":1}],"conjunction":"and","kind":"group"},"showArchived":true}`, `(((e.id = 1)))`},
		{`{"filterGroup":{"children":[{"type":"COLUMN_TYPE_DATE","location":"LOCATION_TYPE_EXPERIMENT", "columnName":"startTime","kind":"field","operator":">", "value":"2021-04-14T14:14:18.915483952Z"}],"conjunction":"and","kind":"group"},"showArchived":false}`, `(((e.start_time > '2021-04-14T14:14:18.915483952Z'))) AND ((e.archived = false))`},
		{`{"filterGroup":{"children":[{"type":"COLUMN_TYPE_DATE","location":"LOCATION_TYPE_EXPERIMENT", "columnName":"endTime","kind":"field","operator":"<=", "value":"2021-04-14T14:14:18.915483952Z"}],"conjunction":"and","kind":"group"},"showArchived":false}`, `(((e.end_time <= '2021-04-14T14:14:18.915483952Z'))) AND ((e.archived = false))`},
		{`{"filterGroup":{"children":[{"type":"COLUMN_TYPE_TEXT","location":"LOCATION_TYPE_EXPERIMENT", "columnName":"tags","kind":"field","operator":"contains", "value":"val"}],"conjunction":"and","kind":"group"},"showArchived":true}`, `(((e.config->>'labels' ILIKE '%val%')))`},
		{`{"filterGroup":{"children":[{"type":"COLUMN_TYPE_TEXT","location":"LOCATION_TYPE_EXPERIMENT", "columnName":"tags","kind":"field","operator":"notContains", "value":"val"}],"conjunction":"and","kind":"group"},"showArchived":true}`, `(((e.config->>'labels' NOT ILIKE '%val%')))`},
		{`{"filterGroup":{"children":[{"type":"COLUMN_TYPE_NUMBER","location":"LOCATION_TYPE_EXPERIMENT", "columnName":"duration","kind":"field","operator":">", "value":0}],"conjunction":"and","kind":"group"},"showArchived":true}`, `(((extract(epoch FROM coalesce(e.end_time, now()) - e.start_time) > 0)))`},
		{`{"filterGroup":{"children":[{"columnName":"projectId","location":"LOCATION_TYPE_EXPERIMENT", "kind":"field","operator":">=","value":-1}],"conjunction":"and","kind":"group"},"showArchived":true}`, `(((project_id >= -1)))`},
		{`{"filterGroup":{"children":[{"type":"COLUMN_TYPE_NUMBER","location":"LOCATION_TYPE_VALIDATIONS", "columnName":"validation.validation_accuracy.mean","kind":"field","operator":">=","value":0}],"conjunction":"and","kind":"group"},"showArchived":true}`, `((((r.summary_metrics->'validation_metrics'->'validation_accuracy'->>'mean')::float8 >= 0)))`},
		{`{"filterGroup":{"children":[{"type":"COLUMN_TYPE_TEXT","location":"LOCATION_TYPE_VALIDATIONS", "columnName":"validation.validation_string.min","kind":"field","operator":"=","value":"string"}],"conjunction":"and","kind":"group"},"showArchived":true}`, `(((r.summary_metrics->'validation_metrics'->'validation_string'->>'min' = 'string')))`},
		{`{"filterGroup":{"children":[{"type":"COLUMN_TYPE_TEXT","location":"LOCATION_TYPE_VALIDATIONS", "columnName":"validation.validation_string.max","kind":"field","operator":"!=","value":"string"}],"conjunction":"and","kind":"group"},"showArchived":true}`, `(((r.summary_metrics->'validation_metrics'->'validation_string'->>'max' != 'string')))`},
		{`{"filterGroup":{"children":[{"type":"COLUMN_TYPE_TEXT","location":"LOCATION_TYPE_VALIDATIONS", "columnName":"validation.validation_string.mean","kind":"field","operator":"contains","value":"string"}],"conjunction":"and","kind":"group"},"showArchived":true}`, `(((r.summary_metrics->'validation_metrics'->'validation_string'->>'mean' LIKE '%string%')))`},
		{`{"filterGroup":{"children":[{"type":"COLUMN_TYPE_TEXT","location":"LOCATION_TYPE_VALIDATIONS", "columnName":"validation.validation_string.min","kind":"field","operator":"notContains","value":"string"}],"conjunction":"and","kind":"group"},"showArchived":true}`, `(((r.summary_metrics->'validation_metrics'->'validation_string'->>'min' NOT LIKE '%string%')))`},
		{`{"filterGroup":{"children":[{"type":"COLUMN_TYPE_NUMBER","location":"LOCATION_TYPE_VALIDATIONS", "columnName":"validation.validation_error.min","kind":"field","operator":">=","value":0}],"conjunction":"and","kind":"group"},"showArchived":true}`, `((((r.summary_metrics->'validation_metrics'->'validation_error'->>'min')::float8 >= 0)))`},
		{`{"filterGroup":{"children":[{"type":"COLUMN_TYPE_NUMBER","location":"LOCATION_TYPE_VALIDATIONS", "columnName":"validation.validation_error.max","kind":"field","operator":"notEmpty","value":0}],"conjunction":"and","kind":"group"},"showArchived":true}`, `((((r.summary_metrics->'validation_metrics'->'validation_error'->>'max')::float8 IS NOT NULL)))`},
		{`{"filterGroup":{"children":[{"type":"COLUMN_TYPE_NUMBER","location":"LOCATION_TYPE_VALIDATIONS", "columnName":"validation.validation_error.max","kind":"field","operator":"isEmpty","value":0}],"conjunction":"and","kind":"group"},"showArchived":true}`, `((((r.summary_metrics->'validation_metrics'->'validation_error'->>'max')::float8 IS NULL)))`},
		{`{"filterGroup":{"children":[{"type":"COLUMN_TYPE_NUMBER","location":"LOCATION_TYPE_VALIDATIONS", "columnName":"validation.x.max","kind":"field","operator":"=","value": 0}],"conjunction":"and","kind":"group"},"showArchived":true}`, `((((r.summary_metrics->'validation_metrics'->'x'->>'max')::float8 = 0)))`},
		{`{"filterGroup":{"children":[{"type":"COLUMN_TYPE_NUMBER","location":"LOCATION_TYPE_VALIDATIONS", "columnName":"validation.loss.last","kind":"field","operator":"!=","value":0.004}],"conjunction":"and","kind":"group"},"showArchived":true}`, `((((r.summary_metrics->'validation_metrics'->'loss'->>'last')::float8 != 0.004)))`},
		{`{"filterGroup":{"children":[{"type":"COLUMN_TYPE_NUMBER","location":"LOCATION_TYPE_VALIDATIONS", "columnName":"validation.validation_accuracy.max","kind":"field","operator":"<","value":-3}],"conjunction":"and","kind":"group"},"showArchived":true}`, `((((r.summary_metrics->'validation_metrics'->'validation_accuracy'->>'max')::float8 < -3)))`},
		{`{"filterGroup":{"children":[{"type":"COLUMN_TYPE_NUMBER","location":"LOCATION_TYPE_VALIDATIONS", "columnName":"validation.validation_accuracy.min","kind":"field","operator":"<=","value":10}],"conjunction":"and","kind":"group"},"showArchived":true}`, `((((r.summary_metrics->'validation_metrics'->'validation_accuracy'->>'min')::float8 <= 10)))`},
		{`{"filterGroup":{"children":[{"columnName":"projectId","kind":"field","operator":">=","value":null}],"conjunction":"and","kind":"group"},"showArchived":true}`, `(((true)))`},
		{`{"filterGroup":{"children":[{"columnName":"id","kind":"field","operator":"=","value":1},{"children":[{"columnName":"id","kind":"field","operator":"=","value":2},{"columnName":"id","kind":"field","operator":"=","value":3}],"conjunction":"and","kind":"group"},{"columnName":"id","kind":"field","operator":"=","value":4},{"children":[{"columnName":"id","kind":"field","operator":"=","value":5}],"conjunction":"and","kind":"group"}],"conjunction":"and","kind":"group"},"showArchived":true}`, `(((e.id = 1)) AND (((e.id = 2)) AND ((e.id = 3))) AND ((e.id = 4)) AND (((e.id = 5))))`},
		{`{"filterGroup":{"children":[{"children":[{"columnName":"checkpointCount","kind":"field","operator":"=","value":4},{"columnName":"numTrials","kind":"field","operator":"=","value":1},{"columnName":"progress","kind":"field","operator":"=","value":100}],"conjunction":"and","kind":"group"}],"conjunction":"and","kind":"group"},"showArchived":true}`, `((((e.checkpoint_count = 4)) AND (((SELECT COUNT(*) FROM trials t WHERE e.id = t.experiment_id) = 1)) AND ((ROUND(COALESCE(progress, 0) * 100)::INTEGER = 100))))`},
		{
			`{"filterGroup":{"children":[{"type":"COLUMN_TYPE_NUMBER","location":"LOCATION_TYPE_HYPERPARAMETERS", "columnName":"hp.global_batch_size","kind":"field","operator":"=","value":32}],"conjunction":"and","kind":"group"},"showArchived":true}`,
			`((((CASE
				WHEN config->'hyperparameters'->'global_batch_size'->>'type' = 'const' THEN (config->'hyperparameters'->'global_batch_size'->>'val')::float8 = 32
				WHEN config->'hyperparameters'->'global_batch_size'->>'type' IN ('int', 'double', 'log') THEN ((config->'hyperparameters'->'global_batch_size'->>'minval')::float8 = 32 OR (config->'hyperparameters'->'global_batch_size'->>'maxval')::float8 = 32)
				ELSE false
			 END))))`,
		},
		{
			`{"filterGroup":{"children":[{"type":"COLUMN_TYPE_NUMBER","location":"LOCATION_TYPE_HYPERPARAMETERS", "columnName":"hp.global_batch_size","kind":"field","operator":">=","value":32}],"conjunction":"and","kind":"group"},"showArchived":true}`,
			`((((CASE
				WHEN config->'hyperparameters'->'global_batch_size'->>'type' = 'const' THEN (config->'hyperparameters'->'global_batch_size'->>'val')::float8 >= 32
				WHEN config->'hyperparameters'->'global_batch_size'->>'type' IN ('int', 'double', 'log') THEN ((config->'hyperparameters'->'global_batch_size'->>'minval')::float8 >= 32 OR (config->'hyperparameters'->'global_batch_size'->>'maxval')::float8 >= 32)
				ELSE false
			 END))))`,
		},
		{
			`{"filterGroup":{"children":[{"type":"COLUMN_TYPE_DATE","location":"LOCATION_TYPE_HYPERPARAMETERS", "columnName":"hp.global_batch_start","kind":"field","operator":">=","value":"2021-04-14T14:14:18.915483952Z"}],"conjunction":"and","kind":"group"},"showArchived":true}`,
			`((((CASE WHEN config->'hyperparameters'->'global_batch_start'->>'type' = 'const' THEN config->'hyperparameters'->'global_batch_start'->>'val' >= '2021-04-14T14:14:18.915483952Z' ELSE false END))))`,
		},
		{
			`{"filterGroup":{"children":[{"type":"COLUMN_TYPE_NUMBER","location":"LOCATION_TYPE_HYPERPARAMETERS", "columnName":"hp.global_batch_size","kind":"field","operator":"!=","value":32}],"conjunction":"and","kind":"group"},"showArchived":true}`,
			`((((CASE
				WHEN config->'hyperparameters'->'global_batch_size'->>'type' = 'const' THEN (config->'hyperparameters'->'global_batch_size'->>'val')::float8 != 32
				WHEN config->'hyperparameters'->'global_batch_size'->>'type' IN ('int', 'double', 'log') THEN ((config->'hyperparameters'->'global_batch_size'->>'minval')::float8 != 32 OR (config->'hyperparameters'->'global_batch_size'->>'maxval')::float8 != 32)
				ELSE false
			 END))))`,
		},
		{
			`{"filterGroup":{"children":[{"type":"COLUMN_TYPE_NUMBER","location":"LOCATION_TYPE_HYPERPARAMETERS", "columnName":"hp.global_batch_size","kind":"field","operator":"isEmpty"}],"conjunction":"and","kind":"group"},"showArchived":true}`,
			`((((CASE
				WHEN config->'hyperparameters'->'global_batch_size'->>'type' = 'const' THEN (config->'hyperparameters'->'global_batch_size'->>'val')::float8 IS NULL
				WHEN config->'hyperparameters'->'global_batch_size'->>'type' = 'categorical' THEN config->'hyperparameters'->'global_batch_size'->>'vals' IS NULL
				WHEN config->'hyperparameters'->'global_batch_size'->>'type' IN ('int', 'double', 'log') THEN (config->'hyperparameters'->'global_batch_size') IS NULL
				ELSE false
			 END))))`,
		},
		{
			`{"filterGroup":{"children":[{"type":"COLUMN_TYPE_NUMBER","location":"LOCATION_TYPE_HYPERPARAMETERS", "columnName":"hp.global_batch_size","kind":"field","operator":"notEmpty"}],"conjunction":"and","kind":"group"},"showArchived":true}`,
			`((((CASE
				WHEN config->'hyperparameters'->'global_batch_size'->>'type' = 'const' THEN (config->'hyperparameters'->'global_batch_size'->>'val')::float8 IS NOT NULL
				WHEN config->'hyperparameters'->'global_batch_size'->>'type' = 'categorical' THEN config->'hyperparameters'->'global_batch_size'->>'vals' IS NOT NULL
				WHEN config->'hyperparameters'->'global_batch_size'->>'type' IN ('int', 'double', 'log') THEN (config->'hyperparameters'->'global_batch_size') IS NOT NULL
				ELSE false
			 END))))`,
		},
		{`{"filterGroup":{"children":[{"type":"COLUMN_TYPE_TEXT","location":"LOCATION_TYPE_HYPERPARAMETERS", "columnName":"hp.model","kind":"field","operator":"=","value":"efficientdet_d0"}],"conjunction":"and","kind":"group"},"showArchived":true}`, `((((CASE WHEN config->'hyperparameters'->'model'->>'type' = 'const' THEN config->'hyperparameters'->'model'->>'val' = 'efficientdet_d0' ELSE false END))))`},
		{`{"filterGroup":{"children":[{"type":"COLUMN_TYPE_TEXT","location":"LOCATION_TYPE_HYPERPARAMETERS", "columnName":"hp.model","kind":"field","operator":"=","value":"efficientdet_d0"}],"conjunction":"and","kind":"group"},"showArchived":false}`, `((((CASE WHEN config->'hyperparameters'->'model'->>'type' = 'const' THEN config->'hyperparameters'->'model'->>'val' = 'efficientdet_d0' ELSE false END)))) AND ((e.archived = false))`},
		{`{"filterGroup":{"children":[{"type":"COLUMN_TYPE_TEXT","location":"LOCATION_TYPE_HYPERPARAMETERS", "columnName":"hp.model","kind":"field","operator":"isEmpty"}],"conjunction":"and","kind":"group"},"showArchived":true}`, `((((CASE
				WHEN config->'hyperparameters'->'model'->>'type' = 'const' THEN config->'hyperparameters'->'model'->>'val' IS NULL
				WHEN config->'hyperparameters'->'model'->>'type' = 'categorical' THEN config->'hyperparameters'->'model'->>'vals' IS NULL
				ELSE false
			 END))))`},
		{
			`{"filterGroup":{"children":[{"type":"COLUMN_TYPE_NUMBER","location":"LOCATION_TYPE_HYPERPARAMETERS", "columnName":"hp.clip_grad","kind":"field","operator":"contains", "value":8}],"conjunction":"and","kind":"group"},"showArchived":true}`,
			`((((CASE
					WHEN config->'hyperparameters'->'clip_grad'->>'type' = 'categorical' THEN (config->'hyperparameters'->'clip_grad'->>'vals')::jsonb ? '8'
					WHEN config->'hyperparameters'->'clip_grad'->>'type' IN ('int', 'double', 'log') THEN (config->'hyperparameters'->'clip_grad'->>'minval')::float8 <= 8 OR (config->'hyperparameters'->'clip_grad'->>'maxval')::float8 >= 8
					ELSE false
				 END))))`,
		},
		{
			`{"filterGroup":{"children":[{"type":"COLUMN_TYPE_TEXT","location":"LOCATION_TYPE_HYPERPARAMETERS", "columnName":"hp.clip_grad.clip.grad","kind":"field","operator":"contains", "value":"some_string"}],"conjunction":"and","kind":"group"},"showArchived":true}`,
			`((((CASE
				WHEN config->'hyperparameters'->'clip_grad'->'clip'->'grad'->>'type' = 'const' THEN config->'hyperparameters'->'clip_grad'->'clip'->'grad'->>'val' LIKE '%some_string%'
				WHEN config->'hyperparameters'->'clip_grad'->'clip'->'grad'->>'type' = 'categorical' THEN (config->'hyperparameters'->'clip_grad'->'clip'->'grad'->>'vals')::jsonb ? 'some_string'
				ELSE false
			 END))))`,
		},
		{
			`{"filterGroup":{"children":[{"type":"COLUMN_TYPE_TEXT","location":"LOCATION_TYPE_HYPERPARAMETERS", "columnName":"hp.clip_grad.clip.grad","kind":"field","operator":"isEmpty", "value":"some_string"}],"conjunction":"and","kind":"group"},"showArchived":true}`,
			`((((CASE
				WHEN config->'hyperparameters'->'clip_grad'->'clip'->'grad'->>'type' = 'const' THEN config->'hyperparameters'->'clip_grad'->'clip'->'grad'->>'val' IS NULL
				WHEN config->'hyperparameters'->'clip_grad'->'clip'->'grad'->>'type' = 'categorical' THEN config->'hyperparameters'->'clip_grad'->'clip'->'grad'->>'vals' IS NULL
				ELSE false
			 END))))`,
		},
		{
			`{"filterGroup":{"children":[{"type":"COLUMN_TYPE_TEXT","location":"LOCATION_TYPE_HYPERPARAMETERS", "columnName":"hp.clip_grad.clip.grad","kind":"field","operator":"notContains", "value":"some_string"}],"conjunction":"and","kind":"group"},"showArchived":true}`,
			`((((CASE
				WHEN config->'hyperparameters'->'clip_grad'->'clip'->'grad'->>'type' = 'const' THEN config->'hyperparameters'->'clip_grad'->'clip'->'grad'->>'val' NOT LIKE '%some_string%'
				WHEN config->'hyperparameters'->'clip_grad'->'clip'->'grad'->>'type' = 'categorical' THEN (config->'hyperparameters'->'clip_grad'->'clip'->'grad'->>'vals')::jsonb ? 'some_string' IS NOT TRUE
				ELSE false
			 END))))`,
		},
		{
			`{"filterGroup":{"children":[{"type":"COLUMN_TYPE_NUMBER","location":"LOCATION_TYPE_HYPERPARAMETERS", "columnName":"hp.clip_grad","kind":"field","operator":"notContains", "value":8}],"conjunction":"and","kind":"group"},"showArchived":true}`,
			`((((CASE
					WHEN config->'hyperparameters'->'clip_grad'->>'type' = 'categorical' THEN ((config->'hyperparameters'->'clip_grad'->>'vals')::jsonb ? '8') IS NOT TRUE
					WHEN config->'hyperparameters'->'clip_grad'->>'type' IN ('int', 'double', 'log') THEN (config->'hyperparameters'->'clip_grad'->>'minval')::float8 >= 8 OR (config->'hyperparameters'->'clip_grad'->>'maxval')::float8 <= 8
					ELSE false
				 END))))`,
		},
	}
	for _, c := range validTestCases {
		q := db.Bun().NewSelect()
		var efr experimentFilterRoot
		err := json.Unmarshal([]byte(c[0]), &efr)
		require.NoError(t, err)
		q = q.WhereGroup(" AND ", func(q *bun.SelectQuery) *bun.SelectQuery {
			_, err = efr.toSQL(q)
			return q
		}).WhereGroup(" AND ", func(q *bun.SelectQuery) *bun.SelectQuery {
			if !efr.ShowArchived {
				return q.Where(`e.archived = false`)
			}
			return q
		})
		require.NoError(t, err)
		require.Equal(t, fmt.Sprintf(`SELECT * WHERE %v`, c[1]), q.String())
	}
}

func TestDeleteExperiments(t *testing.T) {
	var mockRM mocks.ResourceManager
	// Need _anything_ to error to check the error flow leaves things in DELETE_FAILED and that they are delete-able
	// still after that.
	mockRM.On("DeleteJob", mock.Anything).Return(func(sproto.DeleteJob) sproto.DeleteJobResponse {
		errC := make(chan error, 1)
		errC <- fmt.Errorf("something real bad")
		return sproto.DeleteJobResponse{Err: errC}
	}, nil)

	api, curUser, ctx := setupAPITest(t, nil, &mockRM)

	var expIDs []int32
	for i := 0; i < 3; i++ {
		exp := createTestExp(t, api, curUser)
		_, err := db.Bun().NewUpdate().Table("experiments").
			Set("state = ?", model.CompletedState).
			Where("id = ?", exp.ID).Exec(ctx)
		require.NoError(t, err)
		expIDs = append(expIDs, int32(exp.ID))
	}

	t.Log("if DeleteExperiment fails, all experiments become DELETE_FAILED")
	_, err := api.DeleteExperiments(ctx, &apiv1.DeleteExperimentsRequest{
		ProjectId:     1,
		ExperimentIds: expIDs,
	})
	require.NoError(t, err)

	var success bool
	for i := 0; i < 15; i++ {
		var inDeleteFailed int
		for _, expID := range expIDs {
			e, err := api.GetExperiment(ctx, &apiv1.GetExperimentRequest{ExperimentId: expID})
			require.NoError(t, err, "expected experiment to go to DELETE_FAILED")
			if e.Experiment.State == experimentv1.State_STATE_DELETE_FAILED {
				inDeleteFailed++
			}
		}
		if len(expIDs) == inDeleteFailed {
			success = true
			break
		}
		time.Sleep(1 * time.Second)
	}
	if !success {
		t.Error("expected experiments to move to DELETE_FAILED after 15 seconds and they did not")
	}

	t.Log("and if the RM then succeeds, the experiments are then successfully deleted")
	api.m.rm = MockRM()
	_, err = api.DeleteExperiments(ctx, &apiv1.DeleteExperimentsRequest{ProjectId: 1, ExperimentIds: expIDs})
	require.NoError(t, err)

	// Delete is async so we need to retry until it completes.
	for i := 0; i < 15; i++ {
		var deleted int
		for _, expID := range expIDs {
			e, err := api.GetExperiment(ctx, &apiv1.GetExperimentRequest{ExperimentId: expID})
			if err != nil {
				require.Equal(t, apiPkg.NotFoundErrs("experiment", strconv.Itoa(int(expID)), true), err)
				deleted++
				continue
			}
			require.NotEqual(t, experimentv1.State_STATE_DELETE_FAILED, e.Experiment.State)
		}
		if len(expIDs) == deleted {
			return
		}
		time.Sleep(1 * time.Second)
	}
	t.Error("expected experiments to delete after 15 seconds and they did not")
}

func TestDeleteExperimentsFiltered(t *testing.T) {
	var mockRM mocks.ResourceManager
	// Need _anything_ to error to check the error flow leaves things in DELETE_FAILED and that they are delete-able
	// still after that.
	mockRM.On("DeleteJob", mock.Anything).Return(func(sproto.DeleteJob) sproto.DeleteJobResponse {
		errC := make(chan error, 1)
		errC <- fmt.Errorf("something real bad")
		return sproto.DeleteJobResponse{Err: errC}
	}, nil)

	api, curUser, ctx := setupAPITest(t, nil, &mockRM)

	var expIDs []int32
	for i := 0; i < 3; i++ {
		exp := createTestExp(t, api, curUser)
		_, err := db.Bun().NewUpdate().Table("experiments").
			Set("state = ?", model.CompletedState).
			Where("id = ?", exp.ID).Exec(ctx)
		require.NoError(t, err)
		expIDs = append(expIDs, int32(exp.ID))
	}

	t.Log("if DeleteExperiment with filter fails, all experiments become DELETE_FAILED")
	// Try delete experiments using a filter, this tests the branch condition
	// where we have a filter with the experiment delete request.
	_, err := api.DeleteExperiments(ctx, &apiv1.DeleteExperimentsRequest{
		ProjectId:     1,
		ExperimentIds: expIDs,
		Filters:       &apiv1.BulkExperimentFilters{ProjectId: 1},
	})
	require.NoError(t, err)

	var success bool
	for i := 0; i < 15; i++ {
		var inDeleteFailed int
		for _, expID := range expIDs {
			e, err := api.GetExperiment(ctx, &apiv1.GetExperimentRequest{ExperimentId: expID})
			require.NoError(t, err, "expected experiment to go to DELETE_FAILED")
			if e.Experiment.State == experimentv1.State_STATE_DELETE_FAILED {
				inDeleteFailed++
			}
		}
		if len(expIDs) == inDeleteFailed {
			success = true
			break
		}
		time.Sleep(1 * time.Second)
	}
	if !success {
		t.Error("expected experiments to move to DELETE_FAILED after 15 seconds and they did not")
	}

	t.Log("and if the RM then succeeds, the experiments are then successfully deleted")
	api.m.rm = MockRM()
	_, err = api.DeleteExperiments(ctx, &apiv1.DeleteExperimentsRequest{ProjectId: 1, ExperimentIds: expIDs})
	require.NoError(t, err)

	// Delete is async so we need to retry until it completes.
	for i := 0; i < 15; i++ {
		var deleted int
		for _, expID := range expIDs {
			e, err := api.GetExperiment(ctx, &apiv1.GetExperimentRequest{ExperimentId: expID})
			if err != nil {
				require.Equal(t, apiPkg.NotFoundErrs("experiment", strconv.Itoa(int(expID)), true), err)
				deleted++
				continue
			}
			require.NotEqual(t, experimentv1.State_STATE_DELETE_FAILED, e.Experiment.State)
		}
		if len(expIDs) == deleted {
			return
		}
		time.Sleep(1 * time.Second)
	}
	t.Error("expected experiments to delete after 15 seconds and they did not")
}
