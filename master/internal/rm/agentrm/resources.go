package agentrm

import (
	"context"
	"fmt"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/aproto"
	"github.com/determined-ai/determined/master/pkg/cproto"
	"github.com/determined-ai/determined/master/pkg/device"
	"github.com/determined-ai/determined/master/pkg/logger"
	"github.com/determined-ai/determined/master/pkg/tasks"
)

// containerResources contains information for tasks have been allocated but not yet started.
type containerResources struct {
	req         *sproto.AllocateRequest
	agent       *agentState
	devices     []device.Device
	containerID cproto.ID
	started     *sproto.ResourcesStarted
	exited      *sproto.ResourcesStopped
}

// Summary summarizes a container allocation.
func (c containerResources) Summary() sproto.ResourcesSummary {
	return sproto.ResourcesSummary{
		ResourcesID:   sproto.ResourcesID(c.containerID),
		ResourcesType: sproto.ResourcesTypeDockerContainer,
		AllocationID:  c.req.AllocationID,
		AgentDevices: map[aproto.ID][]device.Device{
			aproto.ID(c.agent.Handler.Address().Local()): c.devices,
		},

		ContainerID: &c.containerID,
		Started:     c.started,
		Exited:      c.exited,
	}
}

// StartContainer notifies the agent to start a container.
func (c containerResources) Start(
	// TODO(!!!): Remove this `*actor.System`, either by refactor the RM to not be an actor or by
	// adding a global system before the project's end.
	ctx *actor.System, logCtx logger.Context, spec tasks.TaskSpec, rri sproto.ResourcesRuntimeInfo,
) error {
	handler := c.agent.Handler
	spec.ContainerID = string(c.containerID)
	spec.ResourcesID = string(c.containerID)
	spec.AllocationID = string(c.req.AllocationID)
	spec.AllocationSessionToken = rri.Token
	spec.TaskID = string(c.req.TaskID)
	if spec.LoggingFields == nil {
		spec.LoggingFields = map[string]string{}
	}
	spec.LoggingFields["allocation_id"] = spec.AllocationID
	spec.LoggingFields["task_id"] = spec.TaskID
	spec.ExtraEnvVars[sproto.ResourcesTypeEnvVar] = string(sproto.ResourcesTypeDockerContainer)
	spec.UseHostMode = rri.IsMultiAgent
	spec.Devices = c.devices

	return ctx.Ask(handler, sproto.StartTaskContainer{
		AllocationID: c.req.AllocationID,
		StartContainer: aproto.StartContainer{
			Container: cproto.Container{
				ID:          c.containerID,
				State:       cproto.Assigned,
				Devices:     c.devices,
				Description: c.req.Name,
			},
			Spec: spec.ToDockerSpec(),
		},
		LogContext: logCtx,
	}).Error()
}

// Kill notifies the agent to kill the container.
func (c containerResources) Kill(ctx *actor.System, logCtx logger.Context) {
	ctx.Tell(c.agent.Handler, sproto.KillTaskContainer{
		ContainerID: c.containerID,
		LogContext:  logCtx,
	})
}

func (c containerResources) persist() error {
	summary := c.Summary()

	agentID, _, ok := Single(summary.AgentDevices)
	if !ok {
		return fmt.Errorf("%d agents in containerResources summary", len(summary.AgentDevices))
	}

	snapshot := containerSnapshot{
		ResourceID: summary.ResourcesID,
		ID:         c.containerID,
		AgentID:    agentID,
	}
	_, err := db.Bun().NewInsert().Model(&snapshot).Exec(context.TODO())
	return err
}

// Single asserts there's a single element in the map and take it.
func Single[K comparable, V any](m map[K]V) (kr K, vr V, ok bool) {
	// TODO(ilia): move it into a shared utilities package when
	// it'll be used elsewhere.
	if len(m) != 1 {
		return kr, vr, false
	}
	for k, v := range m {
		kr = k
		vr = v
	}
	return kr, vr, true
}
