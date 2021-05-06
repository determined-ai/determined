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
	RawImage                *EnvironmentImageMapV0     `json:"image"`
	RawEnvironmentVariables *EnvironmentVariablesMapV0 `json:"environment_variables"`

	RawPorts          map[string]int    `json:"ports"`
	RawRegistryAuth   *types.AuthConfig `json:"registry_auth"`
	RawForcePullImage *bool             `json:"force_pull_image"`
	RawPodSpec        *k8sV1.Pod        `json:"pod_spec"`

	RawAddCapabilities  []string `json:"add_capabilities"`
	RawDropCapabilities []string `json:"drop_capabilities"`
}

//go:generate ../gen.sh
// EnvironmentImageMapV0 configures the runtime image.
type EnvironmentImageMapV0 struct {
	RawCPU *string `json:"cpu"`
	RawGPU *string `json:"gpu"`
}

// RuntimeDefaults implements the RuntimeDefaultable interface.
func (e EnvironmentImageMapV0) RuntimeDefaults() interface{} {
	if e.RawCPU == nil {
		cpu := DefaultCPUImage
		e.RawCPU = &cpu
	}
	if e.RawGPU == nil {
		gpu := DefaultGPUImage
		e.RawGPU = &gpu
	}
	return e
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (e *EnvironmentImageMapV0) UnmarshalJSON(data []byte) error {
	var plain string
	if err := json.Unmarshal(data, &plain); err == nil {
		e.RawCPU = &plain
		e.RawGPU = &plain
		return nil
	}
	type DefaultParser EnvironmentImageMapV0
	var jsonItem DefaultParser
	if err := json.Unmarshal(data, &jsonItem); err != nil {
		return errors.Wrapf(err, "failed to parse runtime item")
	}
	e.RawCPU = jsonItem.RawCPU
	e.RawGPU = jsonItem.RawGPU
	return nil
}

// For returns the value for the provided device type.
func (e EnvironmentImageMapV0) For(deviceType device.Type) string {
	switch deviceType {
	case device.CPU:
		return *e.RawCPU
	case device.GPU:
		return *e.RawGPU
	default:
		panic(fmt.Sprintf("unexpected device type: %s", deviceType))
	}
}

//go:generate ../gen.sh
// EnvironmentVariablesMapV0 configures the runtime environment variables.
type EnvironmentVariablesMapV0 struct {
	RawCPU []string `json:"cpu"`
	RawGPU []string `json:"gpu"`
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (e *EnvironmentVariablesMapV0) UnmarshalJSON(data []byte) error {
	var plain []string
	if err := json.Unmarshal(data, &plain); err == nil {
		e.RawCPU = []string{}
		e.RawGPU = []string{}
		e.RawCPU = append(e.RawCPU, plain...)
		e.RawGPU = append(e.RawGPU, plain...)
		return nil
	}
	type DefaultParser EnvironmentVariablesMapV0
	var jsonItems DefaultParser
	if err := json.Unmarshal(data, &jsonItems); err != nil {
		return errors.Wrapf(err, "failed to parse runtime items")
	}
	e.RawCPU = []string{}
	e.RawGPU = []string{}
	if jsonItems.RawCPU != nil {
		e.RawCPU = append(e.RawCPU, jsonItems.RawCPU...)
	}
	if jsonItems.RawGPU != nil {
		e.RawGPU = append(e.RawGPU, jsonItems.RawGPU...)
	}
	return nil
}

// For returns the value for the provided device type.
func (e EnvironmentVariablesMapV0) For(deviceType device.Type) []string {
	switch deviceType {
	case device.CPU:
		return e.RawCPU
	case device.GPU:
		return e.RawGPU
	default:
		panic(fmt.Sprintf("unexpected device type: %s", deviceType))
	}
}
