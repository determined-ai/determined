package internal

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/docker/docker/api/types"
	dcontainer "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"golang.org/x/sys/unix"

	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/aproto"
	"github.com/determined-ai/determined/master/pkg/cproto"
	"github.com/determined-ai/determined/master/pkg/device"
	"github.com/determined-ai/determined/master/pkg/model"
)

const (
	dockerContainerTypeLabel    = "ai.determined.container.type"
	dockerContainerTypeValue    = "task-container"
	dockerContainerIDLabel      = "ai.determined.container.id"
	dockerContainerParentLabel  = "ai.determined.container.parent"
	dockerContainerDevicesLabel = "ai.determined.container.devices"
	dockerAgentLabel            = "ai.determined.container.agent"
	dockerClusterLabel          = "ai.determined.container.cluster"
	dockerMasterLabel           = "ai.determined.container.master"
	dockerContainerVersionLabel = "ai.determined.container.version"
	dockerContainerVersionValue = "0"
)

type containerManager struct {
	Options       Options           `json:"-"`
	MasterInfo    aproto.MasterInfo `json:"-"`
	GlobalEnvVars []string          `json:"global_env_vars"`
	Labels        map[string]string `json:"labels"`
	Devices       []device.Device   `json:"devices"`

	fluentPort int
	docker     *client.Client
}

type (
	requestReattachContainers struct {
		ContainersToReattach []aproto.ContainerReattach
	}
	responseReattachContainers struct {
		ContainersReattached []aproto.ContainerReattachAck
	}
)

func newContainerManager(a *agent, fluentPort int) (*containerManager, error) {
	return &containerManager{
		MasterInfo: a.MasterSetAgentOptions.MasterInfo,
		Options:    a.Options,
		Devices:    a.Devices,
		fluentPort: fluentPort,
	}, nil
}

func (c *containerManager) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart:
		d, err := client.NewClientWithOpts(client.FromEnv)
		if err != nil {
			return err
		}
		c.docker = d

		masterScheme := httpInsecureScheme
		if c.Options.Security.TLS.Enabled {
			masterScheme = httpSecureScheme
		}

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
			fmt.Sprintf("DET_MASTER=%s://%s:%d", masterScheme, masterHost, masterPort),
			fmt.Sprintf("DET_MASTER_HOST=%s", masterHost),
			fmt.Sprintf("DET_MASTER_ADDR=%s", masterHost),
			fmt.Sprintf("DET_MASTER_PORT=%d", masterPort),
			fmt.Sprintf("DET_AGENT_ID=%s", c.Options.AgentID),
		}

		if a := c.Options.Security.TLS.MasterCertName; a != "" {
			c.GlobalEnvVars = append(c.GlobalEnvVars, fmt.Sprintf("DET_MASTER_CERT_NAME=%s", a))
		}

		c.Labels = map[string]string{
			dockerContainerTypeLabel: dockerContainerTypeValue,
			dockerAgentLabel:         c.Options.AgentID,
			dockerClusterLabel:       c.MasterInfo.ClusterID,
			dockerMasterLabel:        c.MasterInfo.MasterID,
		}
	case requestReattachContainers:
		reattachedContainers, err := c.reattachContainers(ctx, msg.ContainersToReattach)
		if err != nil {
			ctx.Log().WithError(err).Warn("failed to reattach containers")
			ctx.Respond(responseReattachContainers{})
		} else {
			ctx.Respond(responseReattachContainers{ContainersReattached: reattachedContainers})
		}

	case aproto.ContainerLog, aproto.ContainerStateChanged, model.TaskLog:
		ctx.Tell(ctx.Self().Parent(), msg)

	case aproto.StartContainer:
		msg.Spec = c.overwriteSpec(msg.Container, msg.Spec)
		if ref, ok := ctx.ActorOf(msg.Container.ID, newContainerActor(msg, c.docker)); !ok {
			ctx.Log().Warnf("container already created: %s", msg.Container.ID)
			if ctx.ExpectingResponse() {
				ctx.Respond(errors.Errorf("container already created: %s", msg.Container.ID))
			}
		} else if ctx.ExpectingResponse() {
			ctx.Respond(ctx.Ask(ref, getContainerSummary{}))
		}

	case aproto.SignalContainer:
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

