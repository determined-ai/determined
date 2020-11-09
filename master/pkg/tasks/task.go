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
	// ContainerWorkDir is the working directory for tasks.
	ContainerWorkDir  = "/run/determined/workdir"
	userPythonBaseDir = "/run/determined/pythonuserbase"
	runDir            = "/run/determined"
	trainDir          = "/run/determined/train"
	rootDir           = "/"
	passwdPath        = "/run/determined/etc/passwd"
	shadowPath        = "/run/determined/etc/shadow"
	groupPath         = "/run/determined/etc/group"
	certPath          = "/run/determined/etc/ssl/master.crt"
)

func defaultEnvVars() map[string]string {
	// PYTHONUSERBASE allows us to `pip install --user` into a location guaranteed to be owned by
	// the user inside the container.
	envVars := map[string]string{"PYTHONUSERBASE": userPythonBaseDir}
	return envVars
}

func addTLSVars(t TaskSpec, env map[string]string) {
	if t.MasterCert != nil {
		env["DET_USE_TLS"] = "true"
		env["DET_MASTER_CERT_FILE"] = certPath
	}
}

// workDirArchive ensures that the workdir is created and owned by the user.
func workDirArchive(aug *model.AgentUserGroup) container.RunArchive {
	return wrapArchive(
		archive.Archive{
			aug.OwnedArchiveItem(runDir, nil, 0700, tar.TypeDir),
			aug.OwnedArchiveItem(ContainerWorkDir, nil, 0700, tar.TypeDir),
			aug.OwnedArchiveItem(trainDir, nil, 0700, tar.TypeDir),
			aug.OwnedArchiveItem(userPythonBaseDir, nil, 0700, tar.TypeDir),
		},
		rootDir,
	)
}

// injectUserArchive creates the user/UID/group/GID for a user by adding passwd/shadow/group files
// to /run/determined/etc, which will be read by libnss_determined inside the container. If
// libnss_determined is not present in the container, these files will be simply ignored and some
// non-root container features will not work properly.
func injectUserArchive(aug *model.AgentUserGroup) container.RunArchive {
	passwdBytes := []byte(
		fmt.Sprintf("%v:x:%v:%v::%v:/bin/sh\n", aug.User, aug.UID, aug.GID, ContainerWorkDir),
	)
	shadowBytes := []byte(fmt.Sprintf("%v:!!:::::::\n", aug.User))
	groupBytes := []byte(fmt.Sprintf("%v:x:%v:\n", aug.Group, aug.GID))

	return wrapArchive(
		archive.Archive{
			archive.RootItem(passwdPath, passwdBytes, 0644, tar.TypeReg),
			archive.RootItem(shadowPath, shadowBytes, 0600, tar.TypeReg),
			archive.RootItem(groupPath, groupBytes, 0644, tar.TypeReg),
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

func getUser(agentUserGroup *model.AgentUserGroup) string {
	user := ""
	if agentUserGroup != nil {
		user = fmt.Sprintf("%d:%d", agentUserGroup.UID, agentUserGroup.GID)
	}
	return user
}

// CommandEnvVars configures environment variables for cmd tasks.
func CommandEnvVars(t TaskSpec) map[string]string {
	envVarsMap := defaultEnvVars()
	envVarsMap["DET_TASK_ID"] = t.TaskID
	addTLSVars(t, envVarsMap)

	return envVarsMap
}

// CommandArchives returns the additional files for a command as an archive.
func CommandArchives(t TaskSpec) []container.RunArchive {
	cmd := *t.StartCommand

	return []container.RunArchive{
		workDirArchive(cmd.AgentUserGroup),
		injectUserArchive(cmd.AgentUserGroup),
		wrapArchive(cmd.AgentUserGroup.OwnArchive(cmd.UserFiles), ContainerWorkDir),
		wrapArchive(cmd.AdditionalFiles, rootDir),
		harnessArchive(t.HarnessPath, cmd.AgentUserGroup),
		masterCertArchive(t.MasterCert),
	}
}

func startCommand(t TaskSpec) container.Spec {
	cmd := *t.StartCommand
	user := getUser(cmd.AgentUserGroup)
	deviceType := device.CPU
	if len(t.Devices) > 0 {
		deviceType = t.Devices[0].Type
	}
	envVarsMap := CommandEnvVars(t)
	envVars := make([]string, 0, len(envVarsMap))
	for envVarKey, envVarValue := range envVarsMap {
		envVars = append(envVars, fmt.Sprintf("%s=%s", envVarKey, envVarValue))
	}
	envVars = append(envVars, cmd.Config.Environment.EnvironmentVariables.For(deviceType)...)
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
				WorkingDir:   ContainerWorkDir,
			},
			HostConfig: docker.HostConfig{
				NetworkMode:     t.TaskContainerDefaults.NetworkMode,
				Mounts:          ToDockerMounts(cmd.Config.BindMounts),
				PublishAllPorts: true,
				ShmSize:         t.TaskContainerDefaults.ShmSizeBytes,
			},
			Archives: CommandArchives(t),
		},
	}
}

