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

const (
	// DeterminedK8ContainerName is the name of the container that executes the task within Kubernetes
	// pods that are launched by Determined.
	DeterminedK8ContainerName = "determined-container"
	// DeterminedK8FluentContainerName is the name of the container running Fluent Bit in each pod.
	DeterminedK8FluentContainerName = "determined-fluent-container"
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
func (r RuntimeItem) For(deviceType device.Type) string {
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
	return validatePodSpec(e.PodSpec)
}

func validatePodSpec(podSpec *k8sV1.Pod) []error {
	if podSpec != nil {
		podSpecErrors := []error{
			check.Equal(podSpec.Name, "", "pod Name is not a configurable option"),
			check.Equal(podSpec.Namespace, "", "pod Namespace is not a configurable option"),
			check.False(podSpec.Spec.HostNetwork, "host networking must be configured via master.yaml"),
		}

		if len(podSpec.Spec.Containers) > 0 {
			for _, container := range podSpec.Spec.Containers {
				if container.Name != DeterminedK8ContainerName {
					continue
				}

				containerSpecErrors := []error{
					check.Equal(container.Image, "",
						"container Image is not configurable, set it in the experiment config"),
					check.Equal(len(container.Command), 0,
						"container Command is not configurable"),
					check.Equal(len(container.Args), 0,
						"container Args are not configurable"),
					check.Equal(container.WorkingDir, "",
						"container WorkingDir is not configurable"),
					check.Equal(len(container.Ports), 0,
						"container Ports are not configurable"),
					check.Equal(len(container.EnvFrom), 0,
						"container EnvFrom is not configurable"),
					check.Equal(len(container.Env), 0,
						"container Env is not configurable, set it in the experiment config"),
					check.True(container.LivenessProbe == nil,
						"container LivenessProbe is not configurable"),
					check.True(container.ReadinessProbe == nil,
						"container ReadinessProbe is not configurable"),
					check.True(container.StartupProbe == nil,
						"container StartupProbe is not configurable"),
					check.True(container.Lifecycle == nil,
						"container Lifecycle is not configurable"),
					check.Match(container.TerminationMessagePath, "",
						"container TerminationMessagePath is not configurable"),
					check.Match(string(container.TerminationMessagePolicy), "",
						"container TerminationMessagePolicy is not configurable"),
					check.Match(string(container.ImagePullPolicy), "",
						"container ImagePullPolicy is not configurable, set it in the experiment config"),
					check.True(container.SecurityContext == nil,
						"container SecurityContext is not configurable, set it in the experiment config"),
				}
				podSpecErrors = append(podSpecErrors, containerSpecErrors...)
			}
		}

		return podSpecErrors
	}
	return nil
}
