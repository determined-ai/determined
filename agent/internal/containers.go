package internal

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"syscall"

	"github.com/docker/docker/api/types"
	dcontainer "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/labstack/echo"
	"github.com/pkg/errors"
	"golang.org/x/sys/unix"

	"github.com/determined-ai/determined/master/pkg/actor"
	proto "github.com/determined-ai/determined/master/pkg/agent"
	"github.com/determined-ai/determined/master/pkg/container"
	cproto "github.com/determined-ai/determined/master/pkg/container"
	"github.com/determined-ai/determined/master/pkg/device"
)

const (
	dockerContainerTypeLabel    = "ai.determined.container.type"
	dockerContainerTypeValue    = "task-container"
	dockerContainerIDLabel      = "ai.determined.container.id"
	dockerContainerParentLabel  = "ai.determined.container.parent"
	dockerContainerDevicesLabel = "ai.determined.container.devices"
	dockerRecoverableLabel      = "ai.determined.container.recoverable"
	dockerAgentLabel            = "ai.determined.container.agent"
	dockerClusterLabel          = "ai.determined.container.cluster"
	dockerMasterLabel           = "ai.determined.container.master"
)

type containerManager struct {
	Options       Options           `json:"-"`
	MasterInfo    proto.MasterInfo  `json:"-"`
	GlobalEnvVars []string          `json:"global_env_vars"`
	Labels        map[string]string `json:"labels"`
	Devices       []device.Device   `json:"devices"`

	docker *client.Client
}

type recoverContainers struct{}

func (c *containerManager) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart:
		d, err := client.NewClientWithOpts(client.FromEnv)
		if err != nil {
			return err
		}
		c.docker = d

		masterHost := c.Options.ContainerMasterHost
		if masterHost == "" {
			masterHost = c.Options.MasterHost
		}

		masterPort := c.Options.ContainerMasterPort
		if masterPort == 0 {
			masterPort = c.Options.MasterPort
		}

		c.GlobalEnvVars = []string{
			fmt.Sprintf("DET_CLUSTER_ID=%s", c.MasterInfo.ClusterID),
			fmt.Sprintf("DET_MASTER_ID=%s", c.MasterInfo.MasterID),
			fmt.Sprintf("DET_MASTER=%s:%d", masterHost, masterPort),
			fmt.Sprintf("DET_MASTER_HOST=%s", masterHost),
			fmt.Sprintf("DET_MASTER_ADDR=%s", masterHost),
			fmt.Sprintf("DET_MASTER_PORT=%d", masterPort),
			fmt.Sprintf("DET_AGENT_ID=%s", c.Options.AgentID),
		}
		c.Labels = map[string]string{
			dockerContainerTypeLabel: dockerContainerTypeValue,
			dockerAgentLabel:         c.Options.AgentID,
			dockerClusterLabel:       c.MasterInfo.ClusterID,
			dockerMasterLabel:        c.MasterInfo.MasterID,
		}

	case recoverContainers:
		containers, err := c.recoverContainers(ctx)
		if err != nil {
			return errors.Wrap(err, "Error attempting to recover prior containers")
		}
		ctx.Respond(containers)

	case proto.ContainerLog, proto.ContainerStateChanged:
		ctx.Tell(ctx.Self().Parent(), msg)

	case proto.StartContainer:
		msg.Spec = c.overwriteSpec(msg.Container, msg.Spec)
		if ref, ok := ctx.ActorOf(msg.Container.ID, newContainerActor(msg, c.docker)); !ok {
			ctx.Log().Warnf("container already created: %s", msg.Container.ID)
			if ctx.ExpectingResponse() {
				ctx.Respond(errors.Errorf("container already created: %s", msg.Container.ID))
			}
		} else if ctx.ExpectingResponse() {
			ctx.Respond(ctx.Ask(ref, getContainerSummary{}))
		}

	case proto.SignalContainer:
		if ref := ctx.Child(msg.ContainerID); ref != nil {
			ctx.Tell(ref, msg)
		} else {
			ctx.Log().Warnf("error signaling container with %s, container not found: %s",
				msg.Signal, msg.ContainerID)
		}

	case echo.Context:
		c.handleAPIRequest(ctx, msg)

	case actor.PostStop:
		ctx.Log().Info("container manager shut down")

	default:
		return actor.ErrUnexpectedMessage(ctx)
	}
	return nil
}

func (c *containerManager) handleAPIRequest(ctx *actor.Context, apiCtx echo.Context) {
	switch apiCtx.Request().Method {
	case echo.GET:
		ctx.Respond(apiCtx.JSON(http.StatusOK,
			ctx.AskAll(getContainerSummary{}, ctx.Children()...)))
	default:
		ctx.Respond(echo.ErrMethodNotAllowed)
	}
}

func (c *containerManager) overwriteSpec(
	cont cproto.Container, spec container.Spec,
) container.Spec {
	spec.RunSpec.HostConfig.AutoRemove = true
	spec.RunSpec.ContainerConfig.Env = append(
		spec.RunSpec.ContainerConfig.Env, c.GlobalEnvVars...)
	spec.RunSpec.ContainerConfig.Env = append(
		spec.RunSpec.ContainerConfig.Env, c.containerEnvVars(cont)...)
	if spec.RunSpec.ContainerConfig.Labels == nil {
		spec.RunSpec.ContainerConfig.Labels = make(map[string]string)
	}
	for key, value := range c.Labels {
		spec.RunSpec.ContainerConfig.Labels[key] = value
	}
	spec.RunSpec.ContainerConfig.Labels[dockerContainerIDLabel] = cont.ID.String()
	spec.RunSpec.ContainerConfig.Labels[dockerContainerParentLabel] = cont.Parent.String()
	spec.RunSpec.ContainerConfig.Labels[dockerRecoverableLabel] = strconv.FormatBool(cont.Recoverable)
	var slotIds []string
	for _, d := range cont.Devices {
		slotIds = append(slotIds, strconv.Itoa(d.ID))
	}
	spec.RunSpec.ContainerConfig.Labels[dockerContainerDevicesLabel] = strings.Join(slotIds, ",")

	spec.RunSpec.HostConfig.DeviceRequests = append(
		spec.RunSpec.HostConfig.DeviceRequests, c.gpuDeviceRequests(cont)...)

	return spec
}

