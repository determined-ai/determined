//go:build integration
// +build integration

package internal

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

func TestSearchRunsSort(t *testing.T) {
	api, curUser, ctx := setupAPITest(t, nil)
	_, projectIDInt := createProjectAndWorkspace(ctx, t, api)
	projectID := int32(projectIDInt)

	// Empty response causes no errors.
	req := &apiv1.SearchRunsRequest{
		ProjectId: &projectID,
		Sort:      ptrs.Ptr("id=asc"),
	}
	resp, err := api.SearchRuns(ctx, req)
	require.NoError(t, err)
	require.Len(t, resp.Runs, 0)

	hyperparameters := map[string]any{"global_batch_size": 1, "test1": map[string]any{"test2": 1}}

	exp := createTestExpWithProjectID(t, api, curUser, projectIDInt)

	task := &model.Task{TaskType: model.TaskTypeTrial, TaskID: model.NewTaskID()}
	require.NoError(t, db.AddTask(ctx, task))
	require.NoError(t, db.AddTrial(ctx, &model.Trial{
		State:        model.PausedState,
		ExperimentID: exp.ID,
		StartTime:    time.Now(),
		HParams:      hyperparameters,
	}, task.TaskID))

	resp, err = api.SearchRuns(ctx, req)
	require.NoError(t, err)
	require.Len(t, resp.Runs, 1)

	hyperparameters2 := map[string]any{"global_batch_size": 2, "test1": map[string]any{"test2": 5}}

	// Add second experiment
	exp2 := createTestExpWithProjectID(t, api, curUser, projectIDInt)

	task2 := &model.Task{TaskType: model.TaskTypeTrial, TaskID: model.NewTaskID()}
	require.NoError(t, db.AddTask(ctx, task2))
	require.NoError(t, db.AddTrial(ctx, &model.Trial{
		State:        model.PausedState,
		ExperimentID: exp2.ID,
		StartTime:    time.Now(),
		HParams:      hyperparameters2,
	}, task2.TaskID))

	// Sort by start time
	resp, err = api.SearchRuns(ctx, &apiv1.SearchRunsRequest{
		ProjectId: req.ProjectId,
		Sort:      ptrs.Ptr("startTime=asc"),
	})

	require.NoError(t, err)
	require.Equal(t, int32(exp.ID), resp.Runs[0].Experiment.Id)
	require.Equal(t, int32(exp2.ID), resp.Runs[1].Experiment.Id)

	// Sort by hyperparameter
	resp, err = api.SearchRuns(ctx, &apiv1.SearchRunsRequest{
		ProjectId: req.ProjectId,
		Sort:      ptrs.Ptr("hp.global_batch_size=desc"),
	})

	require.NoError(t, err)
	require.Equal(t, int32(exp2.ID), resp.Runs[0].Experiment.Id)
	require.Equal(t, int32(exp.ID), resp.Runs[1].Experiment.Id)

	// Sort by nested hyperparameter
	resp, err = api.SearchRuns(ctx, &apiv1.SearchRunsRequest{
		ProjectId: req.ProjectId,
		Sort:      ptrs.Ptr("hp.test1.test2=desc"),
	})

	require.NoError(t, err)
	require.Equal(t, int32(exp2.ID), resp.Runs[0].Experiment.Id)
	require.Equal(t, int32(exp.ID), resp.Runs[1].Experiment.Id)
}

