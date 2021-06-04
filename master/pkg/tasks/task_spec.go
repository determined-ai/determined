package tasks

import (
	"crypto/tls"

	"github.com/docker/docker/api/types/mount"

	"github.com/determined-ai/determined/master/pkg/container"
	"github.com/determined-ai/determined/master/pkg/device"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
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
	Environment(TaskSpec) expconf.EnvironmentConfig
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
	//ResourcesConfig returns the resources config of the model
	ResourcesConfig() expconf.ResourcesConfig
}

// This alias allows TaskSpec to privately embed the public InnerSpec so that it can reuse (some of)
// its methods without directly providing access to the field to other packages.
type inner = InnerSpec

// TaskSpec provides the necessary information for a task to be run.
// It will be transformed into a pod spec or a container spec.
type TaskSpec struct {
	// These fields are set based on the cluster
	ClusterID   string
	HarnessPath string
	MasterCert  *tls.Certificate

	// These fields are set based on the user request.
	TaskContainerDefaults model.TaskContainerDefaultsConfig
	AgentUserGroup        *model.AgentUserGroup

	// These fields are set when the task is allocated.
	TaskID    string
	TaskToken string
	inner

	// These fields are set on a per container basis.
	ContainerID string
	Devices     []device.Device
}

// SetInner sets the concrete task represented by this spec.
func (t *TaskSpec) SetInner(inner InnerSpec) {
	t.inner = inner
}

// SetRuntimeInfo sets the runtime information.
func (t *TaskSpec) SetRuntimeInfo(taskID string, taskToken string) {
	t.TaskID = taskID
	t.TaskToken = taskToken
}

// SetContainerInfo sets the container information.
func (t *TaskSpec) SetContainerInfo(containerID string, devices []device.Device) {
	t.ContainerID = containerID
	t.Devices = devices
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

	return e
}

// Archives returns the archives that should be included in the container for this task.
func (t *TaskSpec) Archives() []container.RunArchive {
	return append(t.baseArchives(), t.inner.Archives(t.AgentUserGroup)...)
}

// Environment returns the container environment for this task.
func (t *TaskSpec) Environment() expconf.EnvironmentConfig { return t.inner.Environment(*t) }

// EnvVars returns the environment variables that should be set in the container for this task.
func (t *TaskSpec) EnvVars() map[string]string {
	e := t.baseEnvVars()
	for k, v := range t.inner.EnvVars(*t) {
		e[k] = v
	}
	return e
}

// TaskSpecMaker is used to make task specs.
type TaskSpecMaker struct {
	BaseTaskSpec  TaskSpec
	PoolsDefaults map[string]model.TaskContainerDefaultsConfig
}

// MakeTaskSpec makes a task spec.
func (t *TaskSpecMaker) MakeTaskSpec(
	poolName string, agentUserGroup *model.AgentUserGroup,
) TaskSpec {
	// Always fall back to the top-level TaskContainerDefaults
	taskContainerDefaults := t.BaseTaskSpec.TaskContainerDefaults

	if t.PoolsDefaults != nil && poolName != "" {
		if d, ok := t.PoolsDefaults[poolName]; ok {
			taskContainerDefaults = d
		}
	}

	// Not a deep copy, but deep enough not to overwrite the master's TaskContainerDefaults.
	taskSpec := t.BaseTaskSpec
	taskSpec.TaskContainerDefaults = taskContainerDefaults

	if agentUserGroup != nil {
		taskSpec.AgentUserGroup = agentUserGroup
	}

	return taskSpec
}
