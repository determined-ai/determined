package config

import (
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/check"
	"github.com/determined-ai/determined/master/pkg/device"
	"github.com/determined-ai/determined/master/pkg/union"
)

const defaultResourcePoolName = "default"

// ResourceManagerConfig hosts configuration fields for the resource manager.
type ResourceManagerConfig struct {
	AgentRM      *AgentResourceManagerConfig      `union:"type,agent" json:"-"`
	KubernetesRM *KubernetesResourceManagerConfig `union:"type,kubernetes" json:"-"`
	DispatcherRM *DispatcherResourceManagerConfig `union:"type,slurm" json:"-"`
	PbsRM        *DispatcherResourceManagerConfig `union:"type,pbs" json:"-"`
}

// Name returns the name for the resource manager.
func (r ResourceManagerConfig) Name() string {
	if agentRM := r.AgentRM; agentRM != nil {
		return agentRM.Name
	}
	if k8RM := r.KubernetesRM; k8RM != nil {
		return k8RM.Name
	}
	if dis := r.DispatcherRM; dis != nil {
		return dis.Name
	}
	if pbs := r.PbsRM; pbs != nil {
		return pbs.Name
	}

	panic(fmt.Sprintf("unknown rm type %+v", r))
}

func (r *ResourceManagerConfig) setName(name string) {
	switch {
	case r.AgentRM != nil:
		r.AgentRM.Name = name
	case r.KubernetesRM != nil:
		r.KubernetesRM.Name = name
	case r.DispatcherRM != nil:
		r.DispatcherRM.Name = name
	case r.PbsRM != nil:
		r.PbsRM.Name = name
	default:
		panic(fmt.Sprintf("unknown rm type %+v", r))
	}
}

// MarshalJSON implements the json.Marshaler interface.
func (r ResourceManagerConfig) MarshalJSON() ([]byte, error) {
	return union.Marshal(r)
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (r *ResourceManagerConfig) UnmarshalJSON(data []byte) error {
	if err := union.Unmarshal(data, r); err != nil {
		return err
	}

	type DefaultParser *ResourceManagerConfig
	if err := json.Unmarshal(data, DefaultParser(r)); err != nil {
		return err
	}

	// Fill in the default config.
	if r.AgentRM == nil && r.KubernetesRM == nil && r.DispatcherRM == nil && r.PbsRM == nil {
		r.AgentRM = &AgentResourceManagerConfig{
			Scheduler: &SchedulerConfig{
				FittingPolicy: defaultFitPolicy,
			},
			DefaultComputeResourcePool: defaultResourcePoolName,
			DefaultAuxResourcePool:     defaultResourcePoolName,
		}
	}
	return nil
}

// AgentResourceManagerConfig hosts configuration fields for the determined resource manager.
type AgentResourceManagerConfig struct {
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

	Name     string            `json:"name"`
	Metadata map[string]string `json:"metadata"`
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (a *AgentResourceManagerConfig) UnmarshalJSON(data []byte) error {
	type DefaultParser *AgentResourceManagerConfig
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
			a.DefaultComputeResourcePool = defaultResourcePoolName
		}
		if a.DefaultAuxResourcePool == "" {
			a.DefaultAuxResourcePool = defaultResourcePoolName
		}
	}

	a.DefaultCPUResourcePool = ""
	a.DefaultGPUResourcePool = ""

	return nil
}

// Validate implements the check.Validatable interface.
func (a AgentResourceManagerConfig) Validate() []error {
	var errors []error
	if a.NoDefaultResourcePools {
		errors = append(errors,
			check.Equal("", a.DefaultAuxResourcePool,
				"default_aux_resource_pool should be empty if no_default_resource_pools is set"),
			check.Equal("", a.DefaultComputeResourcePool,
				"default_compute_resource_pool should be empty if no_default_resource_pools is "+
					"set"))
	}
	return append(errors,
		check.NotEmpty(a.DefaultAuxResourcePool, "default_aux_resource_pool should be non-empty"),
		check.NotEmpty(a.DefaultComputeResourcePool, "default_compute_resource_pool should be non-empty"),
		check.NotEmpty(a.Name, "name is required"),
	)
}

// KubernetesResourceManagerConfig hosts configuration fields for the kubernetes resource manager.
type KubernetesResourceManagerConfig struct {
	Namespace string `json:"namespace"`

	MaxSlotsPerPod *int `json:"max_slots_per_pod"`

	MasterServiceName        string                  `json:"master_service_name"`
	LeaveKubernetesResources bool                    `json:"leave_kubernetes_resources"`
	DefaultScheduler         string                  `json:"default_scheduler"`
	SlotType                 device.Type             `json:"slot_type"`
	SlotResourceRequests     PodSlotResourceRequests `json:"slot_resource_requests"`
	// deprecated, no longer in use.
	Fluent         FluentConfig `json:"fluent"`
	KubeconfigPath string       `json:"kubeconfig_path"`
	DetMasterIP    string       `json:"determined_master_ip,omitempty"`
	DetMasterPort  int32        `json:"determined_master_port,omitempty"`

	DefaultAuxResourcePool     string `json:"default_aux_resource_pool"`
	DefaultComputeResourcePool string `json:"default_compute_resource_pool"`
	NoDefaultResourcePools     bool   `json:"no_default_resource_pools"`

	Name     string            `json:"name"`
	Metadata map[string]string `json:"metadata"`
}

var defaultKubernetesResourceManagerConfig = KubernetesResourceManagerConfig{
	SlotType: device.CUDA, // default to CUDA-backed slots.
}

// GetPreemption returns whether the RM is set to preempt.
func (k *KubernetesResourceManagerConfig) GetPreemption() bool {
	return k.DefaultScheduler == PreemptionScheduler
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (k *KubernetesResourceManagerConfig) UnmarshalJSON(data []byte) error {
	*k = defaultKubernetesResourceManagerConfig
	type DefaultParser *KubernetesResourceManagerConfig
	err := json.Unmarshal(data, DefaultParser(k))

	if k.NoDefaultResourcePools {
		k.DefaultComputeResourcePool = ""
		k.DefaultAuxResourcePool = ""
	} else {
		if k.DefaultComputeResourcePool == "" {
			k.DefaultComputeResourcePool = defaultResourcePoolName
		}
		if k.DefaultAuxResourcePool == "" {
			k.DefaultAuxResourcePool = defaultResourcePoolName
		}
	}

	if err == nil && k.SlotType == "gpu" {
		k.SlotType = device.CUDA
	}
	return err
}

// Validate implements the check.Validatable interface.
func (k KubernetesResourceManagerConfig) Validate() []error {
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
