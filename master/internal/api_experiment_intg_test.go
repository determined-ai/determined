//go:build integration
// +build integration

package internal

import (
	"context"
	"fmt"
	"sort"
	"testing"
	"time"
	"unsafe"

	"github.com/uptrace/bun"

	"github.com/golang/protobuf/ptypes/wrappers"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/ghodss/yaml"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/determined-ai/determined/master/internal/db"
	expauth "github.com/determined-ai/determined/master/internal/experiment"
	"github.com/determined-ai/determined/master/internal/mocks"
	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/master/test/olddata"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/experimentv1"
	"github.com/determined-ai/determined/proto/pkg/userv1"
	"github.com/determined-ai/determined/proto/pkg/utilv1"
	"github.com/determined-ai/determined/proto/pkg/workspacev1"
)

type mockStream[T any] struct {
	ctx context.Context
}

func (m mockStream[T]) Send(resp T) error             { return nil }
func (m mockStream[T]) SetHeader(metadata.MD) error   { return nil }
func (m mockStream[T]) SendHeader(metadata.MD) error  { return nil }
func (m mockStream[T]) SetTrailer(metadata.MD)        {}
func (m mockStream[T]) Context() context.Context      { return m.ctx }
func (m mockStream[T]) SendMsg(mes interface{}) error { return nil }
func (m mockStream[T]) RecvMsg(mes interface{}) error { return nil }

func expNotFoundErr(expID int) error {
	return status.Errorf(codes.NotFound, "experiment not found: %d", expID)
}

var authZExp *mocks.ExperimentAuthZ

// pgdb can be nil to use the singleton database for testing.
func setupExpAuthTest(t *testing.T, pgdb *db.PgDB) (
	*apiServer, *mocks.ExperimentAuthZ, *mocks.ProjectAuthZ, model.User, context.Context,
) {
	api, projectAuthZ, _, user, ctx := setupProjectAuthZTest(t, pgdb)
	if authZExp == nil {
		authZExp = &mocks.ExperimentAuthZ{}
		expauth.AuthZProvider.Register("mock", authZExp)
	}
	return api, authZExp, projectAuthZ, user, ctx
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

//nolint: exhaustivestruct
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
			require.Equal(t, expNotFoundErr(exp.ID), err)
			return
		}
		require.NotEqual(t, experimentv1.State_STATE_DELETE_FAILED, e.Experiment.State)
	}
	t.Error("expected experiment to delete after 1 minute and it did not")
}

//nolint: exhaustivestruct
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