func TestSearchRunsFilter(t *testing.T) {
	api, curUser, ctx := setupAPITest(t, nil)
	_, projectIDInt := createProjectAndWorkspace(ctx, t, api)
	projectID := int32(projectIDInt)

	// Empty response causes no errors.
	req := &apiv1.SearchRunsRequest{
		ProjectId: &projectID,
		Sort:      ptrs.Ptr("id=asc"),
	}
	resp, err := api.SearchRuns(ctx, req)
	require.NoError(t, err)
	require.Len(t, resp.Runs, 0)

	hyperparameters := map[string]any{"global_batch_size": 1, "test1": map[string]any{"test2": 1}}

	exp := createTestExpWithProjectID(t, api, curUser, projectIDInt)

	task := &model.Task{TaskType: model.TaskTypeTrial, TaskID: model.NewTaskID()}
	require.NoError(t, db.AddTask(ctx, task))
	require.NoError(t, db.AddTrial(ctx, &model.Trial{
		State:        model.PausedState,
		ExperimentID: exp.ID,
		StartTime:    time.Now(),
		HParams:      hyperparameters,
	}, task.TaskID))

	resp, err = api.SearchRuns(ctx, req)
	require.NoError(t, err)
	require.Len(t, resp.Runs, 1)

	hyperparameters2 := map[string]any{"global_batch_size": 2, "test1": map[string]any{"test2": 5}}

	// Add second experiment
	exp2 := createTestExpWithProjectID(t, api, curUser, projectIDInt)

	task2 := &model.Task{TaskType: model.TaskTypeTrial, TaskID: model.NewTaskID()}
	require.NoError(t, db.AddTask(ctx, task2))
	require.NoError(t, db.AddTrial(ctx, &model.Trial{
		State:        model.PausedState,
		ExperimentID: exp2.ID,
		StartTime:    time.Now(),
		HParams:      hyperparameters2,
	}, task2.TaskID))

	tests := map[string]struct {
		expectedNumRuns int
		filter          string
	}{
		"RunColEmpty": {
			expectedNumRuns: 0,
			filter: `{"filterGroup":{"children":[{"columnName":"resourcePool","kind":"field",` +
				`"location":"LOCATION_TYPE_RUN","operator":"isEmpty","type":"COLUMN_TYPE_TEST","value":null}],` +
				`"conjunction":"and","kind":"group"},"showArchived":false}`,
		},
		"RunColNotEmpty": {
			expectedNumRuns: 2,
			filter: `{"filterGroup":{"children":[{"columnName":"resourcePool","kind":"field",` +
				`"location":"LOCATION_TYPE_RUN","operator":"notEmpty","type":"COLUMN_TYPE_TEXT","value":null}],` +
				`"conjunction":"and","kind":"group"},"showArchived":false}`,
		},
		"RunColContains": {
			expectedNumRuns: 2,
			filter: `{"filterGroup":{"children":[{"columnName":"resourcePool","kind":"field",` +
				`"location":"LOCATION_TYPE_RUN","operator":"contains","type":"COLUMN_TYPE_TEXT","value":"kube"}],` +
				`"conjunction":"and","kind":"group"},"showArchived":false}`,
		},
		"RunColNotContains": {
			expectedNumRuns: 0,
			filter: `{"filterGroup":{"children":[{"columnName":"resourcePool","kind":"field",` +
				`"location":"LOCATION_TYPE_RUN","operator":"notContains","type":"COLUMN_TYPE_TEXT","value":"kube"}],` +
				`"conjunction":"and","kind":"group"},"showArchived":false}`,
		},
		"RunColOperator": {
			expectedNumRuns: 1,
			filter: fmt.Sprintf(`{"filterGroup":{"children":[{"columnName":"experimentId","kind":"field",`+
				`"location":"LOCATION_TYPE_RUN","operator":"=","type":"COLUMN_TYPE_NUMBER","value":%d}],`+
				`"conjunction":"and","kind":"group"},"showArchived":false}`, int32(exp2.ID)),
		},
		"HyperParamEmpty": {
			expectedNumRuns: 0,
			filter: `{"filterGroup":{"children":[{"columnName":"hp.global_batch_size","kind":"field",` +
				`"location":"LOCATION_TYPE_RUN_HYPERPARAMETERS","operator":"isEmpty","type":"COLUMN_TYPE_NUMBER","value":1}],` +
				`"conjunction":"and","kind":"group"},"showArchived":false}`,
		},
		"HyperParamNotEmpty": {
			expectedNumRuns: 2,
			filter: `{"filterGroup":{"children":[{"columnName":"hp.global_batch_size","kind":"field",` +
				`"location":"LOCATION_TYPE_RUN_HYPERPARAMETERS","operator":"notEmpty","type":"COLUMN_TYPE_NUMBER","value":1}],` +
				`"conjunction":"and","kind":"group"},"showArchived":false}`,
		},
		"HyperParamContains": {
			expectedNumRuns: 1,
			filter: `{"filterGroup":{"children":[{"columnName":"hp.global_batch_size","kind":"field",` +
				`"location":"LOCATION_TYPE_RUN_HYPERPARAMETERS","operator":"contains","type":"COLUMN_TYPE_NUMBER","value":1}],` +
				`"conjunction":"and","kind":"group"},"showArchived":false}`,
		},
		"HyperParamNotContains": {
			expectedNumRuns: 1,
			filter: `{"filterGroup":{"children":[{"columnName":"hp.global_batch_size","kind":"field",` +
				`"location":"LOCATION_TYPE_RUN_HYPERPARAMETERS","operator":"notContains","type":"COLUMN_TYPE_NUMBER","value":1}],` +
				`"conjunction":"and","kind":"group"},"showArchived":false}`,
		},
		"HyperParamOperator": {
			expectedNumRuns: 1,
			filter: `{"filterGroup":{"children":[{"columnName":"hp.global_batch_size","kind":"field",` +
				`"location":"LOCATION_TYPE_RUN_HYPERPARAMETERS","operator":"<=","type":"COLUMN_TYPE_NUMBER","value":1}],` +
				`"conjunction":"and","kind":"group"},"showArchived":false}`,
		},
		"HyperParamNestedEmpty": {
			expectedNumRuns: 0,
			filter: `{"filterGroup":{"children":[{"columnName":"hp.test1.test2","kind":"field",` +
				`"location":"LOCATION_TYPE_RUN_HYPERPARAMETERS","operator":"isEmpty","type":"COLUMN_TYPE_NUMBER","value":1}],` +
				`"conjunction":"and","kind":"group"},"showArchived":false}`,
		},
		"HyperParamNestedNotEmpty": {
			expectedNumRuns: 2,
			filter: `{"filterGroup":{"children":[{"columnName":"hp.test1.test2","kind":"field",` +
				`"location":"LOCATION_TYPE_RUN_HYPERPARAMETERS","operator":"notEmpty","type":"COLUMN_TYPE_NUMBER","value":1}],` +
				`"conjunction":"and","kind":"group"},"showArchived":false}`,
		},
		"HyperParamNestedContains": {
			expectedNumRuns: 1,
			filter: `{"filterGroup":{"children":[{"columnName":"hp.test1.test2","kind":"field",` +
				`"location":"LOCATION_TYPE_RUN_HYPERPARAMETERS","operator":"contains","type":"COLUMN_TYPE_NUMBER","value":1}],` +
				`"conjunction":"and","kind":"group"},"showArchived":false}`,
		},
		"HyperParamNestedNotContains": {
			expectedNumRuns: 1,
			filter: `{"filterGroup":{"children":[{"columnName":"hp.test1.test2","kind":"field",` +
				`"location":"LOCATION_TYPE_RUN_HYPERPARAMETERS","operator":"notContains","type":"COLUMN_TYPE_NUMBER","value":1}],` +
				`"conjunction":"and","kind":"group"},"showArchived":false}`,
		},
		"HyperParamNestedOperator": {
			expectedNumRuns: 1,
			filter: `{"filterGroup":{"children":[{"columnName":"hp.test1.test2","kind":"field",` +
				`"location":"LOCATION_TYPE_RUN_HYPERPARAMETERS","operator":"<=","type":"COLUMN_TYPE_NUMBER","value":1}],` +
				`"conjunction":"and","kind":"group"},"showArchived":false}`,
		},
	}

	for testCase, testVars := range tests {
		t.Run(testCase, func(t *testing.T) {
			resp, err = api.SearchRuns(ctx, &apiv1.SearchRunsRequest{
				ProjectId: req.ProjectId,
				Filter:    ptrs.Ptr(testVars.filter),
			})

			require.NoError(t, err)
			require.Len(t, resp.Runs, testVars.expectedNumRuns)
		})
	}
}

