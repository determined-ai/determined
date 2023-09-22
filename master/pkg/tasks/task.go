package tasks

import (
	"archive/tar"
	"encoding/json"
	"fmt"
	"path/filepath"
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

// File location constants.
const (
	// DefaultWorkDir is the default workdir.
	DefaultWorkDir    = "/run/determined/workdir"
	userPythonBaseDir = "/run/determined/pythonuserbase"
	RunDir            = "/run/determined"
	infoDir           = "/run/determined/info"
	trainDir          = "/run/determined/train"
	modelCopy         = "/run/determined/train/model"
	rootDir           = "/"
	PasswdPath        = "/run/determined/etc/passwd"
	ShadowPath        = "/run/determined/etc/shadow"
	GroupPath         = "/run/determined/etc/group"
	certPath          = "/run/determined/etc/ssl/master.crt"
	// DtrainSSHPortBase is starting range for Dtrain ports.
	DtrainSSHPortBase = 12350
	// InterTrainProcessCommPort1Base is starting range for intertraincomm1 ports.
	InterTrainProcessCommPort1Base = 12360
	// InterTrainProcessCommPort2Base is starting range for intertraincomm2 ports.
	InterTrainProcessCommPort2Base = 12365
	// C10DPortBase is starting range for c10D ports.
	C10DPortBase = 29400
	// DTrainSSHPort is the name of a port.
	DTrainSSHPort = "DTRAIN_SSH_PORT"
	// InterTrainProcessCommPort1 is the name of a port.
	InterTrainProcessCommPort1 = "INTER_TRAIN_PROCESS_COMM_PORT_1"
	// InterTrainProcessCommPort2 is the name of a port.
	InterTrainProcessCommPort2 = "INTER_TRAIN_PROCESS_COMM_PORT_2"
	// C10DPort is the name of a port.
	C10DPort = "C10D_PORT"
)

// TaskSpecifier creates a TaskSpec. ToTaskSpec must only be called once per specifier.
type TaskSpecifier interface {
	ToTaskSpec() TaskSpec
}

// TaskSpec defines the spec of a task.
type TaskSpec struct {
	// Fields that are only for task logics.
	Description string
	// LoggingFields are fields to include in each record of structured logging.
	LoggingFields map[string]string

	// Fields that are set on the cluster level.
	ClusterID   string
	HarnessPath string
	MasterCert  []byte
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
	SlurmConfig      expconf.SlurmConfig
	PbsConfig        expconf.PbsConfig

	ExtraProxyPorts expconf.ProxyPortsConfig

	Workspace string
	Project   string
	Labels    []string
	// Ports required by trial or commands and their respective base port values.
	UniqueExposedPortRequests map[string]int
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
func (t *TaskSpec) Archives() ([]cproto.RunArchive, []cproto.RunArchive) {
	res := []cproto.RunArchive{
		workDirArchive(t.AgentUserGroup, t.WorkDir, t.WorkDir == DefaultWorkDir),
		runDirHelpersArchive(t.AgentUserGroup),
		injectUserArchive(t.AgentUserGroup, t.WorkDir),
		harnessArchive(t.HarnessPath, t.AgentUserGroup),
		masterCertArchive(t.MasterCert),
	}
	res = append(res, t.ExtraArchives...)

	// Split into root and non root required files. In the case the user
	// is root we will still differentiate files that need to be root
	// versus files that should be owned by the user.
	var user, root []cproto.RunArchive
	for _, a := range res {
		var uItems, rItems archive.Archive
		for _, item := range a.Archive {
			if item.IsRootItem {
				rItems = append(rItems, item)
			} else {
				uItems = append(uItems, item)
			}
		}

		if len(rItems) > 0 {
			root = append(root, cproto.RunArchive{
				Path:        a.Path,
				CopyOptions: a.CopyOptions,
				Archive:     rItems,
			})
		}
		if len(uItems) > 0 {
			user = append(user, cproto.RunArchive{
				Path:        a.Path,
				CopyOptions: a.CopyOptions,
				Archive:     uItems,
			})
		}
	}
	return user, root
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
		"DET_WORKDIR":       t.WorkDir,
	}
	if t.Owner != nil {
		e["DET_USER"] = t.Owner.Username
	}

	if t.TaskContainerDefaults.GLOOPortRange != "" {
		e["GLOO_PORT_RANGE"] = t.TaskContainerDefaults.GLOOPortRange
	}

	networkInterface := t.TaskContainerDefaults.DtrainNetworkInterface
	if networkInterface != "" {
		e["DET_INTER_NODE_NETWORK_INTERFACE"] = networkInterface
	}

	if len(t.MasterCert) != 0 {
		e["DET_USE_TLS"] = "true"
		e["DET_MASTER_CERT_FILE"] = certPath
	} else {
		e["DET_USE_TLS"] = "false"
	}

	e["DET_SEGMENT_ENABLED"] = fmt.Sprintf("%v", t.SegmentEnabled)
	if t.SegmentEnabled {
		e["DET_SEGMENT_API_KEY"] = t.SegmentAPIKey
	}

	if t.LoggingFields != nil {
		j, err := json.Marshal(t.LoggingFields)
		if err != nil {
			// TODO(DET-7565): propagate errors.
			panic(fmt.Errorf("serializing logging fields: %w", err))
		}
		e["DET_TASK_LOGGING_METADATA"] = string(j)
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

	// Prepend the entrypoint like: `ship-logs.sh "$@"`.
	shipLogsShell := filepath.Join(RunDir, taskShipLogsShell)
	shipLogsPython := filepath.Join(RunDir, taskShipLogsPython)
	entrypoint := append([]string{shipLogsShell, shipLogsPython}, t.Entrypoint...)

	runArchives, rootArchives := t.Archives()
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
				Cmd:          entrypoint,
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
			Archives:   append(runArchives, rootArchives...),
			DeviceType: deviceType,
			Registry:   env.RegistryAuth(),
		},
	}

	return spec
}

