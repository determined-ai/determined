//go:build integration
// +build integration

package internal

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/taskv1"
)

func TestGetGenericTaskConfig(t *testing.T) {
	api, _, ctx := setupAPITest(t, nil)

	expectedConfig := "{\"test_config\": \"val\"}"
	taskID := model.NewTaskID()

	task := &model.Task{TaskType: model.TaskTypeGeneric, TaskID: taskID, Config: &expectedConfig}
	require.NoError(t, db.AddTask(ctx, task))

	resp, err := api.GetGenericTaskConfig(ctx, &apiv1.GetGenericTaskConfigRequest{
		TaskId: string(taskID),
	})

	require.NoError(t, err)
	require.Equal(t, expectedConfig, resp.Config)
}

func TestGetTask(t *testing.T) {
	api, _, ctx := setupAPITest(t, nil)

	taskID := model.NewTaskID()
	state := model.TaskStateCompleted

	task := &model.Task{TaskType: model.TaskTypeGeneric, TaskID: taskID, State: &state}
	require.NoError(t, db.AddTask(ctx, task))

	resp, err := api.GetTask(ctx, &apiv1.GetTaskRequest{TaskId: taskID.String()})

	require.NoError(t, err)
	require.Equal(t, taskv1.GenericTaskState_GENERIC_TASK_STATE_COMPLETED, *resp.Task.TaskState)
	require.Equal(t, taskID.String(), resp.Task.TaskId)
	require.Equal(t, taskv1.TaskType_TASK_TYPE_GENERIC, resp.Task.TaskType)
}
