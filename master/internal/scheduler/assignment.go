package scheduler

import (
	"github.com/determined-ai/determined/master/pkg/agent"
	cproto "github.com/determined-ai/determined/master/pkg/container"
	"github.com/determined-ai/determined/master/pkg/device"
	"github.com/determined-ai/determined/master/pkg/model"
	sproto "github.com/determined-ai/determined/master/pkg/scheduler"
	image "github.com/determined-ai/determined/master/pkg/tasks"
)

// assignment contains information for tasks have been assigned but not yet started.
// TODO: Expose assignment information (e.g. device type, num slots) to task actors.
type assignment struct {
	task                  *Task
	container             *container
	agent                 *agentState
	clusterID             string
	numContainers         int
	devices               []device.Device
	harnessPath           string
	taskContainerDefaults model.TaskContainerDefaultsConfig
}

// StartTask notifies the agent that the task is ready to start with the provided task spec.
func (a *assignment) StartTask(spec image.TaskSpec) TaskSummary {
	handler := a.agent.handler
	spec.ClusterID = a.clusterID
	spec.TaskID = string(a.task.ID)
	spec.HarnessPath = a.harnessPath
	spec.TaskContainerDefaults = a.taskContainerDefaults
	spec.Devices = a.devices
	handler.System().Tell(handler, sproto.StartTaskOnAgent{
		Task: a.task.handler,
		StartContainer: agent.StartContainer{
			Container: cproto.Container{
				Parent:  a.task.handler.Address(),
				ID:      cproto.ID(a.container.id),
				State:   cproto.Assigned,
				Devices: a.devices,
			},
			Spec: image.ToContainerSpec(spec),
		},
	})
	return newTaskSummary(a.task)
}
