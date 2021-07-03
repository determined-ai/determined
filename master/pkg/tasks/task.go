package tasks

import (
	"archive/tar"
	"crypto/tls"
	"fmt"

	docker "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"

	"github.com/determined-ai/determined/master/pkg/archive"
	"github.com/determined-ai/determined/master/pkg/container"
	"github.com/determined-ai/determined/master/pkg/device"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
)

const (
	// ContainerWorkDir is the working directory for tasks.
	ContainerWorkDir  = "/run/determined/workdir"
	userPythonBaseDir = "/run/determined/pythonuserbase"
	runDir            = "/run/determined"
	trainDir          = "/run/determined/train"
	modelCopy         = "/run/determined/train/model"
	rootDir           = "/"
	passwdPath        = "/run/determined/etc/passwd"
	shadowPath        = "/run/determined/etc/shadow"
	groupPath         = "/run/determined/etc/group"
	certPath          = "/run/determined/etc/ssl/master.crt"
)

// TaskContainer defines the interface for a particular kind of task container.
type TaskContainer interface {
	// Archives returns the files to include in the container for this task (apart from the base files
	// put into in all containers).
	ExtraArchives(*model.AgentUserGroup) []container.RunArchive
	// Description returns a brief description of this task.
	Description() string
	// Entrypoint returns the command and arguments to run in the container for this task.
	Entrypoint() []string
	// Environment returns the container environment for this task.
	Environment() expconf.EnvironmentConfig
	// EnvVars returns the environment variables to set for this task (apart from the base ones set for
	// all containers).
	ExtraEnvVars() map[string]string
	// LoggingFields returns fields to include in each record of structured (i.e., Fluent Bit) logging.
	LoggingFields() map[string]string
	// Mounts returns the list of Docker mounts to use for this task.
	Mounts() []mount.Mount
	// ShmSize specifies the shared memory size to allocate to this task's container in bytes (0 for
	// default behavior).
	ShmSize() int64
	// UseFluentLogging specifies whether to use Fluent Bit logging (as opposed to native logging).
	UseFluentLogging() bool
	// UseHostMode indicates whether host mode networking would be desirable for this task.
	UseHostMode() bool
	//ResourcesConfig returns the resources config of the model
	ResourcesConfig() expconf.ResourcesConfig
}

// TaskSpec defines the spec of a task.
type TaskSpec struct {
	TaskContainer

	TaskID         string
	TaskToken      string
	ContainerID    string
	Devices        []device.Device
	AgentUserGroup *model.AgentUserGroup

	ClusterID             string
	HarnessPath           string
	TaskContainerDefaults model.TaskContainerDefaultsConfig
	MasterCert            *tls.Certificate
}

// Archives returns all the archives.
func (t *TaskSpec) Archives() []container.RunArchive {
	res := []container.RunArchive{
		workDirArchive(t.AgentUserGroup),
		injectUserArchive(t.AgentUserGroup),
		harnessArchive(t.HarnessPath, t.AgentUserGroup),
		masterCertArchive(t.MasterCert),
	}
	res = append(res, t.TaskContainer.ExtraArchives(t.AgentUserGroup)...)
	return res
}

// EnvVars returns all the environment variables.
func (t *TaskSpec) EnvVars() map[string]string {
	e := map[string]string{
		// PYTHONUSERBASE allows us to `pip install --user` into a location guaranteed to be owned by
		// the user inside the container.
		"PYTHONUSERBASE": userPythonBaseDir,
		"DET_TASK_ID":    t.TaskID,
		"DET_TASK_TOKEN": t.TaskToken,
	}
	if t.TaskContainerDefaults.NCCLPortRange != "" {
		e["NCCL_PORT_RANGE"] = t.TaskContainerDefaults.NCCLPortRange
	}
	if t.TaskContainerDefaults.NCCLPortRange != "" {
		e["GLOO_PORT_RANGE"] = t.TaskContainerDefaults.NCCLPortRange
	}

	networkInterface := t.TaskContainerDefaults.DtrainNetworkInterface
	if networkInterface == "" {
		networkInterface = "DET_AUTO_DETECT_NETWORK_INTERFACE"
	}
	e["DET_TRIAL_RUNNER_NETWORK_INTERFACE"] = networkInterface

	if t.MasterCert != nil {
		e["DET_USE_TLS"] = "true"
		e["DET_MASTER_CERT_FILE"] = certPath
	}

	for k, v := range t.TaskContainer.ExtraEnvVars() {
		e[k] = v
	}
	return e
}

// ToContainerSpec converts a task spec to a docker container spec.
func (t *TaskSpec) ToContainerSpec() container.Spec {
	var envVars []string
	for k, v := range t.EnvVars() {
		envVars = append(envVars, fmt.Sprintf("%s=%s", k, v))
	}

	env := t.Environment()
	deviceType := device.CPU
	if len(t.Devices) > 0 {
		deviceType = t.Devices[0].Type
	}
	envVars = append(envVars, env.EnvironmentVariables().For(deviceType)...)

	network := t.TaskContainerDefaults.NetworkMode
	if t.UseHostMode() {
		network = hostMode
	}

	shmSize := t.ShmSize()
	if shmSize == 0 {
		shmSize = t.TaskContainerDefaults.ShmSizeBytes
	}

	resources := t.ResourcesConfig()
	var devices []docker.DeviceMapping
	for _, device := range resources.Devices() {
		devices = append(devices, docker.DeviceMapping{
			PathOnHost:        device.HostPath(),
			PathInContainer:   device.ContainerPath(),
			CgroupPermissions: device.Mode(),
		})
	}

	spec := container.Spec{
		PullSpec: container.PullSpec{
			Registry:  env.RegistryAuth(),
			ForcePull: env.ForcePullImage(),
		},
		RunSpec: container.RunSpec{
			ContainerConfig: docker.Config{
				User:         getUser(t.AgentUserGroup),
				ExposedPorts: toPortSet(env.Ports()),
				Env:          envVars,
				Cmd:          t.Entrypoint(),
				Image:        env.Image().For(deviceType),
				WorkingDir:   ContainerWorkDir,
			},
			HostConfig: docker.HostConfig{
				NetworkMode:     network,
				Mounts:          t.Mounts(),
				PublishAllPorts: true,
				ShmSize:         shmSize,
				CapAdd:          env.AddCapabilities(),
				CapDrop:         env.DropCapabilities(),

				Resources: docker.Resources{
					Devices: devices,
				},
			},
			Archives:         t.Archives(),
			UseFluentLogging: t.UseFluentLogging(),
		},
	}

	return spec
}

// workDirArchive ensures that the workdir is created and owned by the user.
func workDirArchive(aug *model.AgentUserGroup) container.RunArchive {
	return wrapArchive(
		archive.Archive{
			aug.OwnedArchiveItem(runDir, nil, 0700, tar.TypeDir),
			aug.OwnedArchiveItem(ContainerWorkDir, nil, 0700, tar.TypeDir),
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
		fmt.Sprintf("%v:x:%v:%v::%v:/bin/bash\n", aug.User, aug.UID, aug.GID, ContainerWorkDir),
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

func getUser(agentUserGroup *model.AgentUserGroup) string {
	if agentUserGroup == nil {
		return ""
	}
	return fmt.Sprintf("%d:%d", agentUserGroup.UID, agentUserGroup.GID)
}
