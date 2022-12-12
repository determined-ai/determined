package model

import (
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
)

// ToExpconf translates old model objects into an expconf object.
func (d DeviceConfig) ToExpconf() expconf.Device {
	return schemas.WithDefaults(expconf.Device{
		RawHostPath:      d.HostPath,
		RawContainerPath: d.ContainerPath,
		RawMode:          ptrs.Ptr(d.Mode),
	})
}

// ToExpconf translates old model objects into an expconf object.
func (d DevicesConfig) ToExpconf() expconf.DevicesConfig {
	var out expconf.DevicesConfig
	for _, d := range d {
		out = append(out, d.ToExpconf())
	}
	return schemas.WithDefaults(out)
}

// ToExpconf translates old model objects into an expconf object.
func (r ResourcesConfig) ToExpconf() expconf.ResourcesConfig {
	var shm *int
	if r.ShmSize != nil {
		shm = ptrs.Ptr(int(*r.ShmSize))
	}

	return schemas.WithDefaults(expconf.ResourcesConfig{
		RawSlots:          ptrs.Ptr(r.Slots),
		RawMaxSlots:       r.MaxSlots,
		RawSlotsPerTrial:  ptrs.Ptr(1),
		RawWeight:         ptrs.Ptr(r.Weight),
		RawNativeParallel: ptrs.Ptr(r.NativeParallel),
		RawShmSize:        shm,
		RawAgentLabel:     ptrs.Ptr(r.AgentLabel),
		RawResourcePool:   ptrs.Ptr(r.ResourcePool),
		RawPriority:       r.Priority,
		RawDevices:        r.Devices.ToExpconf(),
	})
}

// ToExpconf translates old model objects into an expconf object.
func (b BindMount) ToExpconf() expconf.BindMount {
	return schemas.WithDefaults(expconf.BindMount{
		RawHostPath:      b.HostPath,
		RawContainerPath: b.ContainerPath,
		RawReadOnly:      ptrs.Ptr(b.ReadOnly),
		RawPropagation:   ptrs.Ptr(b.Propagation),
	})
}

// ToExpconf translates old model objects into an expconf object.
func (b BindMountsConfig) ToExpconf() expconf.BindMountsConfig {
	var out expconf.BindMountsConfig
	for _, m := range b {
		out = append(out, m.ToExpconf())
	}
	return schemas.WithDefaults(out)
}

// ToModelBindMount converts new expconf bind mounts into old modl bind mounts.
func ToModelBindMount(b expconf.BindMount) BindMount {
	return BindMount{
		HostPath:      b.HostPath(),
		ContainerPath: b.ContainerPath(),
		ReadOnly:      b.ReadOnly(),
		Propagation:   b.Propagation(),
	}
}

// ToExpconf translates old model objects into an expconf object.
func (r RuntimeItems) ToExpconf() expconf.EnvironmentVariablesMap {
	return schemas.WithDefaults(expconf.EnvironmentVariablesMap{
		RawCPU:  r.CPU,
		RawCUDA: r.CUDA,
		RawROCM: r.ROCM,
	})
}

// ToExpconf translates old model objects into an expconf object.
func (r RuntimeItem) ToExpconf() expconf.EnvironmentImageMap {
	return schemas.WithDefaults(expconf.EnvironmentImageMap{
		RawCPU:  ptrs.Ptr(r.CPU),
		RawCUDA: ptrs.Ptr(r.CUDA),
		RawROCM: ptrs.Ptr(r.ROCM),
	})
}

// ToExpconf translates old model objects into an expconf object.
func (e Environment) ToExpconf() expconf.EnvironmentConfig {
	image := e.Image.ToExpconf()
	vars := e.EnvironmentVariables.ToExpconf()

	return schemas.WithDefaults(expconf.EnvironmentConfig{
		RawImage:                &image,
		RawEnvironmentVariables: &vars,
		RawPorts:                e.Ports,
		RawRegistryAuth:         e.RegistryAuth,
		RawForcePullImage:       ptrs.Ptr(e.ForcePullImage),
		RawPodSpec:              (*expconf.PodSpec)(e.PodSpec),
		RawAddCapabilities:      e.AddCapabilities,
		RawDropCapabilities:     e.DropCapabilities,
	})
}
