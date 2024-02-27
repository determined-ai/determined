package model

import (
	"github.com/determined-ai/determined/master/pkg/check"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
)

// DefaultConfigGenericTaskConfig merges task containter defaults into a default generic task config struct.
func DefaultConfigGenericTaskConfig(
	taskContainerDefaults *TaskContainerDefaultsConfig,
) GenericTaskConfig {
	out := GenericTaskConfig{
		Resources: expconf.ResourcesConfig{
			RawSlots:           ptrs.Ptr(1),
			RawIsSingleNode:    ptrs.Ptr(true),
			RawMaxSlots:        nil,
			RawSlotsPerTrial:   nil,
			RawWeight:          nil,
			RawNativeParallel:  nil,
			RawShmSize:         nil,
			RawResourceManager: nil,
			RawResourcePool:    nil,
			RawPriority:        nil,
			RawDevices:         expconf.DevicesConfigV0{},
		},
		Environment: DefaultEnvConfig(taskContainerDefaults),
	}

	if taskContainerDefaults != nil {
		out.WorkDir = taskContainerDefaults.WorkDir
		out.BindMounts = taskContainerDefaults.BindMounts
		out.Pbs = taskContainerDefaults.Pbs
		out.Slurm = taskContainerDefaults.Slurm
	}

	return out
}

// GenericTaskConfig like expconf or command config but for generic tasks.
type GenericTaskConfig struct {
	BindMounts  BindMountsConfig        `json:"bind_mounts"`
	Environment Environment             `json:"environment"`
	Resources   expconf.ResourcesConfig `json:"resources"`
	Entrypoint  []string                `json:"entrypoint"`
	WorkDir     *string                 `json:"work_dir"`
	Debug       bool                    `json:"debug"`

	Pbs   expconf.PbsConfig   `json:"pbs,omitempty"`
	Slurm expconf.SlurmConfig `json:"slurm,omitempty"`
}

// Validate implements the check.Validatable interface.
func (c *GenericTaskConfig) Validate() []error {
	return []error{
		check.GreaterThanOrEqualTo(c.Resources.Slots(), 0,
			"resources.slots must be >= 0"),
		check.GreaterThan(len(c.Entrypoint), 0, "entrypoint must be non-empty"),
	}
}
