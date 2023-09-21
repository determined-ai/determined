package model

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/jinzhu/copier"

	k8sV1 "k8s.io/api/core/v1"

	"github.com/docker/docker/api/types/container"
	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/check"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/master/pkg/set"
)

// TaskContainerDefaultsConfig configures docker defaults for all containers.
// If you add a field to this, you must update the merge impl.
type TaskContainerDefaultsConfig struct {
	DtrainNetworkInterface string                `json:"dtrain_network_interface,omitempty"`
	NCCLPortRange          string                `json:"nccl_port_range,omitempty"`
	GLOOPortRange          string                `json:"gloo_port_range,omitempty"`
	ShmSizeBytes           int64                 `json:"shm_size_bytes,omitempty"`
	NetworkMode            container.NetworkMode `json:"network_mode,omitempty"`
	// TODO(DET-9855) we should move these over to KubernetesTaskContainerDefaults.
	CPUPodSpec           *k8sV1.Pod        `json:"cpu_pod_spec"`
	GPUPodSpec           *k8sV1.Pod        `json:"gpu_pod_spec"`
	Image                *RuntimeItem      `json:"image,omitempty"`
	RegistryAuth         *types.AuthConfig `json:"registry_auth,omitempty"`
	ForcePullImage       bool              `json:"force_pull_image,omitempty"`
	EnvironmentVariables *RuntimeItems     `json:"environment_variables,omitempty"`

	AddCapabilities  []string      `json:"add_capabilities"`
	DropCapabilities []string      `json:"drop_capabilities"`
	Devices          DevicesConfig `json:"devices"`

	BindMounts BindMountsConfig      `json:"bind_mounts"`
	WorkDir    *string               `json:"work_dir"`
	Slurm      expconf.SlurmConfigV0 `json:"slurm"`
	Pbs        expconf.PbsConfigV0   `json:"pbs"`

	// TODO(DET-9856) we should probably eventually move this to expconf and allow setting
	// on a per task level.
	Kubernetes *KubernetesTaskContainerDefaults `json:"kubernetes"`
}

// DefaultTaskContainerDefaults returns the default for TaskContainerDefaultsConfig.
func DefaultTaskContainerDefaults() *TaskContainerDefaultsConfig {
	return &TaskContainerDefaultsConfig{
		ShmSizeBytes: 4294967296,
		NetworkMode:  "bridge",
	}
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

	errs = append(errs, validatePodSpec(c.CPUPodSpec)...)
	errs = append(errs, validatePodSpec(c.GPUPodSpec)...)

	return errs
}

// KubernetesTaskContainerDefaults is task container defaults specific to Kubernetes.
type KubernetesTaskContainerDefaults struct {
	MaxSlotsPerPod int `json:"max_slots_per_pod"`
}

// MergeIntoExpConfig sets any unset ExperimentConfig values from TaskContainerDefaults.
func (c *TaskContainerDefaultsConfig) MergeIntoExpConfig(config *expconf.ExperimentConfig) {
	if c == nil {
		return
	}

	// Merge Resources-related settings into the config.
	//nolint:exhaustivestruct // Devices are the only thing relevant from TaskContainerDefaults.
	resources := expconf.ResourcesConfig{
		RawDevices: c.Devices.ToExpconf(),
	}
	config.RawResources = schemas.Merge(config.RawResources, &resources)

	// Merge Environment-related settings into the config.
	var image *expconf.EnvironmentImageMapV0
	if c.Image != nil {
		i := c.Image.ToExpconf()
		image = &i
	}

	var envVars *expconf.EnvironmentVariablesMapV0
	if c.EnvironmentVariables != nil {
		envVars = ptrs.Ptr(c.EnvironmentVariables.ToExpconf())
	}

	// We just update config.RawResources so we know it can't be nil.
	defaultedResources := schemas.WithDefaults(*config.RawResources)
	podSpec := c.CPUPodSpec
	if defaultedResources.SlotsPerTrial() > 0 {
		podSpec = c.GPUPodSpec
	}

	//nolint:exhaustivestruct // RawPorts is not in TaskContainerDefaults.
	env := expconf.EnvironmentConfig{
		RawAddCapabilities:      c.AddCapabilities,
		RawDropCapabilities:     c.DropCapabilities,
		RawForcePullImage:       ptrs.Ptr(c.ForcePullImage),
		RawImage:                image,
		RawPodSpec:              (*expconf.PodSpec)(podSpec),
		RawRegistryAuth:         c.RegistryAuth,
		RawEnvironmentVariables: envVars,
	}
	config.RawEnvironment = schemas.Merge(config.RawEnvironment, &env)

	bindMounts := c.BindMounts.ToExpconf()
	config.RawBindMounts = schemas.Merge(config.RawBindMounts, bindMounts)

	configRawSlurmConfig := config.RawSlurmConfig
	config.RawSlurmConfig = schemas.Merge(config.RawSlurmConfig, &c.Slurm)
	if configRawSlurmConfig != nil {
		config.RawSlurmConfig.RawSbatchArgs = append(
			c.Slurm.SbatchArgs(), configRawSlurmConfig.SbatchArgs()...)
	}

	configRawPbsConfig := config.RawPbsConfig
	config.RawPbsConfig = schemas.Merge(config.RawPbsConfig, &c.Pbs)
	if configRawPbsConfig != nil {
		config.RawPbsConfig.RawSbatchArgs = append(
			c.Pbs.SbatchArgs(), configRawPbsConfig.SbatchArgs()...)
	}
}

