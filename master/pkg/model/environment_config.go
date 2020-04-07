package model

import (
	"encoding/json"
	"fmt"

	"github.com/docker/docker/api/types"
	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/device"
)

// Environment configures the environment of a Determined command or experiment.
type Environment struct {
	Image                RuntimeItem  `json:"image"`
	EnvironmentVariables RuntimeItems `json:"environment_variables,omitempty"`

	Ports          map[string]int    `json:"ports"`
	RegistryAuth   *types.AuthConfig `json:"registry_auth,omitempty"`
	ForcePullImage bool              `json:"force_pull_image"`
}

// RuntimeItem configures the runtime image.
type RuntimeItem struct {
	CPU string `json:"cpu,omitempty"`
	GPU string `json:"gpu,omitempty"`
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (r *RuntimeItem) UnmarshalJSON(data []byte) error {
	var plain string
	if err := json.Unmarshal(data, &plain); err == nil {
		r.CPU = plain
		r.GPU = plain
		return nil
	}
	type DefaultParser RuntimeItem
	var jsonItem DefaultParser
	if err := json.Unmarshal(data, &jsonItem); err != nil {
		return errors.Wrapf(err, "failed to parse runtime item")
	}
	r.CPU = jsonItem.CPU
	r.GPU = jsonItem.GPU
	return nil
}

// For returns the value for the provided device type.
func (r *RuntimeItem) For(deviceType device.Type) string {
	switch deviceType {
	case device.CPU:
		return r.CPU
	case device.GPU:
		return r.GPU
	default:
		panic(fmt.Sprintf("unexpected device type: %s", deviceType))
	}
}

// RuntimeItems configures the runtime environment variables.
type RuntimeItems struct {
	CPU []string `json:"cpu,omitempty"`
	GPU []string `json:"gpu,omitempty"`
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (r *RuntimeItems) UnmarshalJSON(data []byte) error {
	var plain []string
	if err := json.Unmarshal(data, &plain); err == nil {
		r.CPU = append(r.CPU, plain...)
		r.GPU = append(r.GPU, plain...)
		return nil
	}
	type DefaultParser RuntimeItems
	var jsonItems DefaultParser
	if err := json.Unmarshal(data, &jsonItems); err != nil {
		return errors.Wrapf(err, "failed to parse runtime items")
	}
	r.CPU = append(r.CPU, jsonItems.CPU...)
	r.GPU = append(r.GPU, jsonItems.GPU...)
	return nil
}

// For returns the value for the provided device type.
func (r *RuntimeItems) For(deviceType device.Type) []string {
	switch deviceType {
	case device.CPU:
		return r.CPU
	case device.GPU:
		return r.GPU
	default:
		panic(fmt.Sprintf("unexpected device type: %s", deviceType))
	}
}
