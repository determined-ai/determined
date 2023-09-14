package model

import (
	"encoding/json"
	"fmt"

	k8sV1 "k8s.io/api/core/v1"

	"github.com/determined-ai/determined/master/pkg/check"
	"github.com/determined-ai/determined/proto/pkg/apiv1"

	"github.com/docker/docker/api/types"
	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/device"

	"github.com/ghodss/yaml"
	"google.golang.org/protobuf/encoding/protojson"
)

const (
	// DeterminedK8ContainerName is the name of the container that executes the task within Kubernetes
	// pods that are launched by Determined.
	DeterminedK8ContainerName = "determined-container"
)

// Environment configures the environment of a Determined command or experiment.
type Environment struct {
	Image                RuntimeItem      `json:"image"`
	EnvironmentVariables RuntimeItems     `json:"environment_variables,omitempty"`
	ProxyPorts           ProxyPortsConfig `json:"proxy_ports"`

	Ports          map[string]int    `json:"ports"`
	RegistryAuth   *types.AuthConfig `json:"registry_auth,omitempty"`
	ForcePullImage bool              `json:"force_pull_image"`
	PodSpec        *k8sV1.Pod        `json:"pod_spec"`

	AddCapabilities  []string `json:"add_capabilities"`
	DropCapabilities []string `json:"drop_capabilities"`
}

// RuntimeItem configures the runtime image.
type RuntimeItem struct {
	CPU  string `json:"cpu,omitempty"`
	CUDA string `json:"cuda,omitempty"`
	ROCM string `json:"rocm,omitempty"`
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (r *RuntimeItem) UnmarshalJSON(data []byte) error {
	var plain string
	if err := json.Unmarshal(data, &plain); err == nil {
		r.CPU = plain
		r.ROCM = plain
		r.CUDA = plain
		return nil
	}

	type DefaultParser RuntimeItem
	var jsonItem DefaultParser
	if err := json.Unmarshal(data, &jsonItem); err != nil {
		return errors.Wrapf(err, "failed to parse runtime item")
	}

	// A map to hold the decoded the JSON structure.
	imagesMap := make(map[string]json.RawMessage)

	// Decode the JSON data into our map.
	if err := json.Unmarshal(data, &imagesMap); err != nil {
		return errors.Wrapf(err, "failed to decode JSON runtime item into map")
	}

	// Overwrite the images only if the device type (i.e., "CPU", "CUDA",
	// "ROCM") is contained in the JSON. We don't want to unconditionally copy
	// the device type from the "jsonItem" because it will be an empty string
	// if the device type is not contained in the JSON, and we would overwrite
	// the original value with an empty string.
	if _, ok := imagesMap["cpu"]; ok {
		r.CPU = jsonItem.CPU
	}

	if _, ok := imagesMap["rocm"]; ok {
		r.ROCM = jsonItem.ROCM
	}

	if _, ok := imagesMap["cuda"]; ok {
		r.CUDA = jsonItem.CUDA
	}

	if r.CUDA == "" {
		type RuntimeItemCompat struct {
			GPU string `json:"gpu,omitempty"`
		}
		var compatItem RuntimeItemCompat
		if err := json.Unmarshal(data, &compatItem); err != nil {
			return errors.Wrapf(err, "failed to parse runtime item")
		}
		r.CUDA = compatItem.GPU
	}

	return nil
}

// For returns the value for the provided device type.
func (r RuntimeItem) For(deviceType device.Type) string {
	switch deviceType {
	case device.CPU:
		return r.CPU
	case device.CUDA:
		return r.CUDA
	case device.ROCM:
		return r.ROCM
	default:
		panic(fmt.Sprintf("unexpected device type: %s", deviceType))
	}
}

// RuntimeItems configures the runtime environment variables.
type RuntimeItems struct {
	CPU  []string `json:"cpu,omitempty"`
	CUDA []string `json:"cuda,omitempty"`
	ROCM []string `json:"rocm,omitempty"`
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (r *RuntimeItems) UnmarshalJSON(data []byte) error {
	var plain []string
	if err := json.Unmarshal(data, &plain); err == nil {
		r.CPU = append(r.CPU, plain...)
		r.ROCM = append(r.ROCM, plain...)
		r.CUDA = append(r.CUDA, plain...)
		return nil
	}

	type DefaultParser RuntimeItems
	var jsonItems DefaultParser
	if err := json.Unmarshal(data, &jsonItems); err != nil {
		return errors.Wrapf(err, "failed to parse runtime items")
	}
	r.CPU = append(r.CPU, jsonItems.CPU...)
	r.ROCM = append(r.ROCM, jsonItems.ROCM...)

	r.CUDA = append(r.CUDA, jsonItems.CUDA...)

	if len(r.CUDA) == 0 {
		type RuntimeItemsCompat struct {
			GPU []string `json:"gpu,omitempty"`
		}
		var compatItems RuntimeItemsCompat
		if err := json.Unmarshal(data, &compatItems); err != nil {
			return errors.Wrapf(err, "failed to parse runtime items")
		}
		r.CUDA = append(r.CUDA, compatItems.GPU...)
	}
	return nil
}

// For returns the value for the provided device type.
func (r *RuntimeItems) For(deviceType device.Type) []string {
	switch deviceType {
	case device.CPU:
		return r.CPU
	case device.CUDA:
		return r.CUDA
	case device.ROCM:
		return r.ROCM
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
				}
				podSpecErrors = append(podSpecErrors, containerSpecErrors...)
			}
		}

		return podSpecErrors
	}
	return nil
}

// UsingCustomImage checks for image argument in request.
// It's only used for tensor board now.
// Error is ignored because we treat unexpected error when parsing as not using custom image.
func UsingCustomImage(req *apiv1.LaunchTensorboardRequest) bool {
	if req.Config == nil {
		return false
	}

	configBytes, err := protojson.Marshal(req.Config)
	if err != nil {
		return false
	}

	type DummyEnv struct {
		Image *RuntimeItem `json:"image"`
	}
	type DummyConfig struct {
		Environment DummyEnv `json:"environment"`
	}

	dummy := DummyConfig{
		Environment: DummyEnv{},
	}

	err = yaml.Unmarshal(configBytes, &dummy)

	return err == nil && dummy.Environment.Image != nil
}
