package tasks

import (
	"crypto/tls"
	"encoding/json"
	"github.com/determined-ai/determined/master/pkg/workload"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/archive"
	"github.com/determined-ai/determined/master/pkg/device"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/union"
)

// TaskSpec provides the necessary information for an agent to start a task.
type TaskSpec struct {
	TaskID                string                            `json:"task_id"`
	ContainerID           string                            `json:"container_id"`
	ClusterID             string                            `json:"cluster_id"`
	Devices               []device.Device                   `json:"devices"`
	HarnessPath           string                            `json:"harness_path"`
	TaskContainerDefaults model.TaskContainerDefaultsConfig `json:"task_container_defaults"`
	MasterCert            *tls.Certificate                  `json:"master_cert"`

	StartCommand   *StartCommand   `union:"type,START_TASK" json:"-"`
	StartContainer *StartContainer `union:"type,START_CONTAINER" json:"-"`
	GCCheckpoints  *GCCheckpoints  `union:"type,GC_CHECKPOINTS" json:"-"`
	KillContainer  *KillContainer  `union:"type,KILL_CONTAINER" json:"-"`
	RunWorkload    *RunWorkload    `union:"type,RUN_WORKLOAD" json:"-"`
}

// MarshalJSON serializes a TaskSpec.
func (t TaskSpec) MarshalJSON() ([]byte, error) {
	return union.Marshal(t)
}

// UnmarshalJSON deserializes a TaskSpec.
func (t *TaskSpec) UnmarshalJSON(data []byte) error {
	if err := union.Unmarshal(data, t); err != nil {
		return err
	}

	type DefaultParser *TaskSpec
	return errors.Wrap(json.Unmarshal(data, DefaultParser(t)), "failed to parse task specification")
}

// StartCommand is the information sent to an agent to start a command.
type StartCommand struct {
	// AgentUserGroup is the user and group to run this task as.
	AgentUserGroup *model.AgentUserGroup `json:"agent_user_group,omitempty"`

	Config          model.CommandConfig `json:"config"`
	UserFiles       archive.Archive     `json:"user_files"`
	AdditionalFiles archive.Archive     `json:"additional_files"`
}

// GCCheckpoints is the information sent to an agent to garbage collect a checkpoint.
type GCCheckpoints struct {
	// AgentUserGroup is the user and group to run this task as.
	AgentUserGroup *model.AgentUserGroup `json:"agent_user_group,omitempty"`

	ExperimentID     int                    `json:"experiment_id"`
	ExperimentConfig model.ExperimentConfig `json:"experiment_config"`
	ToDelete         json.RawMessage        `json:"to_delete"`
}

// StartContainer is the information sent to an agent to start a container (trial).
type StartContainer struct {
	// AgentUserGroup is the user and group to run this task as.
	AgentUserGroup *model.AgentUserGroup `json:"agent_user_group,omitempty"`

	ExperimentConfig    model.ExperimentConfig    `json:"experiment_config"`
	ModelDefinition     archive.Archive           `json:"model_definition"`
	HParams             map[string]interface{}    `json:"hparams"`
	TrialSeed           uint32                    `json:"trial_seed"`
	LatestCheckpoint    *model.Checkpoint         `json:"latest_checkpoint"`
	InitialWorkload     workload.Workload         `json:"initial_workload"`
	WorkloadManagerType model.WorkloadManagerType `json:"workload_manager_type"`
	AdditionalFiles     archive.Archive           `json:"additional_files"`

	// This is used to hint the resource provider to override defaults and start
	// the container in host mode iff it has been scheduled across multiple agents.
	IsMultiAgent bool `json:"is_multi_agent"`
}

// KillContainer is the information sent to an agent to kill a task (i.e., container or
// command).
type KillContainer struct{}

// RunWorkload is the information sent to an agent to run a workload.
type RunWorkload struct {
	Workload workload.Workload `json:"workload"`
}