func (c *containerManager) gpuDeviceRequests(cont cproto.Container) []dcontainer.DeviceRequest {
	gpuUUIDs := cont.GPUDeviceUUIDs()
	if len(gpuUUIDs) == 0 {
		return nil
	}
	return []dcontainer.DeviceRequest{
		{
			Driver:       "nvidia",
			Capabilities: [][]string{{"gpu", "compute", "utility"}},
			DeviceIDs:    gpuUUIDs,
		},
	}
}

func (c *containerManager) containerEnvVars(cont cproto.Container) []string {
	var slotIds []string
	for _, d := range cont.Devices {
		slotIds = append(slotIds, strconv.Itoa(d.ID))
	}
	return []string{
		fmt.Sprintf("DET_CONTAINER_ID=%s", cont.ID),
		fmt.Sprintf("DET_SLOT_IDS=[%s]", strings.Join(slotIds, ",")),
		fmt.Sprintf("DET_USE_GPU=%t", len(cont.GPUDeviceUUIDs()) > 0),
	}
}

func (c *containerManager) recoverContainers(
	ctx *actor.Context) ([]proto.ContainerRecovered, error) {
	ctx.Log().Info("attempting to recover prior containers")
	options := types.ContainerListOptions{
		Filters: filters.NewArgs(),
	}
	options.Filters.Add("label", fmt.Sprintf("%s=%s",
		dockerContainerTypeLabel, dockerContainerTypeValue))
	options.Filters.Add("label", fmt.Sprintf("%s=%s", dockerAgentLabel, c.Options.AgentID))
	options.Filters.Add("label", fmt.Sprintf("%s=%s", dockerClusterLabel, c.MasterInfo.ClusterID))
	options.Filters.Add("label", fmt.Sprintf("%s=true", dockerRecoverableLabel))

	dockerContainers, err := c.docker.ContainerList(context.Background(), options)
	if err != nil {
		return nil, errors.Wrap(err, "error listing out containers to recover")
	}
	var containers []proto.ContainerRecovered
	for _, cont := range dockerContainers {
		if cont.State != "running" {
			ctx.Log().Warnf("killing container found in %s state: %s", cont.State, cont.ID)
			c.killContainer(ctx, cont)
			continue
		}
		ctx.Log().Infof("attempting to recover docker container: %s", cont.ID)
		if parsedContainer, err := c.recoverContainer(ctx, cont); err != nil {
			ctx.Log().WithError(err).Warnf("error recovering container %s, attempting to kill", cont.ID)
			c.killContainer(ctx, cont)
		} else {
			containers = append(containers, parsedContainer)
		}
	}
	ctx.Log().Infof("finished recovering prior containers")
	return containers, nil
}

func (c *containerManager) recoverContainer(
	ctx *actor.Context, dCont types.Container) (proto.ContainerRecovered, error) {
	recovery := proto.ContainerRecovered{}
	deviceIds := dCont.Labels[dockerContainerDevicesLabel]
	var devices []device.Device
	expectedDevices := strings.Split(deviceIds, ",")
	for _, idStr := range expectedDevices {
		id, err := strconv.Atoi(idStr)
		if err != nil {
			return recovery, errors.Wrapf(err, "error parsing device id")
		}
		for _, d := range c.Devices {
			if d.ID == id {
				devices = append(devices, d)
			}
		}
	}
	if len(expectedDevices) != len(devices) {
		return recovery, errors.New("not enough devices registered with agent")
	}
	parentID, ok := dCont.Labels[dockerContainerParentLabel]
	if !ok {
		return recovery, errors.New("no parent id found for container")
	}
	parent := actor.Address{}
	if err := parent.UnmarshalText([]byte(parentID)); err != nil {
		return recovery, errors.Wrap(err, "malformed container parent id")
	}
	containerID, ok := dCont.Labels[dockerContainerIDLabel]
	if !ok {
		return recovery, errors.New("no container id found for container")
	}
	cont := cproto.Container{
		Parent:  parent,
		ID:      cproto.ID(containerID),
		State:   cproto.Running,
		Devices: devices,
	}
	info, err := c.docker.ContainerInspect(context.Background(), dCont.ID)
	if err != nil {
		return recovery, errors.New("error retrieving container info from docker")
	}
	if _, ok := ctx.ActorOf(cont.ID, recoverContainerActor(cont, info, c.docker)); !ok {
		ctx.Log().Warnf("container already created: %s", cont.ID)
	}
	recovery.Container = cont
	recovery.ContainerStarted = proto.ContainerStarted{
		ContainerInfo: info,
	}
	return recovery, nil
}

func (c *containerManager) killContainer(ctx *actor.Context, dCont types.Container) {
	// TODO (DET-2725): Ensure container is removed
	err := c.docker.ContainerKill(context.Background(), dCont.ID, unix.SignalName(syscall.SIGKILL))
	if err != nil {
		ctx.Log().WithError(err).Warnf("error killing rouge container: %s", dCont.ID)
	}
}
