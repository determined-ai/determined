package expconf

import (
	"encoding/json"
	"fmt"

	k8sV1 "k8s.io/api/core/v1"

	"github.com/docker/docker/api/types"
	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/device"
)

// EnvironmentConfigV0 configures the environment of a Determined command or experiment.
type EnvironmentConfigV0 struct {
	Image                *RuntimeItem               `json:"image"`
	EnvironmentVariables *EnvironmentVariablesMapV0 `json:"environment_variables"`

	Ports          *map[string]int   `json:"ports"`
	RegistryAuth   *types.AuthConfig `json:"registry_auth"`
	ForcePullImage *bool             `json:"force_pull_image"`
	PodSpec        *k8sV1.Pod        `json:"pod_spec"`
}

// RuntimeItem configures the runtime image.
type RuntimeItem struct {
	CPU *string `json:"cpu"`
	GPU *string `json:"gpu"`
}

// RuntimeDefaults implements the RuntimeDefautlable interface.
func (r *RuntimeItem) RuntimeDefaults() {
	if r.CPU == nil {
		s := DefaultCPUImage
		r.CPU = &s
	}
	if r.GPU == nil {
		s := DefaultGPUImage
		r.GPU = &s
	}
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (r *RuntimeItem) UnmarshalJSON(data []byte) error {
	var plain string
	if err := json.Unmarshal(data, &plain); err == nil {
		r.CPU = &plain
		r.GPU = &plain
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
		return *r.CPU
	case device.GPU:
		return *r.GPU
	default:
		panic(fmt.Sprintf("unexpected device type: %s", deviceType))
	}
}

// EnvironmentVariablesMapV0 configures the runtime environment variables.
type EnvironmentVariablesMapV0 struct {
	CPU *[]string `json:"cpu"`
	GPU *[]string `json:"gpu"`
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (r *EnvironmentVariablesMapV0) UnmarshalJSON(data []byte) error {
	var plain []string
	if err := json.Unmarshal(data, &plain); err == nil {
		r.CPU = &[]string{}
		r.GPU = &[]string{}
		*r.CPU = append(*r.CPU, plain...)
		*r.GPU = append(*r.GPU, plain...)
		return nil
	}
	type DefaultParser EnvironmentVariablesMapV0
	var jsonItems DefaultParser
	if err := json.Unmarshal(data, &jsonItems); err != nil {
		return errors.Wrapf(err, "failed to parse runtime items")
	}
	r.CPU = &[]string{}
	r.GPU = &[]string{}
	if jsonItems.CPU != nil {
		*r.CPU = append(*r.CPU, *jsonItems.CPU...)
	}
	if jsonItems.GPU != nil {
		*r.GPU = append(*r.GPU, *jsonItems.GPU...)
	}
	return nil
}

// For returns the value for the provided device type.
func (r *EnvironmentVariablesMapV0) For(deviceType device.Type) []string {
	switch deviceType {
	case device.CPU:
		return *r.CPU
	case device.GPU:
		return *r.GPU
	default:
		panic(fmt.Sprintf("unexpected device type: %s", deviceType))
	}
}

// XXX: what to do about these dummy types?

// EnvironmentImageMapV0 is a dummy type.
type EnvironmentImageMapV0 struct{}

// EnvironmentImageV0 is a dummy type.
type EnvironmentImageV0 struct{}

// EnvironmentVariablesV0 is a dummy type.
type EnvironmentVariablesV0 struct{}
