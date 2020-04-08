package tasks

import (
	"archive/tar"
	"fmt"
	"path/filepath"
	"strings"

	docker "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/go-connections/nat"

	"github.com/determined-ai/determined/master/pkg/archive"
	"github.com/determined-ai/determined/master/pkg/container"
	"github.com/determined-ai/determined/master/pkg/device"
	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/model"
)

const (
	containerWorkDir  = "/run/determined/workdir"
	userPythonBaseDir = "/run/determined/pythonuserbase"
	runDir            = "/run/determined"
	rootDir           = "/"
)

func defaultEnvVars() []string {
	// PYTHONUSERBASE allows us to `pip install --user` into a location guaranteed to be owned by
	// the user inside the container.
	envVars := []string{"PYTHONUSERBASE=" + userPythonBaseDir}
	return envVars
}

// workDirArchive ensures that the workdir is created and owned by the user.
func workDirArchive(aug *model.AgentUserGroup) container.RunArchive {
	return wrapArchive(
		archive.Archive{
			aug.OwnedArchiveItem(runDir, nil, 0700, tar.TypeDir),
			aug.OwnedArchiveItem(containerWorkDir, nil, 0700, tar.TypeDir),
			aug.OwnedArchiveItem(userPythonBaseDir, nil, 0700, tar.TypeDir),
		},
		rootDir,
	)
}

// ToContainerSpec returns the container spec for associated task spec. This is a bridge method
// for the agent refactor project.
func ToContainerSpec(t TaskSpec) container.Spec {
	switch {
	case t.StartCommand != nil:
		return startCommand(t)
	case t.StartContainer != nil:
		return startContainer(t)
	case t.GCCheckpoints != nil:
		return gcCheckpoint(t)
	default:
		panic("unexpected task spec received")
	}
}

func startCommand(t TaskSpec) container.Spec {
	cmd := *t.StartCommand
	user := ""
	if cmd.AgentUserGroup != nil {
		user = fmt.Sprintf("%d:%d", cmd.AgentUserGroup.UID, cmd.AgentUserGroup.GID)
	}
	deviceType := device.CPU
	if len(t.Devices) > 0 {
		deviceType = t.Devices[0].Type
	}
	envVars := defaultEnvVars()
	envVars = append(envVars, cmd.Config.Environment.EnvironmentVariables.For(deviceType)...)
	envVars = append(envVars, fmt.Sprintf("DET_TASK_ID=%s", t.TaskID))
	return container.Spec{
		PullSpec: container.PullSpec{
			Registry:  cmd.Config.Environment.RegistryAuth,
			ForcePull: cmd.Config.Environment.ForcePullImage,
		},
		RunSpec: container.RunSpec{
			ContainerConfig: docker.Config{
				User:         user,
				ExposedPorts: toPortSet(cmd.Config.Environment.Ports),
				Env:          envVars,
				Cmd:          cmd.Config.Entrypoint,
				Image:        cmd.Config.Environment.Image.For(deviceType),
				WorkingDir:   containerWorkDir,
			},
			HostConfig: docker.HostConfig{
				NetworkMode:     t.ContainerDefaults.NetworkMode,
				Mounts:          toDockerMounts(cmd.Config.BindMounts),
				PublishAllPorts: true,
				ShmSize:         t.ContainerDefaults.ShmSizeBytes,
			},
			Archives: []container.RunArchive{
				workDirArchive(cmd.AgentUserGroup),
				wrapArchive(cmd.AgentUserGroup.OwnArchive(cmd.UserFiles), containerWorkDir),
				wrapArchive(cmd.AdditionalFiles, rootDir),
				harnessArchive(t.HarnessPath, cmd.AgentUserGroup),
			},
		},
	}
}

