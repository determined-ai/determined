package model

import (
	"github.com/determined-ai/determined/master/pkg/check"
)

// DefaultConfig is the default configuration used by all
// commands (e.g., commands, notebooks, shells) if a request
// does not specify any configuration options.
func DefaultConfig(taskContainerDefaults *TaskContainerDefaultsConfig) CommandConfig {
	out := CommandConfig{
		Resources:   DefaultResourcesConfig(taskContainerDefaults),
		Environment: DefaultEnvConfig(taskContainerDefaults),
	}

	if taskContainerDefaults != nil {
		out.WorkDir = taskContainerDefaults.WorkDir
		out.BindMounts = taskContainerDefaults.BindMounts
	}

	return out
}

// CommandConfig holds the necessary configurations to launch a command task in
// the cluster.
type CommandConfig struct {
	Description     string           `json:"description"`
	BindMounts      BindMountsConfig `json:"bind_mounts"`
	Environment     Environment      `json:"environment"`
	Resources       ResourcesConfig  `json:"resources"`
	Entrypoint      []string         `json:"entrypoint"`
	TensorBoardArgs []string         `json:"tensorboard_args,omitempty"`
	IdleTimeout     *Duration        `json:"idle_timeout"`
	WorkDir         *string          `json:"work_dir"`
}

// Validate implements the check.Validatable interface.
func (c *CommandConfig) Validate() []error {
	return []error{
		check.GreaterThanOrEqualTo(c.Resources.Slots, 0, "resources.slots must be >= 0"),
		check.GreaterThan(len(c.Entrypoint), 0, "entrypoint must be non-empty"),
	}
}