var mergeCopier = copier.Option{IgnoreEmpty: true, DeepCopy: true}

// Merge merges other into self, preferring other. The result is a deepcopy of self, with deep
// copies of values taken from other.
func (c TaskContainerDefaultsConfig) Merge(
	other TaskContainerDefaultsConfig,
) (TaskContainerDefaultsConfig, error) {
	var res TaskContainerDefaultsConfig
	err := copier.CopyWithOption(&res, c, mergeCopier)
	if err != nil {
		return TaskContainerDefaultsConfig{}, fmt.Errorf("cloning task container defaults: %w", err)
	}

	if other.DtrainNetworkInterface != "" {
		res.DtrainNetworkInterface = other.DtrainNetworkInterface
	}

	if other.NCCLPortRange != "" {
		res.NCCLPortRange = other.NCCLPortRange
	}

	if other.GLOOPortRange != "" {
		res.GLOOPortRange = other.GLOOPortRange
	}

	if other.ShmSizeBytes != 0 {
		res.ShmSizeBytes = other.ShmSizeBytes
	}

	if other.NetworkMode != "" {
		res.NetworkMode = other.NetworkMode
	}

	if other.CPUPodSpec != nil {
		res.CPUPodSpec = other.CPUPodSpec.DeepCopy()
	}

	if other.GPUPodSpec != nil {
		res.GPUPodSpec = other.GPUPodSpec.DeepCopy()
	}

	if other.Image != nil {
		err := copier.CopyWithOption(&res.Image, other.Image, mergeCopier)
		if err != nil {
			return TaskContainerDefaultsConfig{}, fmt.Errorf("merge copying image: %w", err)
		}
	}

	if other.RegistryAuth != nil {
		// Total overwrite, since merging auth doesn't make a lot of sense.
		res.RegistryAuth = other.RegistryAuth
	}

	if other.ForcePullImage {
		res.ForcePullImage = other.ForcePullImage
	}

	if otherEnvVars := other.EnvironmentVariables; otherEnvVars != nil {
		otherEnvs := other.EnvironmentVariables
		res.EnvironmentVariables.CPU = mergeEnvVars(res.EnvironmentVariables.CPU, otherEnvs.CPU)
		res.EnvironmentVariables.CUDA = mergeEnvVars(res.EnvironmentVariables.CUDA, otherEnvs.CUDA)
		res.EnvironmentVariables.ROCM = mergeEnvVars(res.EnvironmentVariables.ROCM, otherEnvs.ROCM)
	}

	if other.AddCapabilities != nil {
		caps := set.FromSlice(append(other.AddCapabilities, res.AddCapabilities...))
		res.AddCapabilities = caps.ToSlice()
		slices.Sort(res.AddCapabilities) // Convenience for testing equality.
	}

	if other.DropCapabilities != nil {
		caps := set.FromSlice(append(other.DropCapabilities, res.DropCapabilities...))
		res.DropCapabilities = caps.ToSlice()
		slices.Sort(res.DropCapabilities) // Convenience for testing equality.
	}

	if other.Devices != nil {
		tmp := res.Devices
		res.Devices = other.Devices

		containerPaths := set.New[string]()
		for _, d := range res.Devices {
			containerPaths.Insert(d.ContainerPath)
		}
		for _, d := range tmp {
			if containerPaths.Contains(d.ContainerPath) {
				continue
			}
			res.Devices = append(res.Devices, d)
		}
	}

	if other.BindMounts != nil {
		tmp := res.BindMounts
		res.BindMounts = other.BindMounts

		containerPaths := set.New[string]()
		for _, b := range res.BindMounts {
			containerPaths.Insert(b.ContainerPath)
		}
		for _, b := range tmp {
			if containerPaths.Contains(b.ContainerPath) {
				continue
			}
			res.BindMounts = append(res.BindMounts, b)
		}
	}

	if other.WorkDir != nil {
		tmp := *other.WorkDir
		res.WorkDir = &tmp
	}

	if other.Slurm.GpuType() != nil {
		res.Slurm.SetGpuType(other.Slurm.GpuType())
	}
	if other.Slurm.SlotsPerNode() != nil {
		res.Slurm.SetSlotsPerNode(other.Slurm.SlotsPerNode())
	}
	if len(other.Slurm.SbatchArgs()) > 0 {
		tmp := slices.Clone(append(other.Slurm.SbatchArgs(), res.Slurm.SbatchArgs()...))
		res.Slurm.SetSbatchArgs(tmp)
	}

	if other.Pbs.SlotsPerNode() != nil {
		res.Pbs.SetSlotsPerNode(other.Pbs.SlotsPerNode())
	}
	if len(other.Pbs.SbatchArgs()) > 0 {
		tmp := slices.Clone(append(other.Pbs.SbatchArgs(), res.Pbs.SbatchArgs()...))
		res.Pbs.SetSbatchArgs(tmp)
	}

	return res, nil
}

func mergeEnvVars(self, other []string) []string {
	var result []string
	uniques := set.New[string]()
	for _, v := range other {
		uniques.Insert(envVarName(v))
		result = append(result, v)
	}

	for _, v := range self {
		if uniques.Contains(envVarName(v)) {
			continue
		}
		result = append(result, v)
	}
	return result
}

func envVarName(v string) string {
	parts := strings.Split(v, "=")
	return parts[0]
}