func startContainer(t TaskSpec) container.Spec {
	exp := *t.StartContainer
	user := ""
	if exp.AgentUserGroup != nil {
		user = fmt.Sprintf("%d:%d", exp.AgentUserGroup.UID, exp.AgentUserGroup.GID)
	}
	deviceType := device.CPU
	if len(t.Devices) > 0 {
		deviceType = t.Devices[0].Type
	}
	mounts := toDockerMounts(exp.ExperimentConfig.BindMounts)
	if exp.ExperimentConfig.CheckpointStorage.SharedFSConfig != nil {
		sharedFS := exp.ExperimentConfig.CheckpointStorage.SharedFSConfig
		mounts = append(mounts, mount.Mount{
			Type:   mount.TypeBind,
			Source: sharedFS.HostPath,
			Target: sharedFS.ContainerPath.String(),
			BindOptions: &mount.BindOptions{
				Propagation: mount.Propagation(sharedFS.Propagation.String()),
			},
		})
	}
	networkMode := t.ContainerDefaults.NetworkMode
	if exp.ExperimentConfig.Resources.SlotsPerTrial > 1 {
		networkMode = hostMode
	}
	rPorts := rendezvousPorts(t.Devices, networkMode)
	ports := make(nat.PortSet)
	var rPortsEnvVars []string
	for _, port := range rPorts {
		rPortsEnvVars = append(rPortsEnvVars, port.Port())
		ports[port] = struct{}{}
	}

	networkInterface := exp.TrialRunnerConfig.NetworkInterface
	if networkInterface == "" {
		networkInterface = "DET_AUTO_DETECT_NETWORK_INTERFACE"
	}

	envVars := defaultEnvVars()
	envVars = append(envVars, exp.ExperimentConfig.Environment.EnvironmentVariables.For(deviceType)...)
	envVars = append(envVars,
		fmt.Sprintf("DET_EXPERIMENT_ID=%d", exp.InitialWorkload.ExperimentID),
		fmt.Sprintf("DET_TRIAL_ID=%d", exp.InitialWorkload.TrialID),
		fmt.Sprintf("DET_TRIAL_SEED=%d", exp.TrialSeed),
		fmt.Sprintf("DET_EXPERIMENT_CONFIG=%s", jsonify(exp.ExperimentConfig)),
		fmt.Sprintf("DET_HPARAMS=%s", jsonify(exp.HParams)),
		fmt.Sprintf("DET_INITIAL_WORKLOAD=%s", jsonify(exp.InitialWorkload)),
		fmt.Sprintf("DET_LATEST_CHECKPOINT=%s", jsonify(exp.LatestCheckpoint)),
		fmt.Sprintf("DET_WORKLOAD_MANAGER_TYPE=%s", exp.WorkloadManagerType),
		fmt.Sprintf("DET_RENDEZVOUS_PORTS=%s", strings.Join(rPortsEnvVars, ",")),
		fmt.Sprintf("DET_TRIAL_RUNNER_NETWORK_INTERFACE=%s", networkInterface),
	)

	if exp.TrialRunnerConfig.NCCLPortRange != "" {
		envVars = append(envVars, fmt.Sprintf("NCCL_PORT_RANGE=%s", exp.TrialRunnerConfig.NCCLPortRange))
	}
	if exp.TrialRunnerConfig.GLOOPortRange != "" {
		envVars = append(envVars, fmt.Sprintf("GLOO_PORT_RANGE=%s", exp.TrialRunnerConfig.GLOOPortRange))
	}

	spec := container.Spec{
		PullSpec: container.PullSpec{
			ForcePull: exp.ExperimentConfig.Environment.ForcePullImage,
			Registry:  exp.ExperimentConfig.Environment.RegistryAuth,
		},
		RunSpec: container.RunSpec{
			ContainerConfig: docker.Config{
				Cmd:          []string{"/run/determined/workdir/entrypoint.sh"},
				User:         user,
				Image:        exp.ExperimentConfig.Environment.Image.For(deviceType),
				ExposedPorts: ports,
				Env:          envVars,
				WorkingDir:   containerWorkDir,
			},
			HostConfig: docker.HostConfig{
				NetworkMode:     networkMode,
				Mounts:          mounts,
				PublishAllPorts: true,
			},
			Archives: []container.RunArchive{
				workDirArchive(exp.AgentUserGroup),
				wrapArchive(exp.AdditionalFiles, rootDir),
				wrapArchive(exp.AgentUserGroup.OwnArchive(exp.ModelDefinition), containerWorkDir),
				harnessArchive(t.HarnessPath, exp.AgentUserGroup),
			},
		},
	}
	spec.RunSpec.HostConfig.ShmSize = t.ContainerDefaults.ShmSizeBytes
	if exp.ExperimentConfig.Resources.ShmSize != nil {
		spec.RunSpec.HostConfig.ShmSize = int64(*exp.ExperimentConfig.Resources.ShmSize)
	}
	return spec
}

func gcCheckpoint(t TaskSpec) container.Spec {
	gcc := *t.GCCheckpoints
	user := ""
	if gcc.AgentUserGroup != nil {
		user = fmt.Sprintf("%d:%d", gcc.AgentUserGroup.UID, gcc.AgentUserGroup.GID)
	}
	deviceType := device.CPU
	if len(t.Devices) > 0 {
		deviceType = t.Devices[0].Type
	}
	envVars := defaultEnvVars()
	envVars = append(envVars, gcc.ExperimentConfig.Environment.EnvironmentVariables.For(deviceType)...)
	envVars = append(envVars,
		fmt.Sprintf("DET_EXPERIMENT_ID=%d", gcc.ExperimentID),
		fmt.Sprintf("DET_EXPERIMENT_CONFIG=%s", jsonify(gcc.ExperimentConfig)),
		fmt.Sprintf("DET_DELETE=%s", jsonify(gcc.ToDelete)),
	)
	mounts := toDockerMounts(gcc.ExperimentConfig.BindMounts)
	if gcc.ExperimentConfig.CheckpointStorage.SharedFSConfig != nil {
		sharedFS := gcc.ExperimentConfig.CheckpointStorage.SharedFSConfig
		mounts = append(mounts, mount.Mount{
			Type:   mount.TypeBind,
			Source: sharedFS.HostPath,
			Target: sharedFS.ContainerPath.String(),
			BindOptions: &mount.BindOptions{
				Propagation: mount.Propagation(sharedFS.Propagation.String()),
			},
		})
	}
	return container.Spec{
		PullSpec: container.PullSpec{
			ForcePull: gcc.ExperimentConfig.Environment.ForcePullImage,
			Registry:  gcc.ExperimentConfig.Environment.RegistryAuth,
		},
		RunSpec: container.RunSpec{
			ContainerConfig: docker.Config{
				Cmd:        []string{filepath.Join(containerWorkDir, etc.GCCheckpointsEntrypointResource)},
				User:       user,
				Image:      gcc.ExperimentConfig.Environment.Image.For(deviceType),
				Env:        envVars,
				WorkingDir: containerWorkDir,
			},
			HostConfig: docker.HostConfig{
				NetworkMode:     t.ContainerDefaults.NetworkMode,
				Mounts:          mounts,
				PublishAllPorts: true,
			},
			Archives: []container.RunArchive{
				workDirArchive(gcc.AgentUserGroup),
				wrapArchive(
					archive.Archive{
						gcc.AgentUserGroup.OwnedArchiveItem(
							etc.GCCheckpointsEntrypointResource,
							etc.MustStaticFile(etc.GCCheckpointsEntrypointResource),
							0700,
							tar.TypeReg,
						),
					},
					containerWorkDir,
				),
				harnessArchive(t.HarnessPath, gcc.AgentUserGroup),
			},
		},
	}
}