func TestMoveRunsIds(t *testing.T) {
	api, curUser, ctx := setupAPITest(t, nil)
	_, projectIDInt := createProjectAndWorkspace(ctx, t, api)
	sourceprojectID := int32(1)
	destprojectID := int32(projectIDInt)

	run1, _ := createTestTrial(t, api, curUser)
	run2, _ := createTestTrial(t, api, curUser)

	moveIds := []int32{int32(run1.ID)}

	moveReq := &apiv1.MoveRunsRequest{
		RunIds:               moveIds,
		SourceProjectId:      sourceprojectID,
		DestinationProjectId: destprojectID,
		SkipMultitrial:       false,
	}

	moveResp, err := api.MoveRuns(ctx, moveReq)
	require.NoError(t, err)
	require.Len(t, moveResp.Results, 1)
	require.Equal(t, "", moveResp.Results[0].Error)

	// run no longer in old project
	filter := fmt.Sprintf(`{"filterGroup":{"children":[{"columnName":"experimentId","kind":"field",`+
		`"location":"LOCATION_TYPE_RUN","operator":"=","type":"COLUMN_TYPE_NUMBER","value":%d}],`+
		`"conjunction":"and","kind":"group"},"showArchived":false}`, int32(run2.ExperimentID))
	req := &apiv1.SearchRunsRequest{
		ProjectId: &sourceprojectID,
		Filter:    &filter,
	}
	resp, err := api.SearchRuns(ctx, req)
	require.NoError(t, err)
	require.Len(t, resp.Runs, 1)
	require.Equal(t, int32(run2.ID), resp.Runs[0].Id)

	// runs in new project
	req = &apiv1.SearchRunsRequest{
		ProjectId: &destprojectID,
		Sort:      ptrs.Ptr("id=desc"),
	}

	resp, err = api.SearchRuns(ctx, req)
	require.NoError(t, err)
	require.Len(t, resp.Runs, 1)
	require.Equal(t, moveIds[0], resp.Runs[0].Id)
	i := strings.Index(resp.Runs[0].LocalId, "-")
	localID := resp.Runs[0].LocalId[i+1:]
	require.Equal(t, "1", localID)

	// Experiment in new project
	exp, err := api.getExperiment(ctx, curUser, run1.ExperimentID)
	require.NoError(t, err)
	require.Equal(t, destprojectID, exp.ProjectId)
}

