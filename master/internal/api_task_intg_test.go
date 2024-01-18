//go:build integration
// +build integration

package internal

import (
	"testing"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/stretchr/testify/require"
)

func TestGetGenericTaskConfig(t *testing.T) {
	api, _, ctx := setupAPITest(t, nil)

	expected_config := "{\"test_config\": \"val\"}"
	task_id := model.NewTaskID()

	task := &model.Task{TaskType: model.TaskTypeGeneric, TaskID: task_id, Config: &expected_config}
	require.NoError(t, api.m.db.AddTask(task))

	resp, err := api.GetGenericTaskConfig(ctx, &apiv1.GetGenericTaskConfigRequest{
		TaskId: string(task_id),
	})

	require.NoError(t, err)
	require.Equal(t, expected_config, resp.Config)
}
