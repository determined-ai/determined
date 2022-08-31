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

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/golang/protobuf/ptypes/wrappers"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/experimentv1"
	"github.com/determined-ai/determined/proto/pkg/userv1"
)

var minExpConfig = expconf.ExperimentConfig{
	RawResources: &expconf.ResourcesConfig{
		RawResourcePool: ptrs.Ptr("kubernetes"),
	},
	RawEntrypoint: &expconf.EntrypointV0{"test"},
	RawCheckpointStorage: &expconf.CheckpointStorageConfig{
		RawSharedFSConfig: &expconf.SharedFSConfig{
			RawHostPath: ptrs.Ptr("/"),
		},
	},
	RawHyperparameters: expconf.Hyperparameters{},
	RawReproducibility: &expconf.ReproducibilityConfig{ptrs.Ptr(uint32(42))},
	RawSearcher: &expconf.SearcherConfig{
		RawMetric: ptrs.Ptr("loss"),
		RawSingleConfig: &expconf.SingleConfig{
			&expconf.Length{Units: 10, Unit: "batches"},
		},
	},
}

func TestGetExperiments(t *testing.T) {
	// Setup.
	api, _, ctx := SetupAPITest(t)

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
	exp0 := &model.Experiment{
		StartTime:            startTime,
		EndTime:              &endTime,
		ModelDefinitionBytes: []byte{1, 2, 3},
		JobID:                model.JobID(job0ID),
		Archived:             false,
		State:                model.PausedState,
		Notes:                "notes",
		Config: schemas.Merge(minExpConfig, expconf.ExperimentConfig{
			RawDescription: ptrs.Ptr("12345"),
			RawName:        expconf.Name{ptrs.Ptr("name")},
			RawLabels:      expconf.Labels{"l0": true, "l1": true},
		}).(expconf.ExperimentConfig),
		OwnerID:   ptrs.Ptr(model.UserID(1)),
		ProjectID: int(pid),
	}
	require.NoError(t, api.m.db.AddExperiment(exp0))
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
		Description:    *exp0.Config.RawDescription,
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
	exp1 := &model.Experiment{
		StartTime:            secondStartTime,
		ModelDefinitionBytes: []byte{1, 2, 3},
		JobID:                model.JobID(job1ID),
		Archived:             true,
		State:                model.ErrorState,
		ParentID:             ptrs.Ptr(exp0.ID),
		Config: schemas.Merge(minExpConfig, expconf.ExperimentConfig{
			RawDescription: ptrs.Ptr("234"),
			RawName:        expconf.Name{ptrs.Ptr("longername")},
			RawLabels:      expconf.Labels{"l0": true},
		}).(expconf.ExperimentConfig),
		OwnerID:   ptrs.Ptr(model.UserID(userResp.User.Id)),
		ProjectID: int(pid),
	}
	require.NoError(t, api.m.db.AddExperiment(exp1))
	exp1Expected := &experimentv1.Experiment{
		StartTime:      timestamppb.New(secondStartTime),
		Id:             int32(exp1.ID),
		Description:    *exp1.Config.RawDescription,
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
	getExperimentsTest(t, api, ctx, pid, &apiv1.GetExperimentsRequest{}, exp0Expected, exp1Expected)

	getExperimentsTest(t, api, ctx, pid,
		&apiv1.GetExperimentsRequest{Description: "12345"}, exp0Expected)
	getExperimentsTest(t, api, ctx, pid,
		&apiv1.GetExperimentsRequest{Description: "234"}, exp0Expected, exp1Expected)
	getExperimentsTest(t, api, ctx, pid,
		&apiv1.GetExperimentsRequest{Description: "123456"})

	getExperimentsTest(t, api, ctx, pid,
		&apiv1.GetExperimentsRequest{Name: "longername"}, exp1Expected)
	getExperimentsTest(t, api, ctx, pid,
		&apiv1.GetExperimentsRequest{Name: "name"}, exp0Expected, exp1Expected)
	getExperimentsTest(t, api, ctx, pid,
		&apiv1.GetExperimentsRequest{Name: "longlongername"})

	getExperimentsTest(t, api, ctx, pid,
		&apiv1.GetExperimentsRequest{Labels: []string{"l0", "l1"}}, exp0Expected)
	getExperimentsTest(t, api, ctx, pid,
		&apiv1.GetExperimentsRequest{Labels: []string{"l0"}}, exp0Expected, exp1Expected)
	getExperimentsTest(t, api, ctx, pid,
		&apiv1.GetExperimentsRequest{Labels: []string{"l0", "l1", "l3"}})

	getExperimentsTest(t, api, ctx, pid,
		&apiv1.GetExperimentsRequest{Archived: wrapperspb.Bool(false)}, exp0Expected)
	getExperimentsTest(t, api, ctx, pid,
		&apiv1.GetExperimentsRequest{Archived: wrapperspb.Bool(true)}, exp1Expected)

	getExperimentsTest(t, api, ctx, pid,
		&apiv1.GetExperimentsRequest{
			States: []experimentv1.State{experimentv1.State_STATE_PAUSED},
		}, exp0Expected)
	getExperimentsTest(t, api, ctx, pid,
		&apiv1.GetExperimentsRequest{
			States: []experimentv1.State{experimentv1.State_STATE_ERROR},
		}, exp1Expected)
	getExperimentsTest(t, api, ctx, pid,
		&apiv1.GetExperimentsRequest{
			States: []experimentv1.State{
				experimentv1.State_STATE_PAUSED,
				experimentv1.State_STATE_ERROR,
			},
		}, exp0Expected, exp1Expected)
	getExperimentsTest(t, api, ctx, pid,
		&apiv1.GetExperimentsRequest{
			States: []experimentv1.State{experimentv1.State_STATE_CANCELED},
		})

	getExperimentsTest(t, api, ctx, pid,
		&apiv1.GetExperimentsRequest{Users: []string{"admin"}}, exp0Expected)
	getExperimentsTest(t, api, ctx, pid,
		&apiv1.GetExperimentsRequest{Users: []string{userResp.User.Username}}, exp1Expected)
	getExperimentsTest(t, api, ctx, pid,
		&apiv1.GetExperimentsRequest{Users: []string{"admin", userResp.User.Username}},
		exp0Expected, exp1Expected)
	getExperimentsTest(t, api, ctx, pid, &apiv1.GetExperimentsRequest{Users: []string{"notarealuser"}})

	getExperimentsTest(t, api, ctx, pid,
		&apiv1.GetExperimentsRequest{UserIds: []int32{1}}, exp0Expected)
	getExperimentsTest(t, api, ctx, pid,
		&apiv1.GetExperimentsRequest{UserIds: []int32{userResp.User.Id}}, exp1Expected)
	getExperimentsTest(t, api, ctx, pid,
		&apiv1.GetExperimentsRequest{UserIds: []int32{1, userResp.User.Id}},
		exp0Expected, exp1Expected)
	getExperimentsTest(t, api, ctx, pid, &apiv1.GetExperimentsRequest{UserIds: []int32{-999}})

	// Sort and order by tests.
	getExperimentsTest(t, api, ctx, pid,
		&apiv1.GetExperimentsRequest{
			SortBy: apiv1.GetExperimentsRequest_SORT_BY_NUM_TRIALS,
		}, exp1Expected, exp0Expected)
	getExperimentsTest(t, api, ctx, pid,
		&apiv1.GetExperimentsRequest{
			SortBy:  apiv1.GetExperimentsRequest_SORT_BY_NUM_TRIALS,
			OrderBy: apiv1.OrderBy_ORDER_BY_ASC,
		}, exp1Expected, exp0Expected)
	getExperimentsTest(t, api, ctx, pid,
		&apiv1.GetExperimentsRequest{
			SortBy:  apiv1.GetExperimentsRequest_SORT_BY_NUM_TRIALS,
			OrderBy: apiv1.OrderBy_ORDER_BY_DESC,
		}, exp0Expected, exp1Expected)

	// Pagination tests.
	// No experiments should be returned for Limit -2.
	getExperimentsTest(t, api, ctx, pid, &apiv1.GetExperimentsRequest{Limit: -2})
	getExperimentsPageTest(t, api, ctx, pid, &apiv1.GetExperimentsRequest{Offset: 1},
		&apiv1.Pagination{Offset: 1, Limit: 0, StartIndex: 1, EndIndex: 2, Total: 2})
	getExperimentsPageTest(t, api, ctx, pid, &apiv1.GetExperimentsRequest{Limit: 1},
		&apiv1.Pagination{Offset: 0, Limit: 1, StartIndex: 0, EndIndex: 1, Total: 2})
	getExperimentsPageTest(t, api, ctx, pid, &apiv1.GetExperimentsRequest{Limit: 1, Offset: 1},
		&apiv1.Pagination{Offset: 1, Limit: 1, StartIndex: 1, EndIndex: 2, Total: 2})
	getExperimentsPageTest(t, api, ctx, pid, &apiv1.GetExperimentsRequest{Offset: 2},
		&apiv1.Pagination{Offset: 2, Limit: 0, StartIndex: 2, EndIndex: 2, Total: 2})

	getExperimentsPageTest(t, api, ctx, pid, &apiv1.GetExperimentsRequest{Limit: -1},
		&apiv1.Pagination{Offset: 0, Limit: -1, StartIndex: 0, EndIndex: 2, Total: 2})
}

func getExperimentsPageTest(t *testing.T, api *apiServer, ctx context.Context, pid int32,
	req *apiv1.GetExperimentsRequest, expected *apiv1.Pagination,
) {
	req.ProjectId = pid
	res, err := api.GetExperiments(ctx, req)
	require.NoError(t, err)
	proto.Equal(expected, res.Pagination)
	require.Equal(t, expected, res.Pagination)
}

func getExperimentsTest(t *testing.T, api *apiServer, ctx context.Context, pid int32,
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

var res *apiv1.GetExperimentsResponse // Avoid compiler optimizing res out.

func benchmarkGetExperiments(b *testing.B, n int) {
	// This should be fine as long as no error happens. For some
	// reason passing nil gives an error. In addition this
	// benchmark won't run when integration tests run
	// (since it needs the -bench flag) so if this breaks in the
	// future it won't cause any issues.
	api, _, ctx := SetupAPITest((*testing.T)(unsafe.Pointer(b)))

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

	exp := &model.Experiment{
		ModelDefinitionBytes: []byte{1, 2, 3},
		State:                model.PausedState,
		Config: schemas.Merge(minExpConfig, expconf.ExperimentConfig{
			RawDescription: ptrs.Ptr("desc"),
			RawName:        expconf.Name{ptrs.Ptr("name")},
		}).(expconf.ExperimentConfig),
		OwnerID:   ptrs.Ptr(model.UserID(userResp.User.Id)),
		ProjectID: 1,
	}
	for i := 0; i < n; i++ {
		jobID := uuid.New().String()
		exp.ID = 0
		exp.JobID = model.JobID(jobID)

		if err := api.m.db.AddExperiment(exp); err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var err error
		res, err = api.GetExperiments(ctx, &apiv1.GetExperimentsRequest{
			Limit: -1, UserIds: []int32{int32(userResp.User.Id)},
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
