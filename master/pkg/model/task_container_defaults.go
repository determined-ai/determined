package model

import (
	"regexp"
	"strconv"

	"github.com/docker/docker/api/types"

	k8sV1 "k8s.io/api/core/v1"

	"github.com/docker/docker/api/types/container"
	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/check"
	"github.com/determined-ai/determined/master/pkg/schemas"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
)

// TaskContainerDefaultsConfig configures docker defaults for all containers.
type TaskContainerDefaultsConfig struct {
	DtrainNetworkInterface string                `json:"dtrain_network_interface,omitempty"`
	NCCLPortRange          string                `json:"nccl_port_range,omitempty"`
	GLOOPortRange          string                `json:"gloo_port_range,omitempty"`
	ShmSizeBytes           int64                 `json:"shm_size_bytes,omitempty"`
	NetworkMode            container.NetworkMode `json:"network_mode,omitempty"`
	CPUPodSpec             *k8sV1.Pod            `json:"cpu_pod_spec"`
	GPUPodSpec             *k8sV1.Pod            `json:"gpu_pod_spec"`
	Image                  *expconf.RuntimeItem  `json:"image,omitempty"`
	RegistryAuth           *types.AuthConfig     `json:"registry_auth,omitempty"`
	ForcePullImage         bool                  `json:"force_pull_image,omitempty"`
}

func validatePortRange(portRange string) []error {
	var errs []error

	if portRange == "" {
		return errs
	}

	re := regexp.MustCompile("^([0-9]+):([0-9]+)$")
	submatches := re.FindStringSubmatch(portRange)
	if submatches == nil {
		errs = append(
			errs, errors.Errorf("expected port range of format \"MIN:MAX\" but got %q", portRange),
		)
		return errs
	}

	var min, max uint64
	var err error
	if min, err = strconv.ParseUint(submatches[1], 10, 16); err != nil {
		errs = append(errs, errors.Wrap(err, "invalid minimum port value"))
	}
	if max, err = strconv.ParseUint(submatches[2], 10, 16); err != nil {
		errs = append(errs, errors.Wrap(err, "invalid maximum port value"))
	}

	if min > max {
		errs = append(errs, errors.Errorf("port range minimum exceeds maximum (%v > %v)", min, max))
	}

	return errs
}

// Validate implements the check.Validatable interface.
func (c TaskContainerDefaultsConfig) Validate() []error {
	errs := []error{
		check.GreaterThan(c.ShmSizeBytes, int64(0), "shm_size_bytes must be >= 0"),
		check.NotEmpty(string(c.NetworkMode), "network_mode must be set"),
	}

	if err := validatePortRange(c.NCCLPortRange); err != nil {
		errs = append(errs, err...)
	}

	if err := validatePortRange(c.GLOOPortRange); err != nil {
		errs = append(errs, err...)
	}

	errs = append(errs, validatePodSpec(c.CPUPodSpec)...)
	errs = append(errs, validatePodSpec(c.GPUPodSpec)...)

	return errs
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

// Filler returns a config suitable for schemas.Merge().
func (c *TaskContainerDefaultsConfig) Filler() expconf.ExperimentConfig {
	var e expconf.ExperimentConfig
	if c == nil {
		return e
	}

	e.Environment = &expconf.EnvironmentConfig{}

	schemas.Merge(&e.Environment.RegistryAuth, c.RegistryAuth)
	schemas.Merge(&e.Environment.ForcePullImage, c.ForcePullImage)
	schemas.Merge(&e.Environment.ForcePullImage, c.ForcePullImage)
	schemas.Merge(&e.Environment.Image, c.Image)

	return e
}
