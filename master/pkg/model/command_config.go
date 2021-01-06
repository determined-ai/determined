package model

import (
	"github.com/determined-ai/determined/master/pkg/check"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
)

// CommandConfig holds the necessary configurations to launch a command task in
// the cluster.
type CommandConfig struct {
	Description     string                    `json:"description"`
	BindMounts      expconf.BindMountsConfig  `json:"bind_mounts"`
	Environment     expconf.EnvironmentConfig `json:"environment"`
	Resources       ResourcesConfig           `json:"resources"`
	Entrypoint      []string                  `json:"entrypoint"`
	TensorBoardArgs []string                  `json:"tensorboard_args"`
}

// Validate implements the check.Validatable interface.
func (c *CommandConfig) Validate() []error {
	return []error{
		check.GreaterThanOrEqualTo(c.Resources.Slots, 0, "resources.slots must be >= 0"),
		check.GreaterThan(len(c.Entrypoint), 0, "entrypoint must be non-empty"),
	}
}

// ResourcesConfig configures commands resource usage.  It uses Slots instead of SlotsPerTrial.
type ResourcesConfig struct {
	Slots          int     `json:"slots,omitempty"`
	MaxSlots       *int    `json:"max_slots,omitempty"`
	Weight         float64 `json:"weight"`
	NativeParallel bool    `json:"native_parallel"`
	ShmSize        *int    `json:"shm_size,omitempty"`
	AgentLabel     string  `json:"agent_label"`
	ResourcePool   string  `json:"resource_pool"`
	Priority       *int    `json:"priority,omitempty"`
}