// TrialDockerMounts returns the host mounts for a trial container.
func TrialDockerMounts(exp StartContainer) []mount.Mount {
	mounts := ToDockerMounts(exp.ExperimentConfig.BindMounts)
	if exp.ExperimentConfig.CheckpointStorage.SharedFSConfig != nil {
		sharedFS := exp.ExperimentConfig.CheckpointStorage.SharedFSConfig
		mounts = append(mounts, mount.Mount{
			Type:   mount.TypeBind,
			Source: sharedFS.HostPath,
			Target: model.DefaultSharedFSContainerPath,
			BindOptions: &mount.BindOptions{
				Propagation: model.DefaultSharedFSPropagation,
			},
		})
	}

	if exp.ExperimentConfig.DataLayer.SharedFSConfig != nil {
		SharedFSConfig := exp.ExperimentConfig.DataLayer.SharedFSConfig
		if SharedFSConfig.HostStoragePath != nil {
			mounts = append(mounts, mount.Mount{
				Type:   mount.TypeBind,
				Source: *SharedFSConfig.HostStoragePath,
				Target: *SharedFSConfig.ContainerStoragePath,
			})
		}
	}
	if exp.ExperimentConfig.DataLayer.S3Config != nil {
		S3Config := exp.ExperimentConfig.DataLayer.S3Config
		if S3Config.LocalCacheHostPath != nil {
			mounts = append(mounts, mount.Mount{
				Type:   mount.TypeBind,
				Source: *S3Config.LocalCacheHostPath,
				Target: *S3Config.LocalCacheContainerPath,
			})
		}
	}
	if exp.ExperimentConfig.DataLayer.GCSConfig != nil {
		GCSConfig := exp.ExperimentConfig.DataLayer.GCSConfig
		if GCSConfig.LocalCacheHostPath != nil {
			mounts = append(mounts, mount.Mount{
				Type:   mount.TypeBind,
				Source: *GCSConfig.LocalCacheHostPath,
				Target: *GCSConfig.LocalCacheContainerPath,
			})
		}
	}

	return mounts
}

// TrialEnvVars returns environment variables for a trial.
func TrialEnvVars(t TaskSpec, rendezvousPorts []string, tPortOffset int) map[string]string {
	exp := *t.StartContainer

	networkInterface := t.TaskContainerDefaults.DtrainNetworkInterface
	if networkInterface == "" {
		networkInterface = "DET_AUTO_DETECT_NETWORK_INTERFACE"
	}

	envVars := defaultEnvVars()
	envVars["DET_EXPERIMENT_ID"] = fmt.Sprintf("%d", exp.InitialWorkload.ExperimentID)
	envVars["DET_TRIAL_ID"] = fmt.Sprintf("%d", exp.InitialWorkload.TrialID)
	envVars["DET_TRIAL_SEED"] = fmt.Sprintf("%d", exp.TrialSeed)
	envVars["DET_EXPERIMENT_CONFIG"] = jsonify(exp.ExperimentConfig)
	envVars["DET_HPARAMS"] = jsonify(exp.HParams)
	envVars["DET_INITIAL_WORKLOAD"] = jsonify(exp.InitialWorkload)
	envVars["DET_LATEST_CHECKPOINT"] = "/run/determined/train/checkpoint.json"
	envVars["DET_WORKLOAD_MANAGER_TYPE"] = string(exp.WorkloadManagerType)
	envVars["DET_RENDEZVOUS_PORTS"] = strings.Join(rendezvousPorts, ",")
	envVars["DET_TRIAL_UNIQUE_PORT_OFFSET"] = fmt.Sprintf("%d", tPortOffset)
	envVars["DET_TRIAL_RUNNER_NETWORK_INTERFACE"] = networkInterface
	addTLSVars(t, envVars)

	if t.TaskContainerDefaults.NCCLPortRange != "" {
		envVars["NCCL_PORT_RANGE"] = t.TaskContainerDefaults.NCCLPortRange
	}
	if t.TaskContainerDefaults.NCCLPortRange != "" {
		envVars["GLOO_PORT_RANGE"] = t.TaskContainerDefaults.NCCLPortRange
	}

	return envVars
}