//nolint: exhaustivestruct
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
		StartTime:            startTime,
		EndTime:              &endTime,
		ModelDefinitionBytes: []byte{1, 2, 3},
		JobID:                model.JobID(job0ID),
		Archived:             false,
		State:                model.PausedState,
		Notes:                "notes",
		Config:               activeConfig0.AsLegacy(),
		OwnerID:              ptrs.Ptr(model.UserID(1)),
		ProjectID:            int(pid),
	}
	require.NoError(t, api.m.db.AddExperiment(exp0, activeConfig0))
	for i := 0; i < 3; i++ {
		task := &model.Task{TaskType: model.TaskTypeTrial}
		require.NoError(t, api.m.db.AddTask(task))
		require.NoError(t, api.m.db.AddTrial(&model.Trial{
			State:        model.PausedState,
			ExperimentID: exp0.ID,
			TaskID:       task.TaskID,
		}))
	}
	exp0Expected := &experimentv1.Experiment{
		Id:             int32(exp0.ID),
		Description:    *activeConfig0.RawDescription,
		Labels:         []string{"l0", "l1"},
		State:          experimentv1.State_STATE_PAUSED,
		StartTime:      timestamppb.New(startTime),
		EndTime:        timestamppb.New(endTime),
		Archived:       false,
		NumTrials:      3,
		DisplayName:    "admin",
		UserId:         1,
		Username:       "admin",
		SearcherType:   "single",
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
		StartTime:            secondStartTime,
		ModelDefinitionBytes: []byte{1, 2, 3},
		JobID:                model.JobID(job1ID),
		Archived:             true,
		State:                model.ErrorState,
		ParentID:             ptrs.Ptr(exp0.ID),
		Config:               activeConfig1.AsLegacy(),
		OwnerID:              ptrs.Ptr(model.UserID(userResp.User.Id)),
		ProjectID:            int(pid),
	}
	require.NoError(t, api.m.db.AddExperiment(exp1, activeConfig1))
	exp1Expected := &experimentv1.Experiment{
		StartTime:      timestamppb.New(secondStartTime),
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
	getExperimentsTest(ctx, t, api, pid, &apiv1.GetExperimentsRequest{}, exp0Expected, exp1Expected)

	getExperimentsTest(ctx, t, api, pid,
		&apiv1.GetExperimentsRequest{Description: "12345"}, exp0Expected)
	getExperimentsTest(ctx, t, api, pid,
		&apiv1.GetExperimentsRequest{Description: "234"}, exp0Expected, exp1Expected)
	getExperimentsTest(ctx, t, api, pid,
		&apiv1.GetExperimentsRequest{Description: "123456"})

	getExperimentsTest(ctx, t, api, pid,
		&apiv1.GetExperimentsRequest{Name: "longername"}, exp1Expected)
	getExperimentsTest(ctx, t, api, pid,
		&apiv1.GetExperimentsRequest{Name: "name"}, exp0Expected, exp1Expected)
	getExperimentsTest(ctx, t, api, pid,
		&apiv1.GetExperimentsRequest{Name: "longlongername"})

	getExperimentsTest(ctx, t, api, pid,
		&apiv1.GetExperimentsRequest{Labels: []string{"l0", "l1"}}, exp0Expected)
	getExperimentsTest(ctx, t, api, pid,
		&apiv1.GetExperimentsRequest{Labels: []string{"l0"}}, exp0Expected, exp1Expected)
	getExperimentsTest(ctx, t, api, pid,
		&apiv1.GetExperimentsRequest{Labels: []string{"l0", "l1", "l3"}})

	getExperimentsTest(ctx, t, api, pid,
		&apiv1.GetExperimentsRequest{Archived: wrapperspb.Bool(false)}, exp0Expected)
	getExperimentsTest(ctx, t, api, pid,
		&apiv1.GetExperimentsRequest{Archived: wrapperspb.Bool(true)}, exp1Expected)

	getExperimentsTest(ctx, t, api, pid,
		&apiv1.GetExperimentsRequest{
			States: []experimentv1.State{experimentv1.State_STATE_PAUSED},
		}, exp0Expected)
	getExperimentsTest(ctx, t, api, pid,
		&apiv1.GetExperimentsRequest{
			States: []experimentv1.State{experimentv1.State_STATE_ERROR},
		}, exp1Expected)
	getExperimentsTest(ctx, t, api, pid,
		&apiv1.GetExperimentsRequest{
			States: []experimentv1.State{
				experimentv1.State_STATE_PAUSED,
				experimentv1.State_STATE_ERROR,
			},
		}, exp0Expected, exp1Expected)
	getExperimentsTest(ctx, t, api, pid,
		&apiv1.GetExperimentsRequest{
			States: []experimentv1.State{experimentv1.State_STATE_CANCELED},
		})

	getExperimentsTest(ctx, t, api, pid,
		&apiv1.GetExperimentsRequest{Users: []string{"admin"}}, exp0Expected)
	getExperimentsTest(ctx, t, api, pid,
		&apiv1.GetExperimentsRequest{Users: []string{userResp.User.Username}}, exp1Expected)
	getExperimentsTest(ctx, t, api, pid,
		&apiv1.GetExperimentsRequest{Users: []string{"admin", userResp.User.Username}},
		exp0Expected, exp1Expected)
	getExperimentsTest(ctx, t, api, pid,
		&apiv1.GetExperimentsRequest{Users: []string{"notarealuser"}})

	getExperimentsTest(ctx, t, api, pid,
		&apiv1.GetExperimentsRequest{UserIds: []int32{1}}, exp0Expected)
	getExperimentsTest(ctx, t, api, pid,
		&apiv1.GetExperimentsRequest{UserIds: []int32{userResp.User.Id}}, exp1Expected)
	getExperimentsTest(ctx, t, api, pid,
		&apiv1.GetExperimentsRequest{UserIds: []int32{1, userResp.User.Id}},
		exp0Expected, exp1Expected)
	getExperimentsTest(ctx, t, api, pid, &apiv1.GetExperimentsRequest{UserIds: []int32{-999}})

	// Sort and order by tests.
	getExperimentsTest(ctx, t, api, pid,
		&apiv1.GetExperimentsRequest{
			SortBy: apiv1.GetExperimentsRequest_SORT_BY_NUM_TRIALS,
		}, exp1Expected, exp0Expected)
	getExperimentsTest(ctx, t, api, pid,
		&apiv1.GetExperimentsRequest{
			SortBy:  apiv1.GetExperimentsRequest_SORT_BY_NUM_TRIALS,
			OrderBy: apiv1.OrderBy_ORDER_BY_ASC,
		}, exp1Expected, exp0Expected)
	getExperimentsTest(ctx, t, api, pid,
		&apiv1.GetExperimentsRequest{
			SortBy:  apiv1.GetExperimentsRequest_SORT_BY_NUM_TRIALS,
			OrderBy: apiv1.OrderBy_ORDER_BY_DESC,
		}, exp0Expected, exp1Expected)

	// Pagination tests.
	// No experiments should be returned for Limit -2.
	getExperimentsTest(ctx, t, api, pid, &apiv1.GetExperimentsRequest{Limit: -2})
	getExperimentsPageTest(ctx, t, api, pid, &apiv1.GetExperimentsRequest{Offset: 1},
		&apiv1.Pagination{Offset: 1, Limit: 0, StartIndex: 1, EndIndex: 2, Total: 2})
	getExperimentsPageTest(ctx, t, api, pid, &apiv1.GetExperimentsRequest{Limit: 1},
		&apiv1.Pagination{Offset: 0, Limit: 1, StartIndex: 0, EndIndex: 1, Total: 2})
	getExperimentsPageTest(ctx, t, api, pid, &apiv1.GetExperimentsRequest{Limit: 1, Offset: 1},
		&apiv1.Pagination{Offset: 1, Limit: 1, StartIndex: 1, EndIndex: 2, Total: 2})
	getExperimentsPageTest(ctx, t, api, pid, &apiv1.GetExperimentsRequest{Offset: 2},
		&apiv1.Pagination{Offset: 2, Limit: 0, StartIndex: 2, EndIndex: 2, Total: 2})

	getExperimentsPageTest(ctx, t, api, pid, &apiv1.GetExperimentsRequest{Limit: -1},
		&apiv1.Pagination{Offset: 0, Limit: -1, StartIndex: 0, EndIndex: 2, Total: 2})
}