func setUpMultiTrialExperiments(ctx context.Context, t *testing.T, api *apiServer, curUser model.User,
) (int32, int32, int32, int32, int32) {
	_, projectIDInt := createProjectAndWorkspace(ctx, t, api)
	_, projectID2Int := createProjectAndWorkspace(ctx, t, api)
	sourceprojectID := int32(projectIDInt)
	destprojectID := int32(projectID2Int)

	exp := createTestExpWithProjectID(t, api, curUser, projectIDInt)

	task1 := &model.Task{TaskType: model.TaskTypeTrial, TaskID: model.NewTaskID()}
	require.NoError(t, db.AddTask(ctx, task1))
	require.NoError(t, db.AddTrial(ctx, &model.Trial{
		State:        model.PausedState,
		ExperimentID: exp.ID,
		StartTime:    time.Now(),
	}, task1.TaskID))

	task2 := &model.Task{TaskType: model.TaskTypeTrial, TaskID: model.NewTaskID()}
	require.NoError(t, db.AddTask(ctx, task2))
	require.NoError(t, db.AddTrial(ctx, &model.Trial{
		State:        model.PausedState,
		ExperimentID: exp.ID,
		StartTime:    time.Now(),
	}, task2.TaskID))

	req := &apiv1.SearchRunsRequest{
		ProjectId: &sourceprojectID,
		Sort:      ptrs.Ptr("id=asc"),
	}
	resp, err := api.SearchRuns(ctx, req)
	require.NoError(t, err)

	return sourceprojectID, destprojectID, resp.Runs[0].Id, resp.Runs[1].Id, int32(exp.ID)
}