// TrialArchives returns the additional files for a trial as an archive.
func TrialArchives(t TaskSpec) []container.RunArchive {
	exp := *t.StartContainer

	return []container.RunArchive{
		workDirArchive(exp.AgentUserGroup),
		injectUserArchive(exp.AgentUserGroup),
		wrapArchive(exp.AdditionalFiles, rootDir),
		wrapArchive(
			archive.Archive{
				exp.AgentUserGroup.OwnedArchiveItem(
					"checkpoint.json",
					[]byte(jsonify(exp.LatestCheckpoint)),
					0600,
					tar.TypeReg,
				),
			},
			trainDir,
		),
		wrapArchive(exp.AgentUserGroup.OwnArchive(exp.ModelDefinition), ContainerWorkDir),
		harnessArchive(t.HarnessPath, exp.AgentUserGroup),
		masterCertArchive(t.MasterCert),
	}
}

func startContainer(t TaskSpec) container.Spec {
	exp := *t.StartContainer
	user := getUser(exp.AgentUserGroup)
	deviceType := device.CPU
	if len(t.Devices) > 0 {
		deviceType = t.Devices[0].Type
	}
	mounts := TrialDockerMounts(exp)
	networkMode := t.TaskContainerDefaults.NetworkMode
	if exp.IsMultiAgent {
		networkMode = hostMode
	}
	tPortOffset := trialUniquePortOffset(t.Devices)
	rPorts := rendezvousPorts(tPortOffset)
	ports := make(nat.PortSet)
	var rPortsEnvVars []string
	for _, port := range rPorts {
		rPortsEnvVars = append(rPortsEnvVars, port.Port())
		ports[port] = struct{}{}
	}

	envVarsMap := TrialEnvVars(t, rPortsEnvVars, tPortOffset)
	envVars := make([]string, 0, len(envVarsMap))
	for envVarKey, envVarValue := range envVarsMap {
		envVars = append(envVars, fmt.Sprintf("%s=%s", envVarKey, envVarValue))
	}
	envVars = append(envVars, exp.ExperimentConfig.Environment.EnvironmentVariables.For(deviceType)...)

	spec := container.Spec{
		PullSpec: container.PullSpec{
			ForcePull: exp.ExperimentConfig.Environment.ForcePullImage,
			Registry:  exp.ExperimentConfig.Environment.RegistryAuth,
		},
		RunSpec: container.RunSpec{
			ContainerConfig: docker.Config{
				Cmd:          []string{"/run/determined/train/entrypoint.sh"},
				User:         user,
				Image:        exp.ExperimentConfig.Environment.Image.For(deviceType),
				ExposedPorts: ports,
				Env:          envVars,
				WorkingDir:   ContainerWorkDir,
			},
			HostConfig: docker.HostConfig{
				NetworkMode:     networkMode,
				Mounts:          mounts,
				PublishAllPorts: true,
			},
			Archives:         TrialArchives(t),
			UseFluentLogging: true,
		},
	}
	spec.RunSpec.HostConfig.ShmSize = t.TaskContainerDefaults.ShmSizeBytes
	if exp.ExperimentConfig.Resources.ShmSize != nil {
		spec.RunSpec.HostConfig.ShmSize = int64(*exp.ExperimentConfig.Resources.ShmSize)
	}
	return spec
}

