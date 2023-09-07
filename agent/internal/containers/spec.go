package containers

import (
	"fmt"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	dcontainer "github.com/docker/docker/api/types/container"

	"github.com/docker/docker/api/types"
	"github.com/pkg/errors"

	"github.com/determined-ai/determined/agent/internal/container"
	"github.com/determined-ai/determined/agent/internal/detect"
	"github.com/determined-ai/determined/agent/internal/fluent"
	"github.com/determined-ai/determined/agent/internal/options"
	"github.com/determined-ai/determined/agent/pkg/docker"
	"github.com/determined-ai/determined/master/pkg/aproto"
	"github.com/determined-ai/determined/master/pkg/cproto"
	"github.com/determined-ai/determined/master/pkg/device"
)

func overwriteSpec(
	spec cproto.Spec,
	cont cproto.Container,
	opts options.Options,
	mopts aproto.MasterSetAgentOptions,
) (cproto.Spec, error) {
	spec.RunSpec.ContainerConfig.Env = addProxyInfo(spec.RunSpec.ContainerConfig.Env, opts)
	spec.RunSpec.ContainerConfig.Env = append(
		spec.RunSpec.ContainerConfig.Env, makeGlobalEnvVars(opts, mopts)...)
	spec.RunSpec.ContainerConfig.Env = append(spec.RunSpec.ContainerConfig.Env, containerEnv(cont)...)

	spec.RunSpec.HostConfig.AutoRemove = !opts.ContainerAutoRemoveDisabled

	if spec.RunSpec.ContainerConfig.Labels == nil {
		spec.RunSpec.ContainerConfig.Labels = make(map[string]string)
	}
	for k, v := range makeLabels(opts, mopts) {
		spec.RunSpec.ContainerConfig.Labels[k] = v
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

	spec.RunSpec.HostConfig.LogConfig = generateLoggingConfig(opts.Fluent.Port)

	return spec, nil
}

func addProxyInfo(env []string, opts options.Options) []string {
	addVars := map[string]string{
		"HTTP_PROXY":  opts.HTTPProxy,
		"HTTPS_PROXY": opts.HTTPSProxy,
		"FTP_PROXY":   opts.FTPProxy,
		"NO_PROXY":    opts.NoProxy,
	}
	for _, v := range env {
		key := strings.SplitN(v, "=", 2)[0]
		key = strings.ToUpper(key)
		_, ok := addVars[key]
		if ok {
			delete(addVars, key)
		}
	}
	for k, v := range addVars {
		if v != "" {
			env = append(env, k+"="+v)
		}
	}
	return env
}

func generateLoggingConfig(port int) dcontainer.LogConfig {
	return dcontainer.LogConfig{
		Type: "fluentd",
		Config: map[string]string{
			"fluentd-address":              "localhost:" + strconv.Itoa(port),
			"fluentd-sub-second-precision": "true",
			"mode":                         "non-blocking",
			"max-buffer-size":              "10m",
			"env":                          strings.Join(fluent.EnvVarNames, ","),
		},
	}
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
		rocmDevice := detect.GetRocmDeviceByUUID(uuid)
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

func containerEnv(cont cproto.Container) []string {
	var slotIds []string
	for _, d := range cont.Devices {
		slotIds = append(slotIds, strconv.Itoa(int(d.ID)))
	}
	return []string{
		fmt.Sprintf("DET_CONTAINER_ID=%s", cont.ID),
		fmt.Sprintf("DET_SLOT_IDS=[%s]", strings.Join(slotIds, ",")),
	}
}

func makeContainerDockerLabels(cont cproto.Container) map[string]string {
	labels := map[string]string{}
	labels[docker.ContainerVersionLabel] = docker.ContainerVersionValue
	labels[docker.ContainerIDLabel] = cont.ID.String()
	labels[docker.ContainerDescriptionLabel] = cont.Description
	var slotIds []string
	for _, d := range cont.Devices {
		slotIds = append(slotIds, strconv.Itoa(int(d.ID)))
	}
	labels[docker.ContainerDevicesLabel] = strings.Join(slotIds, ",")
	return labels
}

func (m *Manager) unmakeContainerDockerLabels(cont types.Container) (
	*cproto.Container, error,
) {
	// TODO(ilia): Shim old versions whenever possible, when we'll have them.
	if cont.Labels[docker.ContainerVersionLabel] != docker.ContainerVersionValue {
		return nil, fmt.Errorf(
			"can't parse container labels version %s", cont.Labels[docker.ContainerVersionLabel])
	}

	devicesLabel := cont.Labels[docker.ContainerDevicesLabel]
	devices := []device.Device{}

	// devicesLabel is empty for zero-slot tasks.
	if len(devicesLabel) > 0 {
		slotIDs := strings.Split(devicesLabel, ",")
		for _, slotID := range slotIDs {
			number, err := strconv.ParseInt(slotID, 10, 64)
			if err != nil {
				return nil, err
			}
			devices = append(devices, m.devices[number])
		}
	}

	state, err := cproto.ParseStateFromDocker(cont)
	if err != nil {
		return nil, err
	}

	return &cproto.Container{
		ID:          cproto.ID(cont.Labels[docker.ContainerIDLabel]),
		Devices:     devices,
		State:       state,
		Description: cont.Labels[docker.ContainerDescriptionLabel],
	}, nil
}

func makeGlobalEnvVars(opts options.Options, mopts aproto.MasterSetAgentOptions) []string {
	masterScheme := httpInsecureScheme
	if opts.Security.TLS.Enabled {
		masterScheme = httpSecureScheme
	}
	masterHost := opts.ContainerMasterHost
	if masterHost == "" {
		masterHost = opts.MasterHost
	}
	masterPort := opts.ContainerMasterPort
	if masterPort == 0 {
		masterPort = opts.MasterPort
	}
	globalEnvVars := []string{
		fmt.Sprintf("DET_CLUSTER_ID=%s", mopts.MasterInfo.ClusterID),
		fmt.Sprintf("DET_MASTER_ID=%s", mopts.MasterInfo.MasterID),
		fmt.Sprintf("DET_MASTER=%s://%s:%d", masterScheme, masterHost, masterPort),
		fmt.Sprintf("DET_MASTER_HOST=%s", masterHost),
		fmt.Sprintf("DET_MASTER_ADDR=%s", masterHost),
		fmt.Sprintf("DET_MASTER_PORT=%d", masterPort),
		fmt.Sprintf("%s=%s", container.AgentIDEnvVar, opts.AgentID),
	}
	if a := opts.Security.TLS.MasterCertName; a != "" {
		globalEnvVars = append(globalEnvVars, fmt.Sprintf("DET_MASTER_CERT_NAME=%s", a))
	}

	return globalEnvVars
}

func makeLabels(opts options.Options, mopts aproto.MasterSetAgentOptions) map[string]string {
	return map[string]string{
		docker.ContainerTypeLabel: docker.ContainerTypeValue,
		docker.AgentLabel:         opts.AgentID,
		docker.ClusterLabel:       mopts.MasterInfo.ClusterID,
		docker.MasterLabel:        mopts.MasterInfo.MasterID,
	}
}

// validateDevices checks the devices requested in container.Spec are a subset of agent devices.
func validateDevices(agentDevices, containerDevices []device.Device) bool {
	for _, d := range containerDevices {
		if !slices.Contains(agentDevices, d) {
			return false
		}
	}
	return true
}
