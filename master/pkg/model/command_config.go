package model

import (
	"github.com/determined-ai/determined/master/pkg/check"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
)

const (
	// NotebookIdleTypeKernelsOrTerminals indicates that a notebook should be considered active if any
	// kernels or terminals are open.
	NotebookIdleTypeKernelsOrTerminals = "kernels_or_terminals"
	// NotebookIdleTypeKernelConnections indicates that a notebook should be considered active if any
	// connections to kernels are open.
	NotebookIdleTypeKernelConnections = "kernel_connections"
	// NotebookIdleTypeActivity indicates that a notebook should be considered active if any kernel is
	// running a command or any terminal is inputting or outputting data.
	NotebookIdleTypeActivity = "activity"
)

// DefaultConfig is the default configuration used by all
// commands (e.g., commands, notebooks, shells) if a request
// does not specify any configuration options.
func DefaultConfig(taskContainerDefaults *TaskContainerDefaultsConfig) CommandConfig {
	out := CommandConfig{
		Resources:        DefaultResourcesConfig(taskContainerDefaults),
		Environment:      DefaultEnvConfig(taskContainerDefaults),
		NotebookIdleType: NotebookIdleTypeKernelsOrTerminals,
	}

	if taskContainerDefaults != nil {
		out.WorkDir = taskContainerDefaults.WorkDir
		out.BindMounts = taskContainerDefaults.BindMounts
		out.Pbs = taskContainerDefaults.Pbs
		out.Slurm = taskContainerDefaults.Slurm
	}

	return out
}

// CommandConfig holds the necessary configurations to launch a command task in
// the cluster.
type CommandConfig struct {
	Description      string              `json:"description"`
	BindMounts       BindMountsConfig    `json:"bind_mounts"`
	Environment      Environment         `json:"environment"`
	Resources        ResourcesConfig     `json:"resources"`
	Entrypoint       []string            `json:"entrypoint"`
	TensorBoardArgs  []string            `json:"tensorboard_args,omitempty"`
	IdleTimeout      *Duration           `json:"idle_timeout"`
	NotebookIdleType string              `json:"notebook_idle_type"`
	WorkDir          *string             `json:"work_dir"`
	Debug            bool                `json:"debug"`
	Pbs              expconf.PbsConfig   `json:"pbs,omitempty"`
	Slurm            expconf.SlurmConfig `json:"slurm,omitempty"`
}

// Validate implements the check.Validatable interface.
func (c *CommandConfig) Validate() []error {
	return []error{
		check.GreaterThanOrEqualTo(c.Resources.Slots, 0, "resources.slots must be >= 0"),
		check.GreaterThan(len(c.Entrypoint), 0, "entrypoint must be non-empty"),
		check.Contains(
			c.NotebookIdleType,
			[]interface{}{
				NotebookIdleTypeKernelsOrTerminals,
				NotebookIdleTypeKernelConnections,
				NotebookIdleTypeActivity,
			},
			"invalid notebook idle type",
		),
	}
}
