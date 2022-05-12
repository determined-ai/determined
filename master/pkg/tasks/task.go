package tasks

import (
	"archive/tar"
	"crypto/tls"
	"fmt"
	"strings"

	docker "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"

	"github.com/determined-ai/determined/master/pkg/archive"
	"github.com/determined-ai/determined/master/pkg/cproto"
	"github.com/determined-ai/determined/master/pkg/device"
	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
)

const (
	// DefaultWorkDir is the default workdir.
	DefaultWorkDir    = "/run/determined/workdir"
	userPythonBaseDir = "/run/determined/pythonuserbase"
	runDir            = "/run/determined"
	infoDir           = "/run/determined/info"
	trainDir          = "/run/determined/train"
	modelCopy         = "/run/determined/train/model"
	rootDir           = "/"
	passwdPath        = "/run/determined/etc/passwd"
	shadowPath        = "/run/determined/etc/shadow"
	groupPath         = "/run/determined/etc/group"
	certPath          = "/run/determined/etc/ssl/master.crt"
)

// TaskSpec defines the spec of a task.
type TaskSpec struct {
	// Fields that are only for task logics.
	Description string
	// LoggingFields are fields to include in each record of structured (i.e., Fluent Bit) logging.
	LoggingFields map[string]string
	// UseFluentLogging is whether to use Fluent Bit logging (as opposed to directly streaming).
	UseFluentLogging bool

	// Fields that are set on the cluster level.
	ClusterID   string
	HarnessPath string
	MasterCert  *tls.Certificate
	SSHRsaSize  int

	SegmentEnabled bool
	SegmentAPIKey  string

	// Fields that are set on the per-request basis.
	// TaskContainerDefaults should be removed from TaskSpec once we move to using the same
	// schema for the cluster-level defaults and the request-level configuration.
	TaskContainerDefaults model.TaskContainerDefaultsConfig
	Environment           expconf.EnvironmentConfig
	ResourcesConfig       expconf.ResourcesConfig
	WorkDir               string
	Owner                 *model.User
	AgentUserGroup        *model.AgentUserGroup
	ExtraArchives         []cproto.RunArchive
	ExtraEnvVars          map[string]string
	Entrypoint            []string
	Mounts                []mount.Mount
	// UseHostMode is whether host mode networking would be desirable for this task.
	// This is used by Docker only.
	UseHostMode bool
	ShmSize     int64

	// The parent task of an allocation.
	TaskID string

	// Fields that are set on per-resources basis.
	AllocationID           string
	AllocationSessionToken string
	ResourcesID            string
	ContainerID            string
	Devices                []device.Device

	UserSessionToken string
	TaskType         model.TaskType
}

// ResolveWorkDir resolves the work dir.
func (t *TaskSpec) ResolveWorkDir() {
	agentUser := ""
	detUser := ""
	if t.AgentUserGroup != nil {
		agentUser = t.AgentUserGroup.User
	}
	if t.Owner != nil {
		detUser = t.Owner.Username
	}
	workDir := strings.ReplaceAll(t.WorkDir, "$AGENT_USER", agentUser)
	t.WorkDir = strings.ReplaceAll(workDir, "$DET_USER", detUser)
}

// Archives returns all the archives.
func (t *TaskSpec) Archives() []cproto.RunArchive {
	res := []cproto.RunArchive{
		workDirArchive(t.AgentUserGroup, t.WorkDir, t.WorkDir == DefaultWorkDir),
		runDirHelpersArchive(t.AgentUserGroup),
		injectUserArchive(t.AgentUserGroup, t.WorkDir),
		harnessArchive(t.HarnessPath, t.AgentUserGroup),
		masterCertArchive(t.MasterCert),
	}
	res = append(res, t.ExtraArchives...)
	return res
}

// EnvVars returns all the environment variables.
func (t TaskSpec) EnvVars() map[string]string {
	e := map[string]string{
		// PYTHONUSERBASE allows us to `pip install --user` into a location guaranteed to be owned by
		// the user inside the container.
		"PYTHONUSERBASE":    userPythonBaseDir,
		"DET_TASK_ID":       t.TaskID,
		"DET_ALLOCATION_ID": t.AllocationID,
		"DET_RESOURCES_ID":  t.ResourcesID,
		"DET_CONTAINER_ID":  t.ContainerID,
		"DET_SESSION_TOKEN": t.AllocationSessionToken,
		"DET_USER_TOKEN":    t.UserSessionToken,
		"DET_USER":          t.Owner.Username,
	}
	if t.TaskContainerDefaults.NCCLPortRange != "" {
		e["NCCL_PORT_RANGE"] = t.TaskContainerDefaults.NCCLPortRange
	}
	if t.TaskContainerDefaults.NCCLPortRange != "" {
		e["GLOO_PORT_RANGE"] = t.TaskContainerDefaults.NCCLPortRange
	}

	networkInterface := t.TaskContainerDefaults.DtrainNetworkInterface
	if networkInterface != "" {
		e["DET_INTER_NODE_NETWORK_INTERFACE"] = networkInterface
	}

	if t.MasterCert != nil {
		e["DET_USE_TLS"] = "true"
		e["DET_MASTER_CERT_FILE"] = certPath
	} else {
		e["DET_USE_TLS"] = "false"
	}

	e["DET_SEGMENT_ENABLED"] = fmt.Sprintf("%v", t.SegmentEnabled)
	if t.SegmentEnabled {
		e["DET_SEGMENT_API_KEY"] = t.SegmentAPIKey
	}

	for k, v := range t.ExtraEnvVars {
		e[k] = v
	}
	return e
}

