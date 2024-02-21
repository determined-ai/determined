package config

import (
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/check"
	"github.com/determined-ai/determined/master/pkg/device"
	"github.com/determined-ai/determined/master/pkg/union"
)

const (
	defaultRMName = "defaultrm"
	defaultRPName = "default"
)

// ResourceManagerConfigV1 hosts configuration fields for the resource manager.
type ResourceManagerConfigV1 struct {
	AgentRM      *AgentResourceManagerConfigV1      `union:"type,agent" json:"-"`
	KubernetesRM *KubernetesResourceManagerConfigV1 `union:"type,kubernetes" json:"-"`
}

// Pools returns pools for config.
func (r *ResourceManagerConfigV1) Pools() []ResourcePoolConfig {
	if agentRM := r.AgentRM; agentRM != nil {
		return agentRM.ResourcePools
	}
	if k8RM := r.KubernetesRM; k8RM != nil {
		return k8RM.ResourcePools
	}
	// TODO dispatcher.

	panic(fmt.Sprintf("unknown rm type %+v", r))
}

func (r *ResourceManagerConfigV1) setPools(pools []ResourcePoolConfig) {
	switch {
	case r.AgentRM != nil:
		r.AgentRM.ResourcePools = pools
	case r.KubernetesRM != nil:
		r.KubernetesRM.ResourcePools = pools
		// TODO dispatcher.
	default:
		panic(fmt.Sprintf("unknown rm type %+v", r))
	}
}

// Name returns name for the resource manager.
func (r *ResourceManagerConfigV1) Name() string {
	if agentRM := r.AgentRM; agentRM != nil {
		return agentRM.Name
	}
	if k8RM := r.KubernetesRM; k8RM != nil {
		return k8RM.Name
	}
	// TODO dispatcher.

	panic(fmt.Sprintf("unknown rm type %+v", r))
}

func (r *ResourceManagerConfigV1) setName(name string) {
	switch {
	case r.AgentRM != nil:
		r.AgentRM.Name = name
	case r.KubernetesRM != nil:
		r.KubernetesRM.Name = name
		// TODO dispatcher.
	default:
		panic(fmt.Sprintf("unknown rm type %+v", r))
	}
}

// MarshalJSON implements the json.Marshaler interface.
func (r ResourceManagerConfigV1) MarshalJSON() ([]byte, error) {
	return union.Marshal(r)
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (r *ResourceManagerConfigV1) UnmarshalJSON(data []byte) error {
	if err := union.Unmarshal(data, r); err != nil {
		return err
	}

	type DefaultParser *ResourceManagerConfigV1
	if err := json.Unmarshal(data, DefaultParser(r)); err != nil {
		return err
	}

	// Fill in the default config.
	if r.AgentRM == nil && r.KubernetesRM == nil {
		r.AgentRM = &AgentResourceManagerConfigV1{ //nolint:exhaustruct
			Name: defaultRMName,
			Scheduler: &SchedulerConfig{
				FittingPolicy: defaultFitPolicy,
			},
			DefaultComputeResourcePool: defaultRPName,
			DefaultAuxResourcePool:     defaultRPName,
		}
	}
	return nil
}

// AgentResourceManagerConfigV1 hosts configuration fields for the determined resource manager.
type AgentResourceManagerConfigV1 struct {
	Scheduler                  *SchedulerConfig `json:"scheduler"`
	DefaultAuxResourcePool     string           `json:"default_aux_resource_pool"`
	DefaultComputeResourcePool string           `json:"default_compute_resource_pool"`
	NoDefaultResourcePools     bool             `json:"no_default_resource_pools"`
	// Deprecated: use DefaultAuxResourcePool instead.
	DefaultCPUResourcePool string `json:"default_cpu_resource_pool,omitempty"`
	// Deprecated: use DefaultComputeResourcePool instead.
	DefaultGPUResourcePool string `json:"default_gpu_resource_pool,omitempty"`

	Name          string               `json:"name"`
	Metadata      map[string]any       `json:"metadata"`
	ResourcePools []ResourcePoolConfig `json:"resource_pools"`

	RequireAuthentication bool   `json:"require_authentication"`
	ClientCA              string `json:"client_ca"`
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (a *AgentResourceManagerConfigV1) UnmarshalJSON(data []byte) error {
	type DefaultParser *AgentResourceManagerConfigV1
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

// Validate implements the check.Validatable interface.
func (a AgentResourceManagerConfigV1) Validate() []error {
	if a.NoDefaultResourcePools {
		return []error{
			check.Equal("", a.DefaultAuxResourcePool,
				"default_aux_resource_pool should be empty if no_default_resource_pools is set"),
			check.Equal("", a.DefaultComputeResourcePool,
				"default_compute_resource_pool should be empty if no_default_resource_pools is "+
					"set"),
		}
	}

	return []error{
		check.NotEmpty(a.DefaultAuxResourcePool, "default_aux_resource_pool should be non-empty"),
		check.NotEmpty(a.DefaultComputeResourcePool, "default_compute_resource_pool should be non-empty"),
		check.NotEmpty(a.Name, "name is required"),
	}
}

// KubernetesResourceManagerConfigV1 hosts configuration fields for the kubernetes resource manager.
type KubernetesResourceManagerConfigV1 struct {
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

	Name          string               `json:"name"`
	Metadata      map[string]any       `json:"metadata"`
	ResourcePools []ResourcePoolConfig `json:"resource_pools"`

	DefaultAuxResourcePool     string `json:"default_aux_resource_pool"`
	DefaultComputeResourcePool string `json:"default_compute_resource_pool"`
	NoDefaultResourcePools     bool   `json:"no_default_resource_pools"`
}

// GetPreemption returns whether the RM is set to preempt.
func (k *KubernetesResourceManagerConfigV1) GetPreemption() bool {
	return k.DefaultScheduler == PreemptionScheduler
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (k *KubernetesResourceManagerConfigV1) UnmarshalJSON(data []byte) error {
	//nolint:exhaustruct
	*k = KubernetesResourceManagerConfigV1{
		SlotType: device.CUDA, // default to CUDA-backed slots.
	}
	type DefaultParser *KubernetesResourceManagerConfigV1
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

// Validate implements the check.Validatable interface.
func (k KubernetesResourceManagerConfigV1) Validate() []error {
	var checkSlotType error
	switch k.SlotType {
	case device.CPU, device.CUDA:
		break
	case device.ROCM:
		checkSlotType = errors.Errorf("rocm slot_type is not supported yet on k8s")
	default:
		checkSlotType = errors.Errorf("slot_type must be either cuda or cpu")
	}

	var checkCPUResource error
	if k.SlotType == device.CPU {
		checkCPUResource = check.GreaterThan(
			k.SlotResourceRequests.CPU, float32(0), "slot_resource_requests.cpu must be > 0")
	}
	return []error{
		checkSlotType,
		checkCPUResource,
		check.NotEmpty(k.Name, "name is required"),
	}
}

// PodSlotResourceRequests contains the per-slot container requests.
type PodSlotResourceRequests struct {
	CPU float32 `json:"cpu"`
}

// FluentConfig stores k8s-configurable Fluent Bit-related options.
type FluentConfig struct {
	Image string `json:"image"`
	UID   int    `json:"uid"`
	GID   int    `json:"gid"`
}

// PreemptionScheduler is the name of the preemption scheduler for k8.
// HACK(Brad): Here because circular imports; Kubernetes probably needs its own
// configuration package.
const PreemptionScheduler = "preemption"