func TestMoveRunsMultiTrialSkip(t *testing.T) {
	api, curUser, ctx := setupAPITest(t, nil)
	sourceprojectID, destprojectID, runID1, runID2, _ := setUpMultiTrialExperiments(ctx, t, api, curUser)

	moveIds := []int32{runID1}

	moveReq := &apiv1.MoveRunsRequest{
		RunIds:               moveIds,
		SourceProjectId:      sourceprojectID,
		DestinationProjectId: destprojectID,
		SkipMultitrial:       true,
	}

	moveResp, err := api.MoveRuns(ctx, moveReq)
	require.NoError(t, err)
	require.Len(t, moveResp.Results, 1)
	require.Equal(t, fmt.Sprintf("Skipping run '%d' (part of multi-trial).", runID1),
		moveResp.Results[0].Error)

	// run still in old project
	req := &apiv1.SearchRunsRequest{
		ProjectId: &sourceprojectID,
		Sort:      ptrs.Ptr("id=asc"),
	}
	resp, err := api.SearchRuns(ctx, req)
	require.NoError(t, err)
	require.Len(t, resp.Runs, 2)
	require.Equal(t, runID1, resp.Runs[0].Id)
	require.Equal(t, runID2, resp.Runs[1].Id)

	// no run in new project
	req = &apiv1.SearchRunsRequest{
		ProjectId: &destprojectID,
		Sort:      ptrs.Ptr("id=asc"),
	}

	resp, err = api.SearchRuns(ctx, req)
	require.NoError(t, err)
	require.Len(t, resp.Runs, 0)
}

func TestMoveRunsMultiTrialNoSkip(t *testing.T) {
	api, curUser, ctx := setupAPITest(t, nil)
	sourceprojectID, destprojectID, runID1, runID2, expID := setUpMultiTrialExperiments(ctx, t, api, curUser)

	moveIds := []int32{runID1}

	moveReq := &apiv1.MoveRunsRequest{
		RunIds:               moveIds,
		SourceProjectId:      sourceprojectID,
		DestinationProjectId: destprojectID,
		SkipMultitrial:       false,
	}

	moveResp, err := api.MoveRuns(ctx, moveReq)
	require.NoError(t, err)
	require.Len(t, moveResp.Results, 1)
	require.Equal(t, "", moveResp.Results[0].Error)

	// runs no longer in old project
	req := &apiv1.SearchRunsRequest{
		ProjectId: &sourceprojectID,
		Sort:      ptrs.Ptr("id=asc"),
	}
	resp, err := api.SearchRuns(ctx, req)
	require.NoError(t, err)
	require.Len(t, resp.Runs, 0)

	// runs in new project
	req = &apiv1.SearchRunsRequest{
		ProjectId: &destprojectID,
		Sort:      ptrs.Ptr("id=asc"),
	}

	resp, err = api.SearchRuns(ctx, req)
	require.NoError(t, err)
	require.Len(t, resp.Runs, 2)
	// Check if other run moved as well
	require.Equal(t, runID2, resp.Runs[1].Id)
	// Check if runs in same experiment
	require.Equal(t, expID, resp.Runs[0].Experiment.Id)
	require.Equal(t, expID, resp.Runs[1].Experiment.Id)

	i := strings.Index(resp.Runs[0].LocalId, "-")
	localID1 := resp.Runs[0].LocalId[i+1:]
	i = strings.Index(resp.Runs[1].LocalId, "-")
	localID2 := resp.Runs[1].LocalId[i+1:]
	require.Equal(t, "1", localID1)
	require.Equal(t, "2", localID2)
}

