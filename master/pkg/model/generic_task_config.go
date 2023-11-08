package model

import (
	"github.com/determined-ai/determined/master/pkg/check"
)

func DefaultConfigGenericTaskConfig(taskContainerDefaults *TaskContainerDefaultsConfig) GenericTaskConfig {
	out := GenericTaskConfig{ // TODO
		// Resources:   DefaultResourcesConfig(taskContainerDefaults),
		Environment: DefaultEnvConfig(taskContainerDefaults),
		// NotebookIdleType: NotebookIdleTypeKernelsOrTerminals,
	}

	if taskContainerDefaults != nil {
		/*
			out.WorkDir = taskContainerDefaults.WorkDir
			out.BindMounts = taskContainerDefaults.BindMounts
			out.Pbs = taskContainerDefaults.Pbs
			out.Slurm = taskContainerDefaults.Slurm
		*/
	}

	return out
}

type GenericTaskConfig struct {
	BindMounts  BindMountsConfig    `json:"bind_mounts"`
	Environment Environment         `json:"environment"`
	Resources   TaskResourcesConfig `json:"resources"`
	Entrypoint  string              `json:"entrypoint"` // TODO should be string or []string?
	WorkDir     *string             `json:"work_dir"`
	Debug       bool                `json:"debug"`
	Description string              `json:"description"` // should be in run

	// Pbs         expconf.PbsConfig   `json:"pbs,omitempty"`
	// Slurm       expconf.SlurmConfig `json:"slurm,omitempty"`

	// 	TensorBoardArgs []string         `json:"tensorboard_args,omitempty"`
	// IdleTimeout      *Duration           `json:"idle_timeout"`
	// NotebookIdleType string              `json:"notebook_idle_type"`
}

// Validate implements the check.Validatable interface.
func (c *GenericTaskConfig) Validate() []error {
	return []error{
		check.GreaterThanOrEqualTo(c.Resources.SlotsPerTask, 0,
			"resources.slots_per_task must be >= 0"),
		check.GreaterThan(len(c.Entrypoint), 0, "entrypoint must be non-empty"),
	}
}

type TaskResourcesConfig struct {
	SlotsPerTask int    `json:"slots_per_task"`
	SingleNode   bool   `json:"single_node"`
	ResourcePool string `json:"resource_pool"`

	Devices DevicesConfig `json:"devices"`

	// MaxSlots       *int         `json:"max_slots,omitempty"`
	// Weight         float64      `json:"weight"`
	// NativeParallel bool         `json:"native_parallel,omitempty"`
	// ShmSize        *StorageSize `json:"shm_size,omitempty"`
	// Priority *int `json:"priority,omitempty"`
}
