package model

import (
	"encoding/json"
	"fmt"

	k8sV1 "k8s.io/api/core/v1"

	"github.com/determined-ai/determined/master/pkg/check"

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
	PodSpec        *k8sV1.Pod        `json:"pod_spec"`
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

// Validate implements the check.Validatable interface.
func (e Environment) Validate() []error {
	if e.PodSpec != nil {
		podSpecErrors := []error{
			check.Equal(e.PodSpec.Name, "", "pod Name is not a configurable option"),
			check.Equal(e.PodSpec.Namespace, "", "pod Namespace is not a configurable option"),
			check.False(e.PodSpec.Spec.HostNetwork, "host networking must be configured via master.yaml"),
			check.Equal(
				len(e.PodSpec.Spec.InitContainers), 0,
				"init containers are not a configurable option"),
			check.LessThanOrEqualTo(
				len(e.PodSpec.Spec.Containers), 1,
				"can specify at most one container in pod_spec"),
		}

		if len(e.PodSpec.Spec.Containers) > 0 {
			container := e.PodSpec.Spec.Containers[0]
			containerSpecErrors := []error{
				check.Equal(container.Name, "", "container Name is not configurable"),
				check.Equal(container.Image, "",
					"container Image is not configurable, set it in the experiment config"),
				check.Equal(container.Command, nil, "container Command is not configurable"),
				check.Equal(container.Args, nil, "container Args are not configurable"),
				check.Equal(container.WorkingDir, "", "container WorkingDir is not configurable"),
				check.Equal(container.Ports, nil, "container Ports are not configurable"),
				check.Equal(container.EnvFrom, nil, "container EnvFrom is not configurable"),
				check.Equal(container.Env, nil,
					"container Env is not configurable, set it in the experiment config"),
				check.Equal(container.LivenessProbe, nil,
					"container LivenessProbe is not configurable"),
				check.Equal(container.ReadinessProbe, nil,
					"container ReadinessProbe is not configurable"),
				check.Equal(container.StartupProbe, nil,
					"container StartupProbe is not configurable"),
				check.Equal(container.Lifecycle, nil, "container Lifecycle is not configurable"),
				check.Equal(container.TerminationMessagePath, "",
					"container TerminationMessagePath is not configurable"),
				check.Equal(container.TerminationMessagePolicy, "",
					"container TerminationMessagePolicy is not configurable"),
				check.Equal(container.ImagePullPolicy, "",
					"container ImagePullPolicy is not configurable, set it in the experiment config"),
				check.Equal(container.SecurityContext, nil,
					"container SecurityContext is not configurable, set it in the experiment config"),
			}
			podSpecErrors = append(podSpecErrors, containerSpecErrors...)
		}

		return podSpecErrors
	}
	return nil
}
