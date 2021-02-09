package tasks

import (
	"archive/tar"
	"fmt"

	docker "github.com/docker/docker/api/types/container"

	"github.com/determined-ai/determined/master/pkg/archive"
	"github.com/determined-ai/determined/master/pkg/container"
	"github.com/determined-ai/determined/master/pkg/device"
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

// ToContainerSpec translates a task spec into a generic container spec.
func ToContainerSpec(t TaskSpec) container.Spec {
	var envVars []string
	for k, v := range t.EnvVars() {
		envVars = append(envVars, fmt.Sprintf("%s=%s", k, v))
	}

	env := t.Environment()
	deviceType := device.CPU
	if len(t.Devices) > 0 {
		deviceType = t.Devices[0].Type
	}
	envVars = append(envVars, env.EnvironmentVariables.For(deviceType)...)

	network := t.TaskContainerDefaults.NetworkMode
	if t.UseHostMode() {
		network = hostMode
	}

	shmSize := t.ShmSize()
	if shmSize == 0 {
		shmSize = t.TaskContainerDefaults.ShmSizeBytes
	}

	spec := container.Spec{
		PullSpec: container.PullSpec{
			Registry:  env.RegistryAuth,
			ForcePull: env.ForcePullImage,
		},
		RunSpec: container.RunSpec{
			ContainerConfig: docker.Config{
				User:         getUser(t.AgentUserGroup),
				ExposedPorts: toPortSet(env.Ports),
				Env:          envVars,
				Cmd:          t.Entrypoint(),
				Image:        env.Image.For(deviceType),
				WorkingDir:   ContainerWorkDir,
			},
			HostConfig: docker.HostConfig{
				NetworkMode:     network,
				Mounts:          t.Mounts(),
				PublishAllPorts: true,
				ShmSize:         shmSize,
			},
			Archives:         t.Archives(),
			UseFluentLogging: t.UseFluentLogging(),
		},
	}

	return spec
}

func getUser(agentUserGroup *model.AgentUserGroup) string {
	if agentUserGroup == nil {
		return ""
	}
	return fmt.Sprintf("%d:%d", agentUserGroup.UID, agentUserGroup.GID)
}
