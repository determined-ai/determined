package scheduler

import (
	"github.com/determined-ai/determined/master/pkg/agent"
	cproto "github.com/determined-ai/determined/master/pkg/container"
	"github.com/determined-ai/determined/master/pkg/device"
	"github.com/determined-ai/determined/master/pkg/model"
	image "github.com/determined-ai/determined/master/pkg/tasks"
)

// Assigned is a message that tells the task actor that it has been assigned to the provided
// agent.
// TODO: Expose assignment information (e.g. device type, num slots) to task actors.
type Assigned struct {
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
func (a *Assigned) StartTask(spec image.TaskSpec) TaskSummary {
	handler := a.agent.handler
	spec.ClusterID = a.clusterID
	spec.TaskID = string(a.task.ID)
	spec.HarnessPath = a.harnessPath
	spec.TaskContainerDefaults = a.taskContainerDefaults
	spec.Devices = a.devices
	handler.System().Tell(handler, StartTask{
		Task: a.task.handler,
		StartContainer: agent.StartContainer{
			Container: cproto.Container{
				Parent:      a.task.handler.Address(),
				ID:          cproto.ID(a.container.id),
				State:       cproto.Assigned,
				Devices:     a.devices,
				Recoverable: spec.Recoverable,
			},
			Spec: image.ToContainerSpec(spec),
		},
	})
	return newTaskSummary(a.task)
}

// IsLeader returns true if this assignment corresponds to the leader container of the task.
func (a *Assigned) IsLeader() bool {
	return a.container.IsLeader()
}

// NumContainers returns the number of containers to which the task has been assigned.
func (a *Assigned) NumContainers() int {
	return a.numContainers
}
