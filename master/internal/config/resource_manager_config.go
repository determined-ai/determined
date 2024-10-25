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
	defaultResourcePoolName = "default"
	validGWPortRangeStart   = 1025
	validGWPortRangeEnd     = 65535
)

// ResourceManagerConfig hosts configuration fields for the resource manager.
type ResourceManagerConfig struct {
	AgentRM      *AgentResourceManagerConfig      `union:"type,agent" json:"-"`
	KubernetesRM *KubernetesResourceManagerConfig `union:"type,kubernetes" json:"-"`
	DispatcherRM *DispatcherResourceManagerConfig `union:"type,slurm" json:"-"`
	PbsRM        *DispatcherResourceManagerConfig `union:"type,pbs" json:"-"`
}

// ClusterName returns the cluster name associated with the resource manager. If the cluster name
// is empty, it gets assigned to the (possibly) assigned resource manager name.
func (r ResourceManagerConfig) ClusterName() string {
	if agentRM := r.AgentRM; agentRM != nil {
		if len(agentRM.ClusterName) == 0 {
			agentRM.ClusterName = agentRM.Name
		}
		return agentRM.ClusterName
	}
	if k8RM := r.KubernetesRM; k8RM != nil {
		if len(k8RM.ClusterName) == 0 {
			k8RM.ClusterName = k8RM.Name
		}
		return k8RM.ClusterName
	}
	if dis := r.DispatcherRM; dis != nil {
		if len(dis.ClusterName) == 0 {
			dis.ClusterName = dis.Name
		}
		return dis.ClusterName
	}
	if pbs := r.PbsRM; pbs != nil {
		if len(pbs.ClusterName) == 0 {
			pbs.ClusterName = pbs.Name
		}
		return pbs.ClusterName
	}

	panic(fmt.Sprintf("unknown rm type %+v", r))
}

func (r *ResourceManagerConfig) setClusterName(clusterName string) {
	switch {
	case r.AgentRM != nil:
		r.AgentRM.ClusterName = clusterName
	case r.KubernetesRM != nil:
		r.KubernetesRM.ClusterName = clusterName
	case r.DispatcherRM != nil:
		r.DispatcherRM.ClusterName = clusterName
	case r.PbsRM != nil:
		r.PbsRM.ClusterName = clusterName
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
	ClusterName                string           `json:"cluster_name"`
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

	// Deprecated: use ClusterName.
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
		check.NotEmpty(a.ClusterName, "cluster_name is required"),
	)
}

// KubernetesResourceManagerConfig hosts configuration fields for the kubernetes resource manager.
type KubernetesResourceManagerConfig struct {
	// Changed from "Namespace" to "DefaultNamespace". DefaultNamespace is an optional field that
	// allows the user to specify the default namespace to bind a workspace to, for each RM.
	DefaultNamespace string `json:"default_namespace"`

	MaxSlotsPerPod *int `json:"max_slots_per_pod"`

	ClusterName              string                  `json:"cluster_name"`
	MasterServiceName        string                  `json:"master_service_name"`
	LeaveKubernetesResources bool                    `json:"leave_kubernetes_resources"`
	DefaultScheduler         string                  `json:"default_scheduler"`
	SlotType                 device.Type             `json:"slot_type"`
	SlotResourceRequests     PodSlotResourceRequests `json:"slot_resource_requests"`
	// deprecated, no longer in use.
	Fluent         FluentConfig `json:"fluent"`
	KubeconfigPath string       `json:"kubeconfig_path"`

	DetMasterScheme string `json:"determined_master_scheme,omitempty"`
	// DeprecatedDetMasterHost shouldn't be used. Use the method DetMasterHost instead.
	DeprecatedDetMasterHost string `json:"determined_master_host,omitempty"`
	// DeprecatedDetMasterIP shouldn't be used. Use the method DetMasterHost instead.
	DeprecatedDetMasterIP string `json:"determined_master_ip,omitempty"`
	DetMasterPort         int32  `json:"determined_master_port,omitempty"`

	DefaultAuxResourcePool     string `json:"default_aux_resource_pool"`
	DefaultComputeResourcePool string `json:"default_compute_resource_pool"`
	NoDefaultResourcePools     bool   `json:"no_default_resource_pools"`

	InternalTaskGateway *InternalTaskGatewayConfig `json:"internal_task_gateway"`

	// Deprecated: use ClusterName.
	Name string `json:"name"`

	Metadata map[string]string `json:"metadata"`
}