// GCEnvVars returns environment variables for checkpoint gc.
func GCEnvVars() map[string]string {
	return defaultEnvVars()
}

// GCDockerMounts returns the host mounts for a gc container.
func GCDockerMounts(gcc GCCheckpoints) []mount.Mount {
	mounts := ToDockerMounts(gcc.ExperimentConfig.BindMounts)
	if gcc.ExperimentConfig.CheckpointStorage.SharedFSConfig != nil {
		sharedFS := gcc.ExperimentConfig.CheckpointStorage.SharedFSConfig
		mounts = append(mounts, mount.Mount{
			Type:   mount.TypeBind,
			Source: sharedFS.HostPath,
			Target: model.DefaultSharedFSContainerPath,
			BindOptions: &mount.BindOptions{
				Propagation: model.DefaultSharedFSPropagation,
			},
		})
	}

	return mounts
}

// GCArchives returns the additional files for gc as an archive.
func GCArchives(t TaskSpec) []container.RunArchive {
	gcc := *t.GCCheckpoints

	return []container.RunArchive{
		workDirArchive(gcc.AgentUserGroup),
		injectUserArchive(gcc.AgentUserGroup),
		wrapArchive(
			archive.Archive{
				gcc.AgentUserGroup.OwnedArchiveItem(
					"experiment_config.json",
					[]byte(jsonify(gcc.ExperimentConfig)),
					0600,
					tar.TypeReg,
				),
				gcc.AgentUserGroup.OwnedArchiveItem(
					"checkpoints_to_delete.json",
					[]byte(jsonify(gcc.ToDelete)),
					0600,
					tar.TypeReg,
				),
				gcc.AgentUserGroup.OwnedArchiveItem(
					etc.GCCheckpointsEntrypointResource,
					etc.MustStaticFile(etc.GCCheckpointsEntrypointResource),
					0700,
					tar.TypeReg,
				),
			},
			ContainerWorkDir,
		),
		harnessArchive(t.HarnessPath, gcc.AgentUserGroup),
	}
}

// GCCmd configures the entrypoint for GC tasks.
func GCCmd() []string {
	return []string{
		filepath.Join(ContainerWorkDir, etc.GCCheckpointsEntrypointResource),
		"--experiment-config",
		"experiment_config.json",
		"--delete",
		"checkpoints_to_delete.json",
	}
}

func gcCheckpoint(t TaskSpec) container.Spec {
	gcc := *t.GCCheckpoints
	user := getUser(gcc.AgentUserGroup)
	deviceType := device.CPU
	if len(t.Devices) > 0 {
		deviceType = t.Devices[0].Type
	}
	envVarsMap := GCEnvVars()
	envVars := make([]string, 0, len(envVarsMap))
	for envVarKey, envVarValue := range envVarsMap {
		envVars = append(envVars, fmt.Sprintf("%s=%s", envVarKey, envVarValue))
	}
	envVars = append(envVars, gcc.ExperimentConfig.Environment.EnvironmentVariables.For(deviceType)...)

	return container.Spec{
		PullSpec: container.PullSpec{
			ForcePull: gcc.ExperimentConfig.Environment.ForcePullImage,
			Registry:  gcc.ExperimentConfig.Environment.RegistryAuth,
		},
		RunSpec: container.RunSpec{
			ContainerConfig: docker.Config{
				Cmd: []string{
					filepath.Join(ContainerWorkDir, etc.GCCheckpointsEntrypointResource),
					"--experiment-config",
					"experiment_config.json",
					"--delete",
					"checkpoints_to_delete.json",
				},
				User:       user,
				Image:      gcc.ExperimentConfig.Environment.Image.For(deviceType),
				Env:        envVars,
				WorkingDir: ContainerWorkDir,
			},
			HostConfig: docker.HostConfig{
				NetworkMode:     t.TaskContainerDefaults.NetworkMode,
				Mounts:          GCDockerMounts(gcc),
				PublishAllPorts: true,
			},
			Archives: GCArchives(t),
		},
	}
}
