package config

import (
	"encoding/json"

	"github.com/determined-ai/determined/master/pkg/device"
	"github.com/determined-ai/determined/master/pkg/union"
)

// ResourceManagerConfigV0 hosts configuration fields for the resource manager.
//
// Deprecated: look at resource_manager_config_v1.go.
type ResourceManagerConfigV0 struct {
	AgentRM      *AgentResourceManagerConfigV0      `union:"type,agent" json:"-"`
	KubernetesRM *KubernetesResourceManagerConfigV0 `union:"type,kubernetes" json:"-"`
}

// ToV1 converts old config format to v1.
func (r *ResourceManagerConfigV0) ToV1() *ResourceManagerConfigV1 {
	if r == nil {
		return nil
	}

	return &ResourceManagerConfigV1{
		AgentRM:      r.AgentRM.ToV1(),
		KubernetesRM: r.KubernetesRM.ToV1(),
		// TODO dispatcher.
	}
}

// MarshalJSON implements the json.Marshaler interface.
func (r ResourceManagerConfigV0) MarshalJSON() ([]byte, error) {
	return union.Marshal(r)
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (r *ResourceManagerConfigV0) UnmarshalJSON(data []byte) error {
	if err := union.Unmarshal(data, r); err != nil {
		return err
	}

	type DefaultParser *ResourceManagerConfigV0
	if err := json.Unmarshal(data, DefaultParser(r)); err != nil {
		return err
	}

	// Fill in the default config.
	if r.AgentRM == nil && r.KubernetesRM == nil {
		r.AgentRM = &AgentResourceManagerConfigV0{
			Scheduler: &SchedulerConfig{
				FittingPolicy: defaultFitPolicy,
			},
			DefaultComputeResourcePool: defaultRPName,
			DefaultAuxResourcePool:     defaultRPName,
		}
	}
	return nil
}

// AgentResourceManagerConfigV0 hosts configuration fields for the determined resource manager.
//
// Deprecated: look at resource_manager_config_v1.go.
type AgentResourceManagerConfigV0 struct {
	Scheduler                  *SchedulerConfig `json:"scheduler"`
	DefaultAuxResourcePool     string           `json:"default_aux_resource_pool"`
	DefaultComputeResourcePool string           `json:"default_compute_resource_pool"`
	NoDefaultResourcePools     bool             `json:"no_default_resource_pools"`
	// Deprecated: use DefaultAuxResourcePool instead.
	DefaultCPUResourcePool string `json:"default_cpu_resource_pool,omitempty"`
	// Deprecated: use DefaultComputeResourcePool instead.
	DefaultGPUResourcePool string `json:"default_gpu_resource_pool,omitempty"`

	RequireAuthentication bool   `json:"require_authentication"`
	ClientCA              string `json:"client_ca"`
}

// ToV1 converts old config format to v1.
func (a *AgentResourceManagerConfigV0) ToV1() *AgentResourceManagerConfigV1 {
	if a == nil {
		return nil
	}

	return &AgentResourceManagerConfigV1{
		Scheduler:                  a.Scheduler,
		DefaultAuxResourcePool:     a.DefaultAuxResourcePool,
		DefaultComputeResourcePool: a.DefaultComputeResourcePool,
		NoDefaultResourcePools:     a.NoDefaultResourcePools,
		DefaultCPUResourcePool:     a.DefaultCPUResourcePool,
		DefaultGPUResourcePool:     a.DefaultGPUResourcePool,
		RequireAuthentication:      a.RequireAuthentication,
		ClientCA:                   a.ClientCA,
		Name:                       defaultRMName,
		Metadata:                   nil,
		ResourcePools:              nil,
	}
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (a *AgentResourceManagerConfigV0) UnmarshalJSON(data []byte) error {
	type DefaultParser *AgentResourceManagerConfigV0
	if err := json.Unmarshal(data, DefaultParser(a)); err != nil {
		return err
	}

	if a.NoDefaultResourcePools {
		a.DefaultComputeResourcePool = ""
		a.DefaultAuxResourcePool = ""
	} else {
		if a.DefaultAuxResourcePool == "" && a.DefaultCPUResourcePool != "" {
			a.DefaultAuxResourcePool = a.DefaultCPUResourcePool
		}
		if a.DefaultComputeResourcePool == "" && a.DefaultGPUResourcePool != "" {
			a.DefaultComputeResourcePool = a.DefaultGPUResourcePool
		}
		if a.DefaultComputeResourcePool == "" {
			a.DefaultComputeResourcePool = defaultRPName
		}
		if a.DefaultAuxResourcePool == "" {
			a.DefaultAuxResourcePool = defaultRPName
		}
	}

	a.DefaultCPUResourcePool = ""
	a.DefaultGPUResourcePool = ""

	return nil
}

// KubernetesResourceManagerConfigV0 hosts configuration fields for the kubernetes resource manager.
//
// Deprecated: look at resource_manager_config_v1.go.
type KubernetesResourceManagerConfigV0 struct {
	Namespace string `json:"namespace"`

	// Deprecated: this can be per resource pool now on taskContainerDefaults.
	// This will always be the same as global
	// task_container_defaults.kubernetes.max_slots_per_pod so use that.
	MaxSlotsPerPod *int `json:"max_slots_per_pod"`

	MasterServiceName        string                  `json:"master_service_name"`
	LeaveKubernetesResources bool                    `json:"leave_kubernetes_resources"`
	DefaultScheduler         string                  `json:"default_scheduler"`
	SlotType                 device.Type             `json:"slot_type"`
	SlotResourceRequests     PodSlotResourceRequests `json:"slot_resource_requests"`
	// deprecated, no longer in use.
	Fluent     FluentConfig `json:"fluent"`
	CredsDir   string       `json:"_creds_dir,omitempty"`
	MasterIP   string       `json:"_master_ip,omitempty"`
	MasterPort int32        `json:"_master_port,omitempty"`

	DefaultAuxResourcePool     string `json:"default_aux_resource_pool"`
	DefaultComputeResourcePool string `json:"default_compute_resource_pool"`
	NoDefaultResourcePools     bool   `json:"no_default_resource_pools"`
}

// ToV1 converts old config format to v1.
func (k *KubernetesResourceManagerConfigV0) ToV1() *KubernetesResourceManagerConfigV1 {
	if k == nil {
		return nil
	}

	return &KubernetesResourceManagerConfigV1{
		Namespace:                  k.Namespace,
		MaxSlotsPerPod:             k.MaxSlotsPerPod,
		MasterServiceName:          k.MasterServiceName,
		LeaveKubernetesResources:   k.LeaveKubernetesResources,
		DefaultScheduler:           k.DefaultScheduler,
		SlotType:                   k.SlotType,
		SlotResourceRequests:       k.SlotResourceRequests,
		Fluent:                     k.Fluent,
		CredsDir:                   k.CredsDir,
		MasterIP:                   k.MasterIP,
		MasterPort:                 k.MasterPort,
		DefaultAuxResourcePool:     k.DefaultAuxResourcePool,
		DefaultComputeResourcePool: k.DefaultComputeResourcePool,
		NoDefaultResourcePools:     k.NoDefaultResourcePools,
		Name:                       defaultRMName,
		Metadata:                   nil,
		ResourcePools:              nil,
	}
}

// GetPreemption returns whether the RM is set to preempt.
func (k *KubernetesResourceManagerConfigV0) GetPreemption() bool {
	return k.DefaultScheduler == PreemptionScheduler
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (k *KubernetesResourceManagerConfigV0) UnmarshalJSON(data []byte) error {
	*k = KubernetesResourceManagerConfigV0{
		SlotType: device.CUDA, // default to CUDA-backed slots.
	}
	type DefaultParser *KubernetesResourceManagerConfigV0
	err := json.Unmarshal(data, DefaultParser(k))

	if k.NoDefaultResourcePools {
		k.DefaultComputeResourcePool = ""
		k.DefaultAuxResourcePool = ""
	} else {
		if k.DefaultComputeResourcePool == "" {
			k.DefaultComputeResourcePool = defaultRPName
		}
		if k.DefaultAuxResourcePool == "" {
			k.DefaultAuxResourcePool = defaultRPName
		}
	}

	if err == nil && k.SlotType == "gpu" { //nolint:goconst
		k.SlotType = device.CUDA
	}
	return err
}
