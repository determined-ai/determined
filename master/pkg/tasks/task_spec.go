package tasks

import (
	"archive/tar"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/docker/docker/api/types/mount"

	"github.com/determined-ai/determined/master/pkg/container"
	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/workload"

	"github.com/determined-ai/determined/master/pkg/archive"
	"github.com/determined-ai/determined/master/pkg/device"
	"github.com/determined-ai/determined/master/pkg/model"
)

// InnerSpec defines the interface for a particular kind of task container.
type InnerSpec interface {
	// Archives returns the files to include in the container for this task (apart from the base files
	// put into in all containers).
	Archives(*model.AgentUserGroup) []container.RunArchive
	// Description returns a brief description of this task.
	Description() string
	// Entrypoint returns the command and arguments to run in the container for this task.
	Entrypoint() []string
	// Environment returns the container environment for this task.
	Environment(TaskSpec) model.Environment
	// EnvVars returns the environment variables to set for this task (apart from the base ones set for
	// all containers).
	EnvVars(TaskSpec) map[string]string
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
}

// This alias allows TaskSpec to privately embed the public InnerSpec so that it can reuse (some of)
// its methods without directly providing access to the field to other packages.
type inner = InnerSpec

// TaskSpec provides the necessary information for a task to be run.
type TaskSpec struct {
	inner

	TaskID         string
	ContainerID    string
	Devices        []device.Device
	AgentUserGroup *model.AgentUserGroup

	ClusterID             string
	HarnessPath           string
	TaskContainerDefaults model.TaskContainerDefaultsConfig
	MasterCert            *tls.Certificate
}

// SetInner sets the concrete task represented by this spec.
func (t *TaskSpec) SetInner(inner InnerSpec) {
	t.inner = inner
}

func (t *TaskSpec) baseArchives() []container.RunArchive {
	return []container.RunArchive{
		workDirArchive(t.AgentUserGroup),
		injectUserArchive(t.AgentUserGroup),
		harnessArchive(t.HarnessPath, t.AgentUserGroup),
		masterCertArchive(t.MasterCert),
	}
}