func (c *containerManager) overwriteSpec(cont cproto.Container, spec cproto.Spec) cproto.Spec {
	return overwriteSpec(cont, spec, c.GlobalEnvVars, c.Labels, c.fluentPort)
}

func overwriteSpec(
	cont cproto.Container,
	spec cproto.Spec,
	globalEnvVars []string,
	labels map[string]string,
	fluentPort int,
) cproto.Spec {
	spec.RunSpec.HostConfig.AutoRemove = true
	spec.RunSpec.ContainerConfig.Env = append(
		spec.RunSpec.ContainerConfig.Env, globalEnvVars...)
	spec.RunSpec.ContainerConfig.Env = append(
		spec.RunSpec.ContainerConfig.Env, containerEnvVars(cont)...)
	if spec.RunSpec.ContainerConfig.Labels == nil {
		spec.RunSpec.ContainerConfig.Labels = make(map[string]string)
	}
	for key, value := range labels {
		spec.RunSpec.ContainerConfig.Labels[key] = value
	}

	for k, v := range makeContainerDockerLabels(cont) {
		spec.RunSpec.ContainerConfig.Labels[k] = v
	}

	spec.RunSpec.HostConfig.DeviceRequests = append(
		spec.RunSpec.HostConfig.DeviceRequests, gpuDeviceRequests(cont)...)

	if spec.RunSpec.UseFluentLogging {
		spec.RunSpec.HostConfig.LogConfig = dcontainer.LogConfig{
			Type: "fluentd",
			Config: map[string]string{
				"fluentd-address":              "localhost:" + strconv.Itoa(fluentPort),
				"fluentd-sub-second-precision": "true",
				"mode":                         "non-blocking",
				"max-buffer-size":              "10m",
				"env":                          strings.Join(fluentEnvVarNames, ","),
				"labels":                       dockerContainerParentLabel,
			},
		}
	}

	return spec
}