// workDirArchive ensures that the workdir is created and owned by the user.
func workDirArchive(
	aug *model.AgentUserGroup, workDir string, createWorkDir bool,
) cproto.RunArchive {
	a := archive.Archive{
		aug.OwnedArchiveItem(RunDir, nil, 0o700, tar.TypeDir),
		aug.OwnedArchiveItem(infoDir, nil, 0o755, tar.TypeDir),
		aug.OwnedArchiveItem(userPythonBaseDir, nil, 0o700, tar.TypeDir),
	}
	if createWorkDir {
		a = append(a, aug.OwnedArchiveItem(workDir, nil, 0o700, tar.TypeDir))
	}
	return wrapArchive(a, rootDir)
}

// runDirHelpersArchive ensures helper scripts exist in the run dir.
func runDirHelpersArchive(aug *model.AgentUserGroup) cproto.RunArchive {
	return wrapArchive(archive.Archive{
		aug.OwnedArchiveItem(
			taskSetupScript,
			etc.MustStaticFile(etc.TaskSetupScriptResource),
			taskSetupMode,
			tar.TypeReg,
		),
		aug.OwnedArchiveItem(
			taskShipLogsShell,
			etc.MustStaticFile(etc.TaskShipLogsShellResource),
			taskShipLogsShellMode,
			tar.TypeReg,
		),
		aug.OwnedArchiveItem(
			taskShipLogsPython,
			etc.MustStaticFile(etc.TaskShipLogsPythonResource),
			taskShipLogsPythonMode,
			tar.TypeReg,
		),
		aug.OwnedArchiveItem(
			SingularityEntrypointWrapperScript,
			etc.MustStaticFile(etc.SingularityEntrypointWrapperScriptResource),
			singularityEntrypointWrapperMode,
			tar.TypeReg,
		),
	}, RunDir)
}

// injectUserArchive creates the user/UID/group/GID for a user by adding passwd/shadow/group files
// to /run/determined/etc, which will be read by libnss_determined inside the container. If
// libnss_determined is not present in the container, these files will be simply ignored and some
// non-root container features will not work properly.
func injectUserArchive(aug *model.AgentUserGroup, workDir string) cproto.RunArchive {
	passwdBytes := []byte(
		fmt.Sprintf("%v:x:%v:%v::%v:/bin/bash\n", aug.User, aug.UID, aug.GID, workDir),
	)
	// Disable login via password by * in shadow file.  Cannot use ! as that locks the account
	// when using SLURM/Singularity.
	shadowBytes := []byte(fmt.Sprintf("%v:*:::::::\n", aug.User))
	groupBytes := []byte(fmt.Sprintf("%v:x:%v:\n", aug.Group, aug.GID))

	return wrapArchive(
		archive.Archive{
			archive.RootItem(PasswdPath, passwdBytes, 0o644, tar.TypeReg),
			archive.RootItem(ShadowPath, shadowBytes, 0o600, tar.TypeReg),
			archive.RootItem(GroupPath, groupBytes, 0o644, tar.TypeReg),
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
