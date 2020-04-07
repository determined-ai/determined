package model

import (
	"github.com/docker/docker/api/types/container"

	"github.com/determined-ai/determined/master/pkg/check"
)

// ContainerDefaultsConfig configures docker defaults for all containers.
type ContainerDefaultsConfig struct {
	ShmSizeBytes int64                 `json:"shm_size_bytes,omitempty"`
	NetworkMode  container.NetworkMode `json:"network_mode,omitempty"`
}

// Validate implements the check.Validatable interface.
func (c ContainerDefaultsConfig) Validate() []error {
	return []error{
		check.GreaterThan(c.ShmSizeBytes, int64(0), "shm_size_bytes must be >= 0"),
		check.NotEmpty(string(c.NetworkMode), "network_mode must be set"),
	}
}
