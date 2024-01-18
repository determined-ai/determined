//go:build integration
// +build integration

package internal

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

func TestGetGenericTaskConfig(t *testing.T) {
	api, _, ctx := setupAPITest(t, nil)

	expectedConfig := "{\"test_config\": \"val\"}"
	taskID := model.NewTaskID()

	task := &model.Task{TaskType: model.TaskTypeGeneric, TaskID: taskID, Config: &expectedConfig}
	require.NoError(t, api.m.db.AddTask(task))

	resp, err := api.GetGenericTaskConfig(ctx, &apiv1.GetGenericTaskConfigRequest{
		TaskId: string(taskID),
	})

	require.NoError(t, err)
	require.Equal(t, expectedConfig, resp.Config)
}
