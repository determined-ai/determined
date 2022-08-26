package internal

import (
	"container/ring"
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/docker/docker/api/types"
	dcontainer "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"golang.org/x/exp/maps"
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

const recentExitsKept = 32

type containerManager struct {
	Options       Options           `json:"-"`
	MasterInfo    aproto.MasterInfo `json:"-"`
	GlobalEnvVars []string          `json:"global_env_vars"`
	Labels        map[string]string `json:"labels"`
	Devices       []device.Device   `json:"devices"`

	fluentPort int

	docker *client.Client

	recentExits *ring.Ring
}

type (
	requestReattachContainers struct {
		ContainersToReattach []aproto.ContainerReattach
	}
	requestRevalidateContainers struct {
		ContainersToReattach []aproto.ContainerReattach
	}
	responseReattachContainers struct {
		ContainersReattached []aproto.ContainerReattachAck
	}
)

func newContainerManager(a *agent, fluentPort int) (*containerManager, error) {
	return &containerManager{
		MasterInfo:  a.MasterSetAgentOptions.MasterInfo,
		Options:     a.Options,
		Devices:     a.Devices,
		fluentPort:  fluentPort,
		recentExits: ring.New(recentExitsKept),
	}, nil
}

func (c *containerManager) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart:
		d, err := client.NewClientWithOpts(client.WithAPIVersionNegotiation(), client.FromEnv)
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
	case requestRevalidateContainers:
		containers, err := c.revalidateContainers(ctx, msg.ContainersToReattach)
		if err != nil {
			ctx.Log().WithError(err).Warn("failed to revalidate containers")
			ctx.Respond(responseReattachContainers{})
		} else {
			ctx.Respond(responseReattachContainers{ContainersReattached: containers})
		}

	case aproto.ContainerStateChanged:
		if msg.ContainerStopped != nil {
			c.recentExits = c.recentExits.Prev()
			c.recentExits.Value = msg
		}

		ctx.Tell(ctx.Self().Parent(), msg)

	case aproto.ContainerLog, model.TaskLog, aproto.ContainerStatsRecord:
		ctx.Tell(ctx.Self().Parent(), msg)

	case aproto.StartContainer:
		enrichedSpec, err := c.overwriteSpec(msg.Container, msg.Spec)
		if err != nil {
			ctx.Log().WithError(err).Errorf("failed to overwrite spec")
			if ctx.ExpectingResponse() {
				ctx.Respond(errors.Wrap(err, "failed to overwrite spec"))
			}
			return nil
		}
		// actually overwrite the spec.
		msg.Spec = enrichedSpec
		if ref, ok := ctx.ActorOf(
			msg.Container.ID, newContainerActor(msg, c.docker)); !ok {
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
			// Fallback state change if the original is already gone from recent exits.
			csc := aproto.ContainerStateChanged{
				Container: cproto.Container{
					Parent:  ctx.Self().Address(),
					ID:      msg.ContainerID,
					State:   cproto.Terminated,
					Devices: nil,
				},
				ContainerStopped: &aproto.ContainerStopped{
					Failure: &aproto.ContainerFailure{
						FailureType: aproto.ContainerMissing,
						ErrMsg: fmt.Sprintf(
							"cannot signal container with %s, container actor not found: %s",
							msg.Signal, msg.ContainerID,
						),
					},
				},
			}

			// Try to pull the termination message from recent exits.
			c.recentExits.Do(func(v any) {
				if v == nil {
					return
				}

				savedStop := v.(aproto.ContainerStateChanged)
				if msg.ContainerID != savedStop.Container.ID {
					return
				}
				csc = savedStop
			})

			// If the master is still sending us signals for this container, it likely thinks
			// it still exists, so we should clarify.
			ctx.Log().Warnf("resending stop due to %s", unix.SignalName(msg.Signal))
			ctx.Tell(ctx.Self().Parent(), csc)
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

func (c *containerManager) overwriteSpec(cont cproto.Container, spec cproto.Spec) (
	cproto.Spec, error,
) {
	autoRemove := !c.Options.ContainerAutoRemoveDisabled
	return overwriteSpec(cont, spec, c.GlobalEnvVars, c.Labels, c.fluentPort, autoRemove)
}

func overwriteSpec(
	cont cproto.Container,
	spec cproto.Spec,
	globalEnvVars []string,
	labels map[string]string,
	fluentPort int,
	autoRemove bool,
) (cproto.Spec, error) {
	spec.RunSpec.HostConfig.AutoRemove = autoRemove
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

	if len(cont.DeviceUUIDsByType(device.CUDA)) > 0 {
		spec.RunSpec.HostConfig.DeviceRequests = append(
			spec.RunSpec.HostConfig.DeviceRequests, cudaDeviceRequests(cont)...)
	}

	if len(cont.DeviceUUIDsByType(device.ROCM)) > 0 {
		if err := injectRocmDeviceRequests(cont, &spec.RunSpec.HostConfig); err != nil {
			return cproto.Spec{}, err
		}
	}

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

	return spec, nil
}

func cudaDeviceRequests(cont cproto.Container) []dcontainer.DeviceRequest {
	cudaUUIDs := cont.DeviceUUIDsByType(device.CUDA)
	if len(cudaUUIDs) == 0 {
		return nil
	}
	return []dcontainer.DeviceRequest{
		{
			Driver:       "nvidia",
			Capabilities: [][]string{{"gpu", "compute", "utility"}},
			DeviceIDs:    cudaUUIDs,
		},
	}
}

func injectRocmDeviceRequests(cont cproto.Container, hostConfig *dcontainer.HostConfig) error {
	// Docker args for "all rocm gpus":
	//   --device=/dev/kfd --device=/dev/dri --security-opt seccomp=unconfined --group-add video
	// To confine it to individual cards, we've got to pass only `/dev/dri/card<ID1>` and
	// `/dev/dri/renderD<ID2>`, e.g. `/dev/dri/{card0,renderD128}`.
	// rocm-smi gives us UUIDs and PCIBus locations for the cards; we resolve symlinks
	// in `/dev/dri/by-path/` to get the real device paths for docker.
	uuids := cont.DeviceUUIDsByType(device.ROCM)

	if len(uuids) == 0 {
		return errors.New("no rocm device uuids")
	}

	hostConfig.SecurityOpt = append(
		hostConfig.SecurityOpt, "seccomp=unconfined")
	hostConfig.GroupAdd = append(hostConfig.GroupAdd, "video")
	mappedDevices := []string{"/dev/kfd"}

	for _, uuid := range uuids {
		rocmDevice := getRocmDeviceByUUID(uuid)
		devPaths := []string{
			fmt.Sprintf("/dev/dri/by-path/pci-%s-card", strings.ToLower(rocmDevice.PCIBus)),
			fmt.Sprintf("/dev/dri/by-path/pci-%s-render", strings.ToLower(rocmDevice.PCIBus)),
		}
		for _, symlink := range devPaths {
			resolved, err := filepath.EvalSymlinks(symlink)
			if err != nil {
				return err
			}
			mappedDevices = append(mappedDevices, resolved)
		}
	}

	for _, mappedDevice := range mappedDevices {
		hostConfig.Devices = append(
			hostConfig.Devices, dcontainer.DeviceMapping{
				PathOnHost:        mappedDevice,
				PathInContainer:   mappedDevice,
				CgroupPermissions: "rwm",
			})
	}

	return nil
}

func containerEnvVars(cont cproto.Container) []string {
	var slotIds []string
	for _, d := range cont.Devices {
		slotIds = append(slotIds, strconv.Itoa(int(d.ID)))
	}
	return []string{
		fmt.Sprintf("DET_CONTAINER_ID=%s", cont.ID),
		fmt.Sprintf("DET_SLOT_IDS=[%s]", strings.Join(slotIds, ",")),
	}
}

func (c *containerManager) reattachContainers(
	ctx *actor.Context, expectedSurvivors []aproto.ContainerReattach) (
	[]aproto.ContainerReattachAck, error,
) {
	result := make([]aproto.ContainerReattachAck, 0, len(expectedSurvivors))

	runningContainers, err := c.listRunningContainers(ctx)
	if err != nil {
		return nil, err
	}
	ctx.Log().Debug("reattachContainers: running containers: ", maps.Keys(runningContainers))

	for _, expectedSurvivor := range expectedSurvivors {
		var ack aproto.ContainerReattachAck

		containerInfo, ok := runningContainers[expectedSurvivor.Container.ID]
		if !ok {
			ack = aproto.ContainerReattachAck{
				Container: cproto.Container{ID: expectedSurvivor.Container.ID},
				Failure: &aproto.ContainerFailure{
					FailureType: aproto.RestoreError,
					ErrMsg:      "container is gone on reattachment",
				},
			}
		} else {
			ctx.Log().Infof("will reattach container %s", expectedSurvivor.Container.ID)
			cpc, err := c.reattachContainer(ctx, expectedSurvivor.Container, containerInfo)
			if err != nil {
				err = fmt.Errorf("failed to restore info from container labels: %w", err)
				ack = aproto.ContainerReattachAck{
					Container: cproto.Container{ID: expectedSurvivor.Container.ID},
					Failure: &aproto.ContainerFailure{
						FailureType: aproto.RestoreError,
						ErrMsg:      err.Error(),
					},
				}
			} else {
				ack = aproto.ContainerReattachAck{
					Container: *cpc,
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
	*cproto.Container, error,
) {
	containerCurrState, err := c.unmakeContainerDockerLabels(ctx, containerInfo)
	if err != nil {
		return nil, err
	}
	// TODO(ilia): Support reattaching containers that have changed state:
	// - starting -> running,
	// - running -> terminated.
	if containerPrevState.State != "" && containerCurrState.State != containerPrevState.State {
		return nil, fmt.Errorf(
			"container has changed state while offline. now: %s, was: %s",
			containerCurrState.State, containerPrevState.State)
	}

	cid := containerPrevState.ID
	containerRef, ok := ctx.ActorOf(cid, reattachContainerActor(*containerCurrState, c.docker))
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
	ctx.Ask(containerRef, actor.Ping{}).Get()
	ctx.Log().Debugf("reattached container actor %s", cid)

	return containerCurrState, nil
}

func (c *containerManager) listRunningContainers(ctx *actor.Context) (
	map[cproto.ID]types.Container, error,
) {
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
		slotIds = append(slotIds, strconv.Itoa(int(d.ID)))
	}
	labels[dockerContainerDevicesLabel] = strings.Join(slotIds, ",")

	return labels
}

func (c *containerManager) unmakeContainerDockerLabels(ctx *actor.Context, cont types.Container) (
	*cproto.Container, error,
) {
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

func (c *containerManager) revalidateContainers(
	ctx *actor.Context, expectedSurvivors []aproto.ContainerReattach) (
	[]aproto.ContainerReattachAck, error,
) {
	result := make([]aproto.ContainerReattachAck, 0, len(expectedSurvivors))

	for _, expectedSurvivor := range expectedSurvivors {
		cid := expectedSurvivor.Container.ID

		// Fallback container reattach ack.
		ack := aproto.ContainerReattachAck{
			Container: cproto.Container{ID: cid},
			Failure: &aproto.ContainerFailure{
				FailureType: aproto.RestoreError,
				ErrMsg:      "failed to restore container on master blip",
			},
		}

		// If the child is still there, assuming nothing has changed.
		if ref := ctx.Child(cid.String()); ref != nil {
			container := ctx.Ask(ref, getContainerSummary{}).Get().(cproto.Container)
			ack = aproto.ContainerReattachAck{
				Container: container,
			}
		}

		// But if there is a termination message for it, for any reason, go ahead and ack that.
		c.recentExits.Do(func(v any) {
			if v == nil {
				return
			}

			savedStop := v.(aproto.ContainerStateChanged)
			if cid != savedStop.Container.ID {
				return
			}
			ack = aproto.ContainerReattachAck{
				Container: savedStop.Container,
				Failure:   savedStop.ContainerStopped.Failure,
			}
		})

		result = append(result, ack)
	}

	return result, nil
}
