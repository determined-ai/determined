//go:build integration
// +build integration

package internal

import (
	"fmt"
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