func (t *TaskSpec) baseEnvVars() map[string]string {
	e := map[string]string{
		// PYTHONUSERBASE allows us to `pip install --user` into a location guaranteed to be owned by
		// the user inside the container.
		"PYTHONUSERBASE": userPythonBaseDir,
		"DET_TASK_ID":    t.TaskID,
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

	return e
}

// Archives returns the archives that should be included in the container for this task.
func (t *TaskSpec) Archives() []container.RunArchive {
	return append(t.baseArchives(), t.inner.Archives(t.AgentUserGroup)...)
}

// Environment returns the container environment for this task.
func (t *TaskSpec) Environment() model.Environment { return t.inner.Environment(*t) }

// EnvVars returns the environment variables that should be set in the container for this task.
func (t *TaskSpec) EnvVars() map[string]string {
	e := t.baseEnvVars()
	for k, v := range t.inner.EnvVars(*t) {
		e[k] = v
	}
	return e
}

// StartCommand is a description of a task for running a command.
type StartCommand struct {
	Config          model.CommandConfig
	UserFiles       archive.Archive
	AdditionalFiles archive.Archive
}

// Archives implements InnerSpec.
func (s StartCommand) Archives(u *model.AgentUserGroup) []container.RunArchive {
	return []container.RunArchive{
		wrapArchive(u.OwnArchive(s.UserFiles), ContainerWorkDir),
		wrapArchive(s.AdditionalFiles, rootDir),
	}
}

// Description implements InnerSpec.
func (s StartCommand) Description() string { return "cmd" }

// Entrypoint implements InnerSpec.
func (s StartCommand) Entrypoint() []string { return s.Config.Entrypoint }

// Environment implements InnerSpec.
func (s StartCommand) Environment(TaskSpec) model.Environment { return s.Config.Environment }

// EnvVars implements InnerSpec.
func (s StartCommand) EnvVars(TaskSpec) map[string]string { return nil }

// LoggingFields implements InnerSpec.
func (s StartCommand) LoggingFields() map[string]string { return nil }

// Mounts implements InnerSpec.
func (s StartCommand) Mounts() []mount.Mount { return ToDockerMounts(s.Config.BindMounts) }

// ShmSize implements InnerSpec.
func (s StartCommand) ShmSize() int64 {
	if shm := s.Config.Resources.ShmSize; shm != nil {
		return int64(*shm)
	}
	return 0
}

// UseFluentLogging implements InnerSpec.
func (s StartCommand) UseFluentLogging() bool { return false }

// UseHostMode implements InnerSpec.
func (s StartCommand) UseHostMode() bool { return false }

// GCCheckpoints is a description of a task for running checkpoint GC.
type GCCheckpoints struct {
	ExperimentID     int
	ExperimentConfig model.ExperimentConfig
	ToDelete         json.RawMessage
}

// Archives implements InnerSpec.
func (g GCCheckpoints) Archives(u *model.AgentUserGroup) []container.RunArchive {
	return []container.RunArchive{
		wrapArchive(
			archive.Archive{
				u.OwnedArchiveItem(
					"experiment_config.json",
					[]byte(jsonify(g.ExperimentConfig)),
					0600,
					tar.TypeReg,
				),
				u.OwnedArchiveItem(
					"checkpoints_to_delete.json",
					[]byte(jsonify(g.ToDelete)),
					0600,
					tar.TypeReg,
				),
				u.OwnedArchiveItem(
					etc.GCCheckpointsEntrypointResource,
					etc.MustStaticFile(etc.GCCheckpointsEntrypointResource),
					0700,
					tar.TypeReg,
				),
			},
			ContainerWorkDir,
		),
	}
}

// Description implements InnerSpec.
func (g GCCheckpoints) Description() string { return "gc" }

// Entrypoint implements InnerSpec.
func (g GCCheckpoints) Entrypoint() []string {
	return []string{
		filepath.Join(ContainerWorkDir, etc.GCCheckpointsEntrypointResource),
		"--experiment-config",
		"experiment_config.json",
		"--delete",
		"checkpoints_to_delete.json",
	}
}

// Environment implements InnerSpec.
func (g GCCheckpoints) Environment(TaskSpec) model.Environment {
	return g.ExperimentConfig.Environment
}

// EnvVars implements InnerSpec.
func (g GCCheckpoints) EnvVars(TaskSpec) map[string]string { return nil }

// LoggingFields implements InnerSpec.
func (g GCCheckpoints) LoggingFields() map[string]string { return nil }

// Mounts implements InnerSpec.
func (g GCCheckpoints) Mounts() []mount.Mount {
	mounts := ToDockerMounts(g.ExperimentConfig.BindMounts)
	if fs := g.ExperimentConfig.CheckpointStorage.SharedFSConfig; fs != nil {
		mounts = append(mounts, mount.Mount{
			Type:   mount.TypeBind,
			Source: fs.HostPath,
			Target: model.DefaultSharedFSContainerPath,
			BindOptions: &mount.BindOptions{
				Propagation: model.DefaultSharedFSPropagation,
			},
		})
	}
	return mounts
}

// ShmSize implements InnerSpec.
func (g GCCheckpoints) ShmSize() int64 { return 0 }

// UseFluentLogging implements InnerSpec.
func (g GCCheckpoints) UseFluentLogging() bool { return false }

// UseHostMode implements InnerSpec.
func (g GCCheckpoints) UseHostMode() bool { return false }

// StartTrial is a description of a task for running a trial container.
type StartTrial struct {
	ExperimentConfig    model.ExperimentConfig
	ModelDefinition     archive.Archive
	HParams             map[string]interface{}
	TrialSeed           uint32
	LatestCheckpoint    *model.Checkpoint
	InitialWorkload     workload.Workload
	WorkloadManagerType model.WorkloadManagerType
	AdditionalFiles     archive.Archive

	// This is used to hint the resource manager to override defaults and start
	// the container in host mode iff it has been scheduled across multiple agents.
	IsMultiAgent bool

	Rank int
}

// Archives implements InnerSpec.
func (s StartTrial) Archives(u *model.AgentUserGroup) []container.RunArchive {
	return []container.RunArchive{wrapArchive(s.AdditionalFiles, rootDir),
		wrapArchive(
			archive.Archive{
				u.OwnedArchiveItem(
					"checkpoint.json",
					[]byte(jsonify(s.LatestCheckpoint)),
					0600,
					tar.TypeReg,
				),
			},
			trainDir,
		),
		wrapArchive(u.OwnArchive(s.ModelDefinition), ContainerWorkDir),
	}
}

// Description implements InnerSpec.
func (s StartTrial) Description() string {
	return fmt.Sprintf(
		"exp-%d-trial-%d-rank-%d",
		s.InitialWorkload.ExperimentID,
		s.InitialWorkload.TrialID,
		s.Rank,
	)
}

// Entrypoint implements InnerSpec.
func (s StartTrial) Entrypoint() []string {
	return []string{"/run/determined/train/entrypoint.sh"}
}

// Environment implements InnerSpec.
func (s StartTrial) Environment(t TaskSpec) model.Environment {
	env := s.ExperimentConfig.Environment
	if env.Ports == nil {
		env.Ports = make(map[string]int)
	}
	for i, port := range rendezvousPorts(trialUniquePortOffset(t.Devices)) {
		env.Ports[fmt.Sprintf("trial-%d", i)] = port
	}
	return env
}

// EnvVars implements InnerSpec.
func (s StartTrial) EnvVars(t TaskSpec) map[string]string {
	portOffset := trialUniquePortOffset(t.Devices)
	var portStrs []string
	for _, port := range rendezvousPorts(portOffset) {
		portStrs = append(portStrs, strconv.Itoa(port))
	}
	return map[string]string{
		"DET_EXPERIMENT_ID":            fmt.Sprintf("%d", s.InitialWorkload.ExperimentID),
		"DET_TRIAL_ID":                 fmt.Sprintf("%d", s.InitialWorkload.TrialID),
		"DET_TRIAL_SEED":               fmt.Sprintf("%d", s.TrialSeed),
		"DET_EXPERIMENT_CONFIG":        jsonify(s.ExperimentConfig),
		"DET_HPARAMS":                  jsonify(s.HParams),
		"DET_INITIAL_WORKLOAD":         jsonify(s.InitialWorkload),
		"DET_LATEST_CHECKPOINT":        "/run/determined/train/checkpoint.json",
		"DET_WORKLOAD_MANAGER_TYPE":    string(s.WorkloadManagerType),
		"DET_RENDEZVOUS_PORTS":         strings.Join(portStrs, ","),
		"DET_TRIAL_UNIQUE_PORT_OFFSET": strconv.Itoa(portOffset),
	}
}

// LoggingFields implements InnerSpec.
func (s StartTrial) LoggingFields() map[string]string {
	return map[string]string{
		"trial_id": strconv.Itoa(s.InitialWorkload.TrialID),
	}
}

// Mounts implements InnerSpec.
func (s StartTrial) Mounts() []mount.Mount {
	mounts := ToDockerMounts(s.ExperimentConfig.BindMounts)
	addMount := func(source, target string, bindOpts *mount.BindOptions) {
		mounts = append(mounts, mount.Mount{
			Type: mount.TypeBind, Source: source, Target: target, BindOptions: bindOpts,
		})
	}

	if c := s.ExperimentConfig.CheckpointStorage.SharedFSConfig; c != nil {
		addMount(
			c.HostPath,
			model.DefaultSharedFSContainerPath,
			&mount.BindOptions{Propagation: model.DefaultSharedFSPropagation},
		)
	}

	if c := s.ExperimentConfig.DataLayer.SharedFSConfig; c != nil {
		if c.HostStoragePath != nil && c.ContainerStoragePath != nil {
			addMount(*c.HostStoragePath, *c.ContainerStoragePath, nil)
		}
	}
	if c := s.ExperimentConfig.DataLayer.S3Config; c != nil {
		if c.LocalCacheHostPath != nil && c.LocalCacheContainerPath != nil {
			addMount(*c.LocalCacheHostPath, *c.LocalCacheContainerPath, nil)
		}
	}
	if c := s.ExperimentConfig.DataLayer.GCSConfig; c != nil {
		if c.LocalCacheHostPath != nil && c.LocalCacheContainerPath != nil {
			addMount(*c.LocalCacheHostPath, *c.LocalCacheContainerPath, nil)
		}
	}

	return mounts
}

// UseFluentLogging implements InnerSpec.
func (s StartTrial) UseFluentLogging() bool { return true }

// UseHostMode implements InnerSpec.
func (s StartTrial) UseHostMode() bool { return s.IsMultiAgent }

// ShmSize implements InnerSpec.
func (s StartTrial) ShmSize() int64 {
	if shm := s.ExperimentConfig.Resources.ShmSize; shm != nil {
		return int64(*shm)
	}
	return 0
}
