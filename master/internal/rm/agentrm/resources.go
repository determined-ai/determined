package agentrm

import (
	"context"
	"fmt"
	"strconv"

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
	Req         *sproto.AllocateRequest
	Agent       *AgentState
	Devices     []device.Device
	ContainerID cproto.ID
	Started     *sproto.ResourcesStarted
	Exited      *sproto.ResourcesStopped
}

// Summary summarizes a container allocation.
func (c containerResources) Summary() sproto.ResourcesSummary {
	return sproto.ResourcesSummary{
		ResourcesID:   sproto.ResourcesID(c.ContainerID),
		ResourcesType: sproto.ResourcesTypeDockerContainer,
		AllocationID:  c.Req.AllocationID,
		AgentDevices: map[aproto.ID][]device.Device{
			aproto.ID(c.Agent.Handler.Address().Local()): c.Devices,
		},

		ContainerID: &c.ContainerID,
		Started:     c.Started,
		Exited:      c.Exited,
	}
}

// StartContainer notifies the agent to start a container.
func (c containerResources) Start(
	ctx *actor.Context, logCtx logger.Context, spec tasks.TaskSpec, rri sproto.ResourcesRuntimeInfo,
) error {
	handler := c.Agent.Handler
	spec.ContainerID = string(c.ContainerID)
	spec.ResourcesID = string(c.ContainerID)
	spec.AllocationID = string(c.Req.AllocationID)
	spec.AllocationSessionToken = rri.Token
	spec.TaskID = string(c.Req.TaskID)
	if spec.LoggingFields == nil {
		spec.LoggingFields = map[string]string{}
	}
	spec.LoggingFields["allocation_id"] = spec.AllocationID
	spec.LoggingFields["task_id"] = spec.TaskID
	spec.ExtraEnvVars[sproto.ResourcesTypeEnvVar] = string(sproto.ResourcesTypeDockerContainer)
	spec.UseHostMode = rri.IsMultiAgent
	spec.Devices = c.Devices
	// Write the real DET_UNIQUE_PORT_OFFSET value now that we know which devices to use.
	spec.ExtraEnvVars["DET_UNIQUE_PORT_OFFSET"] = strconv.Itoa(tasks.UniquePortOffset(spec.Devices))
	return ctx.Ask(handler, sproto.StartTaskContainer{
		TaskActor: c.Req.AllocationRef,
		StartContainer: aproto.StartContainer{
			Container: cproto.Container{
				Parent:  c.Req.AllocationRef.Address(),
				ID:      c.ContainerID,
				State:   cproto.Assigned,
				Devices: c.Devices,
			},
			Spec: spec.ToDockerSpec(),
		},
		LogContext: logCtx,
	}).Error()
}

// Kill notifies the agent to kill the container.
func (c containerResources) Kill(ctx *actor.Context, logCtx logger.Context) {
	ctx.Tell(c.Agent.Handler, sproto.KillTaskContainer{
		ContainerID: c.ContainerID,
		LogContext:  logCtx,
	})
}

func (c containerResources) persist() error {
	summary := c.Summary()

	agentID, _, ok := Single(summary.AgentDevices)
	if !ok {
		return fmt.Errorf("%d agents in containerResources summary", len(summary.AgentDevices))
	}

	snapshot := ContainerSnapshot{
		ResourceID: summary.ResourcesID,
		ID:         c.ContainerID,
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
