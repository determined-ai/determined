package expconf

import (
	"encoding/json"
	"fmt"

	k8sV1 "k8s.io/api/core/v1"

	"github.com/docker/docker/api/types"
	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/device"
)

//go:generate ../gen.sh --import github.com/docker/docker/api/types,k8sV1:k8s.io/api/core/v1
// EnvironmentConfigV0 configures the environment of a Determined command or experiment.
type EnvironmentConfigV0 struct {
	Image                *EnvironmentImageMapV0     `json:"image"`
	EnvironmentVariables *EnvironmentVariablesMapV0 `json:"environment_variables"`

	Ports          *map[string]int   `json:"ports"`
	RegistryAuth   *types.AuthConfig `json:"registry_auth"`
	ForcePullImage *bool             `json:"force_pull_image"`
	PodSpec        *k8sV1.Pod        `json:"pod_spec"`

	AddCapabilities  *[]string `json:"add_capabilities"`
	DropCapabilities *[]string `json:"drop_capabilities"`
}

//go:generate ../gen.sh
// EnvironmentImageMapV0 configures the runtime image.
type EnvironmentImageMapV0 struct {
	CPU *string `json:"cpu"`
	GPU *string `json:"gpu"`
}

// RuntimeDefaults implements the RuntimeDefaultable interface.
func (e EnvironmentImageMapV0) RuntimeDefaults() interface{} {
	if e.CPU == nil {
		cpu := DefaultCPUImage
		e.CPU = &cpu
	}
	if e.GPU == nil {
		gpu := DefaultGPUImage
		e.GPU = &gpu
	}
	return e
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (e *EnvironmentImageMapV0) UnmarshalJSON(data []byte) error {
	var plain string
	if err := json.Unmarshal(data, &plain); err == nil {
		e.CPU = &plain
		e.GPU = &plain
		return nil
	}
	type DefaultParser EnvironmentImageMapV0
	var jsonItem DefaultParser
	if err := json.Unmarshal(data, &jsonItem); err != nil {
		return errors.Wrapf(err, "failed to parse runtime item")
	}
	e.CPU = jsonItem.CPU
	e.GPU = jsonItem.GPU
	return nil
}

// For returns the value for the provided device type.
func (e *EnvironmentImageMapV0) For(deviceType device.Type) string {
	switch deviceType {
	case device.CPU:
		return *e.CPU
	case device.GPU:
		return *e.GPU
	default:
		panic(fmt.Sprintf("unexpected device type: %s", deviceType))
	}
}

//go:generate ../gen.sh
// EnvironmentVariablesMapV0 configures the runtime environment variables.
type EnvironmentVariablesMapV0 struct {
	CPU *[]string `json:"cpu"`
	GPU *[]string `json:"gpu"`
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (e *EnvironmentVariablesMapV0) UnmarshalJSON(data []byte) error {
	var plain []string
	if err := json.Unmarshal(data, &plain); err == nil {
		e.CPU = &[]string{}
		e.GPU = &[]string{}
		*e.CPU = append(*e.CPU, plain...)
		*e.GPU = append(*e.GPU, plain...)
		return nil
	}
	type DefaultParser EnvironmentVariablesMapV0
	var jsonItems DefaultParser
	if err := json.Unmarshal(data, &jsonItems); err != nil {
		return errors.Wrapf(err, "failed to parse runtime items")
	}
	e.CPU = &[]string{}
	e.GPU = &[]string{}
	if jsonItems.CPU != nil {
		*e.CPU = append(*e.CPU, *jsonItems.CPU...)
	}
	if jsonItems.GPU != nil {
		*e.GPU = append(*e.GPU, *jsonItems.GPU...)
	}
	return nil
}

// For returns the value for the provided device type.
func (e *EnvironmentVariablesMapV0) For(deviceType device.Type) []string {
	switch deviceType {
	case device.CPU:
		return *e.CPU
	case device.GPU:
		return *e.GPU
	default:
		panic(fmt.Sprintf("unexpected device type: %s", deviceType))
	}
}
