package model

import (
	"regexp"
	"strconv"

	"github.com/docker/docker/api/types"

	k8sV1 "k8s.io/api/core/v1"

	"github.com/docker/docker/api/types/container"
	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/check"
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
	Image                  *RuntimeItem          `json:"image,omitempty"`
	RegistryAuth           *types.AuthConfig     `json:"registry_auth,omitempty"`
	ForcePullImage         bool                  `json:"force_pull_image,omitempty"`

	// omitempty since they are not officially announced features yet
	AddCapabilities  *[]string      `json:"add_capabilities,omitempty"`
	DropCapabilities *[]string      `json:"drop_capabilities,omitempty"`
	Devices          *DevicesConfig `json:"devices,omitempty"`
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
