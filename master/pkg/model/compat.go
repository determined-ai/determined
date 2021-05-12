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
		RawMode:          ptrs.StringPtr(d.Mode),
	}).(expconf.Device)
}

// ToExpconf translates old model objects into an expconf object.
func (d DevicesConfig) ToExpconf() expconf.DevicesConfig {
	var out expconf.DevicesConfig
	for _, d := range d {
		out = append(out, d.ToExpconf())
	}
	return schemas.WithDefaults(out).(expconf.DevicesConfig)
}

// ToExpconf translates old model objects into an expconf object.
func (r ResourcesConfig) ToExpconf() expconf.ResourcesConfig {
	return schemas.WithDefaults(expconf.ResourcesConfig{
		RawSlots:          ptrs.IntPtr(r.Slots),
		RawMaxSlots:       r.MaxSlots,
		RawSlotsPerTrial:  ptrs.IntPtr(r.SlotsPerTrial),
		RawWeight:         ptrs.Float64Ptr(r.Weight),
		RawNativeParallel: ptrs.BoolPtr(r.NativeParallel),
		RawShmSize:        r.ShmSize,
		RawAgentLabel:     ptrs.StringPtr(r.AgentLabel),
		RawResourcePool:   ptrs.StringPtr(r.ResourcePool),
		RawPriority:       r.Priority,
		RawDevices:        r.Devices.ToExpconf(),
	}).(expconf.ResourcesConfig)
}

// ToExpconf translates old model objects into an expconf object.
func (b BindMount) ToExpconf() expconf.BindMount {
	return schemas.WithDefaults(expconf.BindMount{
		RawHostPath:      b.HostPath,
		RawContainerPath: b.ContainerPath,
		RawReadOnly:      ptrs.BoolPtr(b.ReadOnly),
		RawPropagation:   ptrs.StringPtr(b.Propagation),
	}).(expconf.BindMount)
}

// ToExpconf translates old model objects into an expconf object.
func (b BindMountsConfig) ToExpconf() expconf.BindMountsConfig {
	var out expconf.BindMountsConfig
	for _, m := range b {
		out = append(out, m.ToExpconf())
	}
	return schemas.WithDefaults(out).(expconf.BindMountsConfig)
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
		RawCPU: r.CPU,
		RawGPU: r.GPU,
	}).(expconf.EnvironmentVariablesMap)
}

// ToExpconf translates old model objects into an expconf object.
func (r RuntimeItem) ToExpconf() expconf.EnvironmentImageMap {
	return schemas.WithDefaults(expconf.EnvironmentImageMap{
		RawCPU: ptrs.StringPtr(r.CPU),
		RawGPU: ptrs.StringPtr(r.GPU),
	}).(expconf.EnvironmentImageMap)
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
		RawForcePullImage:       ptrs.BoolPtr(e.ForcePullImage),
		RawPodSpec:              e.PodSpec,
	}).(expconf.EnvironmentConfig)
}
