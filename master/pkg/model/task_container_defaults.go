package model

import (
	"encoding/json"
	"regexp"
	"strconv"

	"github.com/docker/docker/api/types"

	k8sV1 "k8s.io/api/core/v1"

	"github.com/docker/docker/api/types/container"
	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/check"
	"github.com/determined-ai/determined/master/pkg/ptrs"
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
	Image                  *RuntimeItem          `json:"image,omitempty"`
	RegistryAuth           *types.AuthConfig     `json:"registry_auth,omitempty"`
	ForcePullImage         bool                  `json:"force_pull_image,omitempty"`

	AddCapabilities  []string      `json:"add_capabilities"`
	DropCapabilities []string      `json:"drop_capabilities"`
	Devices          DevicesConfig `json:"devices"`
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

// UnmarshalJSON implements the json.Unmarshaler interface.
// Setting defaults here is necessary over our usual "Define a default struct and unmarshal into it"
// strategy because there are places (resource pool configs) where we need to know if the task
// container defaults were set at all or if they were not; if they were set then that resource
// pool's task container defaults are used instead of the toplevel master config's settings.  To
// know if the user set them at the resource pool level, the resource pool has to have a nullable
// pointer, which is not compatible with our usual strategy for defaults.
func (c *TaskContainerDefaultsConfig) UnmarshalJSON(data []byte) error {
	c.ShmSizeBytes = 4294967296
	c.NetworkMode = "bridge"
	type DefaultParser *TaskContainerDefaultsConfig
	if err := json.Unmarshal(data, DefaultParser(c)); err != nil {
		return errors.Wrap(err, "failed to parse task container defaults")
	}
	return nil
}

// Validate implements the check.Validatable interface.
func (c *TaskContainerDefaultsConfig) Validate() []error {
	if c == nil {
		return nil
	}
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

// MergeIntoConfig sets any unset ExperimentConfig values that are defined by TaskContainerDefaults.
func (c *TaskContainerDefaultsConfig) MergeIntoConfig(config *expconf.ExperimentConfig) {
	if c == nil {
		return
	}

	// Merge Resources-related settings into the config.
	resources := expconf.ResourcesConfig{
		RawDevices: c.Devices.ToExpconf(),
	}
	config.RawResources = schemas.Merge(config.RawResources, &resources).(*expconf.ResourcesConfig)

	// Merge Environment-related settings into the config.
	var image *expconf.EnvironmentImageMapV0
	if c.Image != nil {
		i := c.Image.ToExpconf()
		image = &i
	}

	// We just update config.RawResources so we know it can't be nil.
	defaultedResources := schemas.WithDefaults(*config.RawResources).(expconf.ResourcesConfig)
	podSpec := c.CPUPodSpec
	if defaultedResources.SlotsPerTrial() > 0 {
		podSpec = c.GPUPodSpec
	}

	env := expconf.EnvironmentConfig{
		RawAddCapabilities:  c.AddCapabilities,
		RawDropCapabilities: c.DropCapabilities,
		RawForcePullImage:   ptrs.BoolPtr(c.ForcePullImage),
		RawImage:            image,
		RawPodSpec:          podSpec,
		RawRegistryAuth:     c.RegistryAuth,
	}
	config.RawEnvironment = schemas.Merge(config.RawEnvironment, &env).(*expconf.EnvironmentConfig)
}