func gpuDeviceRequests(cont cproto.Container) []dcontainer.DeviceRequest {
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

func containerEnvVars(cont cproto.Container) []string {
	var slotIds []string
	for _, d := range cont.Devices {
		slotIds = append(slotIds, strconv.Itoa(d.ID))
	}
	return []string{
		fmt.Sprintf("DET_CONTAINER_ID=%s", cont.ID),
		fmt.Sprintf("DET_SLOT_IDS=[%s]", strings.Join(slotIds, ",")),
	}
}

func (c *containerManager) reattachContainers(
	ctx *actor.Context, expectedSurvivors []aproto.ContainerReattach) (
	[]aproto.ContainerReattachAck, error) {
	result := make([]aproto.ContainerReattachAck, 0, len(expectedSurvivors))

	runningContainers, err := c.listRunningContainers(ctx)
	if err != nil {
		return nil, err
	}

	for _, expectedSurvivor := range expectedSurvivors {
		var ack aproto.ContainerReattachAck

		containerInfo, ok := runningContainers[expectedSurvivor.Container.ID]
		if !ok {
			ack = aproto.ContainerReattachAck{
				Container: cproto.Container{ID: expectedSurvivor.Container.ID},
				Failure: &aproto.ContainerFailure{
					FailureType: aproto.AgentFailed,
					ErrMsg:      "container is gone on reattachment",
				},
			}
		} else {
			ctx.Log().Infof("will reattach container %s", expectedSurvivor.Container.ID)
			cpc, err := c.reattachContainer(ctx, expectedSurvivor.Container, containerInfo)
			if err != nil {
				ack = aproto.ContainerReattachAck{
					Container: *cpc,
					Failure: &aproto.ContainerFailure{
						FailureType: aproto.AgentFailed,
						ErrMsg:      "failed to restore info from container labels",
					},
				}
			} else {
				ack = aproto.ContainerReattachAck{
					Container: cproto.Container{ID: expectedSurvivor.Container.ID},
				}
			}
		}

		result = append(result, ack)
		delete(runningContainers, expectedSurvivor.Container.ID)
	}

	// SIGKILL the rest.
	for cid, containerInfo := range runningContainers {
		ctx.Log().Infof("will kill container %s", cid)
		err := c.docker.ContainerKill(
			context.Background(), containerInfo.ID, unix.SignalName(unix.SIGKILL))
		if err != nil {
			ctx.Log().WithError(err).Warnf("failed to kill container %s", cid)
		}
	}

	return result, nil
}

func (c *containerManager) reattachContainer(
	ctx *actor.Context, containerPrevState cproto.Container, containerInfo types.Container) (
	*cproto.Container, error) {
	containerCurrState, err := c.unmakeContainerDockerLabels(ctx, containerInfo)
	if err != nil {
		return nil, err
	}
	// TODO(ilia): Support reattaching containers that have changed state:
	// - starting -> running,
	// - running -> terminated.
	if containerCurrState.State != containerPrevState.State {
		return nil, errors.New("container has changed state while offline")
	}

	cid := containerPrevState.ID
	_, ok := ctx.ActorOf(cid, reattachContainerActor(*containerCurrState, c.docker))
	if !ok {
		errorMsg := fmt.Sprintf("failed to reattach container %s: actor already exists", cid)
		ctx.Log().Warnf(errorMsg)
		if killed := ctx.Kill(cid); killed {
			ctx.Log().Warnf("possible invalid state, killed container actor %s", cid)
		} else {
			ctx.Log().Warnf("possible invalid state, failed to kill container actor %s", cid)
		}
		return nil, errors.New(errorMsg)
	}

	return containerCurrState, nil
}

func (c *containerManager) listRunningContainers(ctx *actor.Context) (
	map[cproto.ID]types.Container, error) {
	// List "our" running containers, based on `dockerAgentLabel`.
	// This doesn't affect fluentbit, or containers spawned by other agents.
	containers, err := c.docker.ContainerList(context.Background(), types.ContainerListOptions{
		All: false,
		Filters: filters.NewArgs(
			filters.Arg("label", dockerAgentLabel+"="+c.Options.AgentID),
		),
	})
	if err != nil {
		return nil, err
	}

	result := make(map[cproto.ID]types.Container, len(containers))

	for _, cont := range containers {
		containerID, ok := cont.Labels[dockerContainerIDLabel]
		if ok {
			result[cproto.ID(containerID)] = cont
		} else {
			ctx.Log().Warnf("container %v has agent label but no container id", cont.ID)
		}
	}

	return result, nil
}

func makeContainerDockerLabels(cont cproto.Container) map[string]string {
	labels := map[string]string{}
	labels[dockerContainerVersionLabel] = dockerContainerVersionValue
	labels[dockerContainerIDLabel] = cont.ID.String()
	labels[dockerContainerParentLabel] = cont.Parent.String()
	var slotIds []string
	for _, d := range cont.Devices {
		slotIds = append(slotIds, strconv.Itoa(d.ID))
	}
	labels[dockerContainerDevicesLabel] = strings.Join(slotIds, ",")

	return labels
}

func (c *containerManager) unmakeContainerDockerLabels(ctx *actor.Context, cont types.Container) (
	*cproto.Container, error) {
	// TODO(ilia): Shim old versions whenever possible, when we'll have them.
	if cont.Labels[dockerContainerVersionLabel] != dockerContainerVersionValue {
		return nil, errors.New(fmt.Sprintf(
			"can't parse container labels version %s", cont.Labels[dockerContainerVersionLabel]))
	}

	devicesLabel := cont.Labels[dockerContainerDevicesLabel]
	devices := []device.Device{}

	// devicesLabel is empty for zero-slot tasks.
	if len(devicesLabel) > 0 {
		slotIDs := strings.Split(devicesLabel, ",")
		for _, slotID := range slotIDs {
			number, err := strconv.ParseInt(slotID, 10, 64)
			if err != nil {
				return nil, err
			}
			devices = append(devices, c.Devices[number])
		}
	}

	state, err := cproto.ParseStateFromDocker(cont)
	if err != nil {
		return nil, err
	}

	return &cproto.Container{
		ID:      cproto.ID(cont.Labels[dockerContainerIDLabel]),
		Parent:  actor.AddrFromString(cont.Labels[dockerContainerParentLabel]),
		Devices: devices,
		State:   state,
	}, nil
}
