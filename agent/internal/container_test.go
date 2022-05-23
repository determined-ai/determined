package internal

import (
	"testing"

	"gotest.tools/assert"

	"github.com/docker/docker/api/types/container"

	"github.com/determined-ai/determined/master/pkg/cproto"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
)

func TestGetBaseTaskLog(t *testing.T) {
	agentID := "agent-id"
	containerID := "container-id"
	taskID := "task-id"
	allocationID := "allocation-id"
	spec := &cproto.Spec{
		RunSpec: cproto.RunSpec{
			ContainerConfig: container.Config{
				Env: []string{
					"test",        // Should not panic.
					agentIDEnvVar, // Should be ignored.
					agentIDEnvVar + "=" + agentID,
					containerIDEnvVar + "=" + containerID,
					taskIDEnvVar + "=" + taskID,
					allocationIDEnvVar + "=" + allocationID,
					taskIDEnvVar, // Should also be ignored.
				},
			},
		},
	}

	expected := model.TaskLog{
		Level:        ptrs.Ptr("INFO"),
		StdType:      ptrs.Ptr("stdout"),
		AgentID:      &agentID,
		ContainerID:  &containerID,
		TaskID:       taskID,
		AllocationID: &allocationID,
	}
	assert.DeepEqual(t, getBaseTaskLog(spec), expected)
}
