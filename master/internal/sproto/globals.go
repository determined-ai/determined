package sproto

import (
	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/actor"
)

var (
	// ResourceManagerAddr is the actor address of the resource manager router.
	ResourceManagerAddr = actor.Addr("resourceManagers")
	// AgentRMAddr is the actor address of the agent resource manager.
	AgentRMAddr = actor.Addr("agentRM")
	// K8sRMAddr is the actor address of the k8s resource manager.
	K8sRMAddr = actor.Addr("kubernetesRM")
	// AgentsAddr is the actor address of the agents.
	AgentsAddr = actor.Addr("agents")
	// PodsAddr is the actor address of the pods.
	PodsAddr = actor.Addr("pods")
)

type (
	// GetDefaultComputeResourcePoolRequest is a message asking for the name of the default
	// GPU resource pool
	GetDefaultComputeResourcePoolRequest struct{}

	// GetDefaultComputeResourcePoolResponse is the response to GetDefaultComputeResourcePoolRequest
	GetDefaultComputeResourcePoolResponse struct {
		PoolName string
	}

	// GetDefaultAuxResourcePoolRequest is a message asking for the name of the default
	// CPU resource pool
	GetDefaultAuxResourcePoolRequest struct{}

	// GetDefaultAuxResourcePoolResponse is the response to GetDefaultAuxResourcePoolRequest
	GetDefaultAuxResourcePoolResponse struct {
		PoolName string
	}
)

// GetRM returns the resource manager router.
func GetRM(system *actor.System) *actor.Ref {
	return system.Get(ResourceManagerAddr)
}

// UseAgentRM returns if using the agent resource manager.
func UseAgentRM(system *actor.System) bool {
	return system.Get(AgentsAddr) != nil
}

// UseK8sRM returns if using the kubernetes resource manager.
func UseK8sRM(system *actor.System) bool {
	return system.Get(PodsAddr) != nil
}

// GetRP returns the resource pool.
func GetRP(system *actor.System, name string) *actor.Ref {
	if rm := system.Get(AgentRMAddr); rm != nil {
		return rm.Child(name)
	}
	return nil
}

// GetCurrentRM returns either the k8s resource manager or the agents
// resource manager, depending on which exists.
func GetCurrentRM(system *actor.System) *actor.Ref {
	if UseK8sRM(system) {
		return system.Get(K8sRMAddr)
	}
	if UseAgentRM(system) {
		return system.Get(AgentRMAddr)
	}
	panic("There should either be a k8s resource manager or an agent resource manager")
}

// GetDefaultComputeResourcePool returns the default GPU resource pool.
func GetDefaultComputeResourcePool(system *actor.System) string {
	resp := system.Ask(GetCurrentRM(system), GetDefaultComputeResourcePoolRequest{}).Get()
	return resp.(GetDefaultComputeResourcePoolResponse).PoolName
}

// GetDefaultAuxResourcePool returns the default CPU resource pool.
func GetDefaultAuxResourcePool(system *actor.System) string {
	resp := system.Ask(GetCurrentRM(system), GetDefaultAuxResourcePoolRequest{}).Get()
	return resp.(GetDefaultAuxResourcePoolResponse).PoolName
}

// ValidateRP validates if the resource pool exists when using the agent resource manager.
func ValidateRP(system *actor.System, name string) error {
	if name == "" || UseAgentRM(system) && GetRP(system, name) != nil {
		return nil
	}
	return errors.Errorf("cannot find resource pool: %s", name)
}