// DetMasterHost returns `det_master_host` from the config, falling back to the older `det_master_ip`. Callers
// should use this method instead the fields directly but the fields must be public for YAML deserialization to work.
func (k KubernetesResourceManagerConfig) DetMasterHost() string {
	if k.DeprecatedDetMasterHost != "" {
		return k.DeprecatedDetMasterHost
	}
	return k.DeprecatedDetMasterIP
}

// InternalTaskGatewayConfig is config for exposing Determined tasks to outside of the cluster.
// Useful for multirm when we can only be running in a single cluster.
type InternalTaskGatewayConfig struct {
	// GatewayName as defined in the k8s cluster.
	GatewayName string `json:"gateway_name"`
	// GatewayNamespace as defined in the k8s cluster.
	GatewayNamespace string `json:"gateway_namespace"`
	GatewayIP        string `json:"gateway_ip"`
	// GWPortStart denotes the inclusive start of the available and exclusive port range to
	// MLDE for InternalTaskGateway.
	GWPortStart int `json:"gateway_port_range_start"`
	// GWPortEnd denotes the inclusive end of the available and exclusive port range to
	// MLDE for InternalTaskGateway.
	GWPortEnd int `json:"gateway_port_range_end"`
}

var defaultInternalTaskGatewayConfig = InternalTaskGatewayConfig{
	GWPortStart: 32768,
	GWPortEnd:   65535,
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (i *InternalTaskGatewayConfig) UnmarshalJSON(data []byte) error {
	*i = defaultInternalTaskGatewayConfig
	type DefaultParser *InternalTaskGatewayConfig
	return json.Unmarshal(data, DefaultParser(i))
}

// Validate implements the check.Validatable interface.
func (i *InternalTaskGatewayConfig) Validate() []error {
	var errs []error

	if err := check.IsValidK8sLabel(i.GatewayName); err != nil {
		errs = append(errs, fmt.Errorf("invalid gateway_name: %w", err))
	}

	if err := check.IsValidK8sLabel(i.GatewayNamespace); err != nil {
		errs = append(errs, fmt.Errorf("invalid gateway_namespace: %w", err))
	}

	// Don't validate the IP just check it is not empty so hostnames and the like can be used.
	if err := check.NotEmpty(i.GatewayIP); err != nil {
		errs = append(errs, fmt.Errorf("invalid gateway_ip: %w", err))
	}

	if err := check.BetweenInclusive(
		i.GWPortStart, validGWPortRangeStart, validGWPortRangeEnd); err != nil {
		errs = append(errs, fmt.Errorf("invalid gateway_port_range_start: %w", err))
	}

	if err := check.BetweenInclusive(
		i.GWPortEnd, validGWPortRangeStart, validGWPortRangeEnd); err != nil {
		errs = append(errs, fmt.Errorf("invalid gateway_port_range_end: %w", err))
	}

	if i.GWPortStart >= i.GWPortEnd {
		errs = append(errs, fmt.Errorf("gateway_port_range_start must be less than or equal to gateway_port_range_end"))
	}
	return errs
}

var defaultKubernetesResourceManagerConfig = KubernetesResourceManagerConfig{
	SlotType: device.CUDA, // default to CUDA-backed slots.
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
	var errs []error
	switch k.SlotType {
	case device.CPU, device.CUDA, device.ROCM:
	default:
		errs = append(errs, errors.New("slot_type must be cuda, cpu, or rocm"))
	}

	if k.SlotType == device.CPU {
		errs = append(errs, check.GreaterThan(
			k.SlotResourceRequests.CPU, float32(0), "slot_resource_requests.cpu must be > 0"))
	}

	if k.DefaultScheduler == PriorityScheduling {
		errs = append(errs, errors.New("the ``priority`` scheduler was deprecated, please "+
			"use the default Kubernetes scheduler or coscheduler"))
	} else if k.DefaultScheduler != "" && k.DefaultScheduler != "coscheduler" {
		errs = append(errs, errors.New("only blank or ``coscheduler`` values allowed for Kubernetes scheduler"))
	}

	if k.DeprecatedDetMasterHost != "" && k.DeprecatedDetMasterIP != "" {
		errs = append(errs, errors.New("use the new determined_master_host instead of determined_master_ip, not both"))
	}

	errs = append(errs, check.NotEmpty(k.ClusterName, "cluster_name is required"))
	return errs
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
