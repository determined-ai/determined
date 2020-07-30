package model

import (
	v1 "k8s.io/api/core/v1"

	"github.com/determined-ai/determined/master/pkg/check"
)

// CommandConfig holds the necessary configurations to launch a command task in
// the cluster.
type CommandConfig struct {
	Description string          `json:"description"`
	BindMounts  []BindMount     `json:"bind_mounts"`
	Environment Environment     `json:"environment"`
	Resources   ResourcesConfig `json:"resources"`
	Entrypoint  []string        `json:"entrypoint"`
	PodSpec     *v1.Pod         `json:"pod_spec"`
}

// Validate implements the check.Validatable interface.
func (c *CommandConfig) Validate() []error {
	return []error{
		check.GreaterThanOrEqualTo(c.Resources.Slots, 0, "resources.slots must be >= 0"),
		check.GreaterThan(len(c.Entrypoint), 0, "entrypoint must be non-empty"),
	}
}