func TestMoveRunsFilter(t *testing.T) {
	api, curUser, ctx := setupAPITest(t, nil)
	_, projectIDInt := createProjectAndWorkspace(ctx, t, api)
	_, projectID2Int := createProjectAndWorkspace(ctx, t, api)
	sourceprojectID := int32(projectIDInt)
	destprojectID := int32(projectID2Int)

	exp1 := createTestExpWithProjectID(t, api, curUser, projectIDInt)
	exp2 := createTestExpWithProjectID(t, api, curUser, projectIDInt)

	hyperparameters1 := map[string]any{"global_batch_size": 1, "test1": map[string]any{"test2": 1}}

	task1 := &model.Task{TaskType: model.TaskTypeTrial, TaskID: model.NewTaskID()}
	require.NoError(t, db.AddTask(ctx, task1))
	require.NoError(t, db.AddTrial(ctx, &model.Trial{
		State:        model.PausedState,
		ExperimentID: exp1.ID,
		StartTime:    time.Now(),
		HParams:      hyperparameters1,
	}, task1.TaskID))

	hyperparameters2 := map[string]any{"global_batch_size": 1, "test1": map[string]any{"test2": 5}}
	task2 := &model.Task{TaskType: model.TaskTypeTrial, TaskID: model.NewTaskID()}
	require.NoError(t, db.AddTask(ctx, task2))
	require.NoError(t, db.AddTrial(ctx, &model.Trial{
		State:        model.PausedState,
		ExperimentID: exp2.ID,
		StartTime:    time.Now(),
		HParams:      hyperparameters2,
	}, task2.TaskID))

	req := &apiv1.SearchRunsRequest{
		ProjectId: &sourceprojectID,
		Sort:      ptrs.Ptr("id=asc"),
	}
	resp, err := api.SearchRuns(ctx, req)
	require.NoError(t, err)

	// If provided with filter MoveRuns should ignore these move ids
	moveIds := []int32{resp.Runs[0].Id, resp.Runs[1].Id}

	moveReq := &apiv1.MoveRunsRequest{
		RunIds:               moveIds,
		SourceProjectId:      sourceprojectID,
		DestinationProjectId: destprojectID,
		Filter: ptrs.Ptr(`{"filterGroup":{"children":[{"columnName":"hp.test1.test2","kind":"field",` +
			`"location":"LOCATION_TYPE_RUN_HYPERPARAMETERS","operator":"<=","type":"COLUMN_TYPE_NUMBER","value":1}],` +
			`"conjunction":"and","kind":"group"},"showArchived":false}`),
		SkipMultitrial: false,
	}

	moveResp, err := api.MoveRuns(ctx, moveReq)
	require.NoError(t, err)
	require.Len(t, moveResp.Results, 1)
	require.Equal(t, "", moveResp.Results[0].Error)

	// check 1 run moved in old project
	resp, err = api.SearchRuns(ctx, req)
	require.NoError(t, err)
	require.Len(t, resp.Runs, 1)

	// run in new project
	req = &apiv1.SearchRunsRequest{
		ProjectId: &destprojectID,
		Sort:      ptrs.Ptr("id=asc"),
	}

	resp, err = api.SearchRuns(ctx, req)
	require.NoError(t, err)
	require.Len(t, resp.Runs, 1)
	i := strings.Index(resp.Runs[0].LocalId, "-")
	localID := resp.Runs[0].LocalId[i+1:]
	require.Equal(t, "1", localID)
}
