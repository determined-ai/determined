package scheduler

import (
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/agent"
	cproto "github.com/determined-ai/determined/master/pkg/container"
	"github.com/determined-ai/determined/master/pkg/device"
	"github.com/determined-ai/determined/master/pkg/model"
	image "github.com/determined-ai/determined/master/pkg/tasks"
)

// containerAssignment contains information for tasks have been assigned but not yet started.
type containerAssignment struct {
	task                  *Task
	container             *container
	agent                 *agentState
	clusterID             string
	devices               []device.Device
	harnessPath           string
	taskContainerDefaults model.TaskContainerDefaultsConfig
}

// StartTask notifies the agent that the task is ready to start with the provided task spec.
func (c *containerAssignment) StartTask(spec image.TaskSpec) {
	handler := c.agent.handler
	spec.ClusterID = c.clusterID
	spec.TaskID = string(c.task.ID)
	spec.HarnessPath = c.harnessPath
	spec.TaskContainerDefaults = c.taskContainerDefaults
	spec.Devices = c.devices
	handler.System().Tell(handler, sproto.StartTaskOnAgent{
		Task: c.task.handler,
		StartContainer: agent.StartContainer{
			Container: cproto.Container{
				Parent:  c.task.handler.Address(),
				ID:      cproto.ID(c.container.id),
				State:   cproto.Assigned,
				Devices: c.devices,
			},
			Spec: image.ToContainerSpec(spec),
		},
	})
}

type podAssignment struct {
	task                  *Task
	container             *container
	agent                 *agentState
	clusterID             string
	harnessPath           string
	taskContainerDefaults model.TaskContainerDefaultsConfig
}

func (p *podAssignment) StartTask(spec image.TaskSpec) {
	//TODO: fill this in.
}
