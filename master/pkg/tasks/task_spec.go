package tasks

import (
	"crypto/tls"
	"encoding/json"

	"github.com/determined-ai/determined/master/pkg/workload"

	"github.com/determined-ai/determined/master/pkg/archive"
	"github.com/determined-ai/determined/master/pkg/device"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
)

// TaskSpec provides the necessary information for an agent to start a task.
type TaskSpec struct {
	TaskID      string
	ContainerID string
	Devices     []device.Device

	ClusterID             string
	HarnessPath           string
	TaskContainerDefaults model.TaskContainerDefaultsConfig
	MasterCert            *tls.Certificate

	StartCommand   *StartCommand
	StartContainer *StartContainer
	GCCheckpoints  *GCCheckpoints
}

// StartCommand is the information sent to an agent to start a command.
type StartCommand struct {
	// AgentUserGroup is the user and group to run this task as.
	AgentUserGroup *model.AgentUserGroup

	Config          model.CommandConfig
	UserFiles       archive.Archive
	AdditionalFiles archive.Archive
}

// GCCheckpoints is the information sent to an agent to garbage collect a checkpoint.
type GCCheckpoints struct {
	// AgentUserGroup is the user and group to run this task as.
	AgentUserGroup *model.AgentUserGroup

	ExperimentID     int
	ExperimentConfig expconf.ExperimentConfig
	ToDelete         json.RawMessage
}

// StartContainer is the information sent to an agent to start a container (trial).
type StartContainer struct {
	// AgentUserGroup is the user and group to run this task as.
	AgentUserGroup *model.AgentUserGroup

	ExperimentConfig    expconf.ExperimentConfig
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