// ToDockerSpec converts a task spec to a docker container spec.
func (t *TaskSpec) ToDockerSpec() cproto.Spec {
	var envVars []string
	for k, v := range t.EnvVars() {
		envVars = append(envVars, fmt.Sprintf("%s=%s", k, v))
	}

	env := t.Environment
	deviceType := device.CPU
	if len(t.Devices) > 0 {
		deviceType = t.Devices[0].Type
	}
	envVars = append(envVars, env.EnvironmentVariables().For(deviceType)...)

	network := t.TaskContainerDefaults.NetworkMode
	if t.UseHostMode {
		network = hostMode
	}

	shmSize := t.ShmSize
	if shmSize == 0 {
		shmSize = t.TaskContainerDefaults.ShmSizeBytes
	}

	resources := t.ResourcesConfig
	var devices []docker.DeviceMapping
	for _, device := range resources.Devices() {
		devices = append(devices, docker.DeviceMapping{
			PathOnHost:        device.HostPath(),
			PathInContainer:   device.ContainerPath(),
			CgroupPermissions: device.Mode(),
		})
	}

	spec := cproto.Spec{
		TaskType: string(t.TaskType),
		PullSpec: cproto.PullSpec{
			Registry:  env.RegistryAuth(),
			ForcePull: env.ForcePullImage(),
		},
		RunSpec: cproto.RunSpec{
			ContainerConfig: docker.Config{
				User:         getUser(t.AgentUserGroup),
				ExposedPorts: toPortSet(env.Ports()),
				Env:          envVars,
				Cmd:          t.Entrypoint,
				Image:        env.Image().For(deviceType),
				WorkingDir:   t.WorkDir,
			},
			HostConfig: docker.HostConfig{
				NetworkMode:     network,
				Mounts:          t.Mounts,
				PublishAllPorts: true,
				ShmSize:         shmSize,
				CapAdd:          env.AddCapabilities(),
				CapDrop:         env.DropCapabilities(),

				Resources: docker.Resources{
					Devices: devices,
				},
			},
			Archives:         t.Archives(),
			UseFluentLogging: t.UseFluentLogging,
		},
	}

	return spec
}

// workDirArchive ensures that the workdir is created and owned by the user.
func workDirArchive(
	aug *model.AgentUserGroup, workDir string, createWorkDir bool,
) cproto.RunArchive {
	a := archive.Archive{
		aug.OwnedArchiveItem(runDir, nil, 0700, tar.TypeDir),
		aug.OwnedArchiveItem(infoDir, nil, 0755, tar.TypeDir),
		aug.OwnedArchiveItem(userPythonBaseDir, nil, 0700, tar.TypeDir),
	}
	if createWorkDir {
		a = append(a, aug.OwnedArchiveItem(workDir, nil, 0700, tar.TypeDir))
	}
	return wrapArchive(a, rootDir)
}

// runDirHelpersArchive ensures helper scripts exist in the run dir.
func runDirHelpersArchive(aug *model.AgentUserGroup) cproto.RunArchive {
	return wrapArchive(archive.Archive{
		aug.OwnedArchiveItem(
			taskLoggingSetupScript,
			etc.MustStaticFile(etc.TaskLoggingSetupScriptResource),
			taskLoggingSetupMode,
			tar.TypeReg,
		),
		aug.OwnedArchiveItem(
			taskLoggingTeardownScript,
			etc.MustStaticFile(etc.TaskLoggingTeardownScriptResource),
			taskLoggingTeardownMode,
			tar.TypeReg,
		),
		aug.OwnedArchiveItem(
			taskSignalHandlingScript,
			etc.MustStaticFile(etc.TaskSignalHandlingScriptResource),
			taskSignalHandlingMode,
			tar.TypeReg,
		),
	}, runDir)
}

// injectUserArchive creates the user/UID/group/GID for a user by adding passwd/shadow/group files
// to /run/determined/etc, which will be read by libnss_determined inside the container. If
// libnss_determined is not present in the container, these files will be simply ignored and some
// non-root container features will not work properly.
func injectUserArchive(aug *model.AgentUserGroup, workDir string) cproto.RunArchive {
	passwdBytes := []byte(
		fmt.Sprintf("%v:x:%v:%v::%v:/bin/bash\n", aug.User, aug.UID, aug.GID, workDir),
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