func getExperimentsPageTest(ctx context.Context, t *testing.T, api *apiServer, pid int32,
	req *apiv1.GetExperimentsRequest, expected *apiv1.Pagination,
) {
	req.ProjectId = pid
	res, err := api.GetExperiments(ctx, req)
	require.NoError(t, err)
	proto.Equal(expected, res.Pagination)
	require.Equal(t, expected, res.Pagination)
}

func getExperimentsTest(ctx context.Context, t *testing.T, api *apiServer, pid int32,
	req *apiv1.GetExperimentsRequest, expected ...*experimentv1.Experiment,
) {
	req.ProjectId = pid
	res, err := api.GetExperiments(ctx, req)
	require.NoError(t, err)
	require.Equal(t, len(expected), len(res.Experiments),
		fmt.Sprintf("wrong length of result set with request %+v", req))

	for i := range expected {
		sort.Strings(expected[i].Labels)
		sort.Strings(res.Experiments[i].Labels)

		// Don't compare config.
		res.Experiments[i].Config = nil

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
			Id:     prse.CompletedPBTExpID,
			SortBy: apiv1.GetExperimentCheckpointsRequest_SORT_BY_SEARCHER_METRIC,
		}
		_, err = api.GetExperimentCheckpoints(ctx, req)
		require.NoError(t, err)
	})

	t.Run("MetricNames", func(t *testing.T) {
		req := &apiv1.MetricNamesRequest{
			ExperimentId: prse.CompletedPBTExpID,
		}
		err = api.MetricNames(req, mockStream[*apiv1.MetricNamesResponse]{ctx})
		require.NoError(t, err)
	})

	t.Run("TrialsSample", func(t *testing.T) {
		req := &apiv1.TrialsSampleRequest{
			ExperimentId: prse.CompletedAdaptiveSimpleExpID,
			MetricName:   "loss",
			MetricType:   apiv1.MetricType_METRIC_TYPE_TRAINING,
		}
		err = api.TrialsSample(req, mockStream[*apiv1.TrialsSampleResponse]{ctx})
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

//nolint: exhaustivestruct
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
		ModelDefinitionBytes: []byte{1, 2, 3},
		State:                model.PausedState,
		Config:               activeConfig.AsLegacy(),
		OwnerID:              ptrs.Ptr(model.UserID(userResp.User.Id)),
		ProjectID:            1,
	}
	for i := 0; i < n; i++ {
		jobID := uuid.New().String()
		exp.ID = 0
		exp.JobID = model.JobID(jobID)

		if err := api.m.db.AddExperiment(exp, activeConfig); err != nil {
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

//nolint: exhaustivestruct
func createTestExpWithProjectID(
	t *testing.T, api *apiServer, curUser model.User, projectID int, labels ...string,
) *model.Experiment {
	labelMap := make(map[string]bool)
	for _, l := range labels {
		labelMap[l] = true
	}

	activeConfig := schemas.Merge(minExpConfig, expconf.ExperimentConfig{
		RawLabels:      labelMap,
		RawDescription: ptrs.Ptr("desc"),
		RawName:        expconf.Name{RawString: ptrs.Ptr("name")},
	})
	activeConfig = schemas.WithDefaults(activeConfig)
	exp := &model.Experiment{
		JobID:                model.JobID(uuid.New().String()),
		State:                model.PausedState,
		OwnerID:              &curUser.ID,
		ProjectID:            projectID,
		StartTime:            time.Now(),
		ModelDefinitionBytes: []byte{10, 11, 12},
		Config:               activeConfig.AsLegacy(),
	}
	require.NoError(t, api.m.db.AddExperiment(exp, activeConfig))

	// Get experiment as our API mostly will to make it easier to mock.
	exp, err := api.m.db.ExperimentByID(exp.ID)
	require.NoError(t, err)
	return exp
}

func TestAuthZGetExperiment(t *testing.T) {
	api, authZExp, _, curUser, ctx := setupExpAuthTest(t, nil)
	exp := createTestExp(t, api, curUser)

	// Not found returns same as permission denied.
	_, err := api.GetExperiment(ctx, &apiv1.GetExperimentRequest{ExperimentId: -999})
	require.Equal(t, expNotFoundErr(-999).Error(), err.Error())

	authZExp.On("CanGetExperiment", mock.Anything, curUser, mock.Anything).
		Return(false, nil).Once()
	_, err = api.GetExperiment(ctx, &apiv1.GetExperimentRequest{ExperimentId: int32(exp.ID)})
	require.Equal(t, expNotFoundErr(exp.ID).Error(), err.Error())

	// Error returns error unmodified.
	expectedErr := fmt.Errorf("canGetExperimentError")
	authZExp.On("CanGetExperiment", mock.Anything, curUser, mock.Anything).
		Return(false, expectedErr).Once()
	_, err = api.GetExperiment(ctx, &apiv1.GetExperimentRequest{ExperimentId: int32(exp.ID)})
	require.Equal(t, expectedErr, err)

	authZExp.On("CanGetExperiment", mock.Anything, curUser, mock.Anything).Return(true, nil).Once()
	res, err := api.GetExperiment(ctx, &apiv1.GetExperimentRequest{ExperimentId: int32(exp.ID)})
	require.NoError(t, err)
	require.Equal(t, int32(exp.ID), res.Experiment.Id)
}

func TestAuthZGetExperiments(t *testing.T) {
	api, authZExp, authZProject, curUser, ctx := setupExpAuthTest(t, nil)
	_, projectID := createProjectAndWorkspace(ctx, t, api)
	exp0 := createTestExpWithProjectID(t, api, curUser, projectID)
	createTestExpWithProjectID(t, api, curUser, projectID, uuid.New().String())

	// Can't view project gets a 404.
	authZProject.On("CanGetProject", mock.Anything, curUser, mock.Anything).Return(false, nil).Once()
	_, err := api.GetExperiments(ctx, &apiv1.GetExperimentsRequest{ProjectId: int32(projectID)})
	require.Equal(t, projectNotFoundErr(projectID).Error(), err.Error())

	// Error from FilterExperimentsQuery passes through.
	authZProject.On("CanGetProject", mock.Anything, curUser, mock.Anything).
		Return(true, nil).Once()
	expectedErr := fmt.Errorf("filterExperimentsQueryError")
	authZExp.On("FilterExperimentsQuery", mock.Anything, curUser, mock.Anything, mock.Anything).
		Return(nil, expectedErr).Once()
	_, err = api.GetExperiments(ctx, &apiv1.GetExperimentsRequest{ProjectId: int32(projectID)})
	require.Equal(t, expectedErr, err)

	// Filter only to only one experiment ID.
	resQuery := &bun.SelectQuery{}
	authZExp.On("FilterExperimentsQuery", mock.Anything, curUser, mock.Anything, mock.Anything).
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
	authZProject.On("CanGetProject", mock.Anything, curUser, mock.Anything).Return(false, nil).Once()
	_, err := api.GetExperimentLabels(ctx, &apiv1.GetExperimentLabelsRequest{
		ProjectId: int32(projectID),
	})
	require.Equal(t, projectNotFoundErr(projectID).Error(), err.Error())

	// Error from FilterExperimentsLabelsQuery passes through.
	authZProject.On("CanGetProject", mock.Anything, curUser, mock.Anything).Return(true, nil).Once()
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
	_, projectID := createProjectAndWorkspace(ctx, t, api)

	// Can't view forked experiment.
	authZExp.On("CanGetExperiment", mock.Anything, curUser, mock.Anything).Return(false, nil).Once()
	_, err := api.CreateExperiment(ctx, &apiv1.CreateExperimentRequest{
		ParentId: int32(forkFrom.ID),
	})
	require.Equal(t, expNotFoundErr(forkFrom.ID), err)

	// Can't fork from experiment.
	expectedErr := status.Errorf(codes.PermissionDenied, "canForkExperimentError")
	authZExp.On("CanGetExperiment", mock.Anything, curUser, mock.Anything).Return(true, nil).Once()
	authZExp.On("CanForkFromExperiment", mock.Anything, curUser, mock.Anything).
		Return(fmt.Errorf("canForkExperimentError")).Once()
	_, err = api.CreateExperiment(ctx, &apiv1.CreateExperimentRequest{
		ParentId: int32(forkFrom.ID),
	})
	require.Equal(t, expectedErr, err)

	// Can't view project passed in.
	pAuthZ.On("CanGetProject", mock.Anything, curUser, mock.Anything).Return(false, nil).Once()
	_, err = api.CreateExperiment(ctx, &apiv1.CreateExperimentRequest{
		ProjectId: int32(projectID),
		Config:    minExpConfToYaml(t),
	})
	require.Equal(t, status.Errorf(codes.NotFound,
		fmt.Sprintf("project (%d) not found", projectID)), err)

	// Can't view project passed in from config.
	pAuthZ.On("CanGetProject", mock.Anything, curUser, mock.Anything).Return(false, nil).Once()
	_, err = api.CreateExperiment(ctx, &apiv1.CreateExperimentRequest{
		Config: minExpConfToYaml(t) + "project: Uncategorized\nworkspace: Uncategorized",
	})
	require.Equal(t, status.Errorf(codes.NotFound,
		"workspace 'Uncategorized' or project 'Uncategorized' not found"), err)

	// Same as passing in a non existent project.
	_, err = api.CreateExperiment(ctx, &apiv1.CreateExperimentRequest{
		Config: minExpConfToYaml(t) + "project: doesntexist123\nworkspace: doesntexist123",
	})
	require.Equal(t, status.Errorf(codes.NotFound,
		"workspace 'doesntexist123' or project 'doesntexist123' not found"), err)

	// Can't create experiment deny.
	expectedErr = status.Errorf(codes.PermissionDenied, "canCreateExperimentError")
	pAuthZ.On("CanGetProject", mock.Anything, curUser, mock.Anything).Return(true, nil).Once()
	authZExp.On("CanCreateExperiment", mock.Anything, curUser, mock.Anything, mock.Anything).
		Return(fmt.Errorf("canCreateExperimentError")).Once()
	_, err = api.CreateExperiment(ctx, &apiv1.CreateExperimentRequest{
		ProjectId: int32(projectID),
		Config:    minExpConfToYaml(t),
	})
	require.Equal(t, expectedErr, err)

	// Can't activate experiment deny.
	expectedErr = status.Errorf(codes.PermissionDenied, "canActivateExperimentError")
	pAuthZ.On("CanGetProject", mock.Anything, curUser, mock.Anything).Return(true, nil).Once()
	authZExp.On("CanCreateExperiment", mock.Anything, curUser, mock.Anything, mock.Anything).
		Return(nil).Once()
	authZExp.On("CanEditExperiment", mock.Anything, curUser, mock.Anything, mock.Anything).Return(
		fmt.Errorf("canActivateExperimentError")).Once()
	_, err = api.CreateExperiment(ctx, &apiv1.CreateExperimentRequest{
		Activate: true,
		Config:   minExpConfToYaml(t),
	})
	require.Equal(t, expectedErr, err)
}

func TestAuthZExpCompareTrialsSample(t *testing.T) {
	api, authZExp, _, curUser, ctx := setupExpAuthTest(t, nil)

	exp0 := createTestExp(t, api, curUser)
	exp1 := createTestExp(t, api, curUser)
	req := &apiv1.ExpCompareTrialsSampleRequest{
		ExperimentIds: []int32{int32(exp0.ID), int32(exp1.ID)},
		MetricName:    "name",
		MetricType:    apiv1.MetricType_METRIC_TYPE_TRAINING,
	}

	// Can't view first experiment gets error.
	expectedErr := status.Errorf(codes.PermissionDenied, "firstError")
	authZExp.On("CanGetExperiment", mock.Anything, curUser, exp0).Return(true, nil).Once()
	authZExp.On("CanGetExperimentArtifacts", mock.Anything, curUser, exp0).
		Return(fmt.Errorf("firstError")).Once()
	err := api.ExpCompareTrialsSample(req, mockStream[*apiv1.ExpCompareTrialsSampleResponse]{ctx})
	require.Equal(t, expectedErr.Error(), err.Error())

	// Can't view second experiment gets error.
	expectedErr = status.Errorf(codes.PermissionDenied, "secondError")
	authZExp.On("CanGetExperiment", mock.Anything, curUser, exp0).Return(true, nil).Once()
	authZExp.On("CanGetExperimentArtifacts", mock.Anything, curUser, exp0).Return(nil).Once()
	authZExp.On("CanGetExperiment", mock.Anything, curUser, exp1).Return(true, nil).Once()
	authZExp.On("CanGetExperimentArtifacts", mock.Anything, curUser, exp1).
		Return(fmt.Errorf("secondError")).Once()
	err = api.ExpCompareTrialsSample(req, mockStream[*apiv1.ExpCompareTrialsSampleResponse]{ctx})
	require.Equal(t, expectedErr.Error(), err.Error())
}

func TestAuthZGetExperimentAndCanDoActions(t *testing.T) {
	api, authZExp, _, curUser, ctx := setupExpAuthTest(t, nil)
	exp := createTestExp(t, api, curUser)

	cases := []struct {
		DenyFuncName string
		IDToReqCall  func(id int) error
	}{
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
		{"CanEditExperiment", func(id int) error {
			_, err := api.ActivateExperiment(ctx, &apiv1.ActivateExperimentRequest{
				Id: int32(id),
			})
			return err
		}},
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
			return api.MetricNames(&apiv1.MetricNamesRequest{
				ExperimentId: int32(id),
			}, mockStream[*apiv1.MetricNamesResponse]{ctx})
		}},
		{"CanGetExperimentArtifacts", func(id int) error {
			return api.MetricBatches(&apiv1.MetricBatchesRequest{
				ExperimentId: int32(id),
				MetricName:   "name",
				MetricType:   apiv1.MetricType_METRIC_TYPE_TRAINING,
			}, mockStream[*apiv1.MetricBatchesResponse]{ctx})
		}},
		{"CanGetExperimentArtifacts", func(id int) error {
			return api.TrialsSnapshot(&apiv1.TrialsSnapshotRequest{
				ExperimentId: int32(id),
				MetricName:   "name",
				MetricType:   apiv1.MetricType_METRIC_TYPE_TRAINING,
			}, mockStream[*apiv1.TrialsSnapshotResponse]{ctx})
		}},
		{"CanGetExperimentArtifacts", func(id int) error {
			return api.TrialsSample(&apiv1.TrialsSampleRequest{
				ExperimentId: int32(id),
				MetricName:   "name",
				MetricType:   apiv1.MetricType_METRIC_TYPE_TRAINING,
			}, mockStream[*apiv1.TrialsSampleResponse]{ctx})
		}},
		{"CanEditExperiment", func(id int) error {
			_, err := api.ComputeHPImportance(ctx, &apiv1.ComputeHPImportanceRequest{
				ExperimentId: int32(id),
			})
			return err
		}},
		{"CanGetExperimentArtifacts", func(id int) error {
			return api.GetHPImportance(&apiv1.GetHPImportanceRequest{
				ExperimentId: int32(id),
			}, mockStream[*apiv1.GetHPImportanceResponse]{ctx})
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
			_, err := api.LaunchTensorboard(ctx, &apiv1.LaunchTensorboardRequest{
				ExperimentIds: []int32{int32(id)},
			})
			return err
		}},
	}

	for _, curCase := range cases {
		// Not found returns same as permission denied.
		require.Equal(t, expNotFoundErr(-999), curCase.IDToReqCall(-999))

		authZExp.On("CanGetExperiment", mock.Anything, curUser, mock.Anything).
			Return(false, nil).Once()
		require.Equal(t, expNotFoundErr(exp.ID), curCase.IDToReqCall(exp.ID))

		// CanGetExperiment error returns unmodified.
		expectedErr := fmt.Errorf("canGetExperimentError")
		authZExp.On("CanGetExperiment", mock.Anything, curUser, mock.Anything).
			Return(false, expectedErr).Once()
		require.Equal(t, expectedErr, curCase.IDToReqCall(exp.ID))

		// Deny returns error with PermissionDenied.
		expectedErr = status.Errorf(codes.PermissionDenied, curCase.DenyFuncName+"Error")
		authZExp.On("CanGetExperiment", mock.Anything, curUser, mock.Anything).
			Return(true, nil).Once()
		authZExp.On(curCase.DenyFuncName, mock.Anything, curUser, mock.Anything).
			Return(fmt.Errorf(curCase.DenyFuncName + "Error")).Once()
		require.Equal(t, expectedErr.Error(), curCase.IDToReqCall(exp.ID).Error())
	}
}
