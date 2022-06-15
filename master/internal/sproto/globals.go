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
	// GPU resource pool.
	GetDefaultComputeResourcePoolRequest struct{}

	// GetDefaultComputeResourcePoolResponse is the response to
	// GetDefaultComputeResourcePoolRequest.
	GetDefaultComputeResourcePoolResponse struct {
		PoolName string
	}

	// GetDefaultAuxResourcePoolRequest is a message asking for the name of the default
	// CPU resource pool.
	GetDefaultAuxResourcePoolRequest struct{}

	// GetDefaultAuxResourcePoolResponse is the response to GetDefaultAuxResourcePoolRequest.
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
	return system.Get(AgentRMAddr) != nil
}

// UseK8sRM returns if using the kubernetes resource manager.
func UseK8sRM(system *actor.System) bool {
	return system.Get(K8sRMAddr) != nil
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

// ValidateResourcePool validates if the resource pool exists when using the agent resource manager,
// or if it's the dummy kubernetes pool.
func ValidateResourcePool(system *actor.System, name string) error {
	if name == "" || UseAgentRM(system) && GetRP(system, name) != nil ||
		UseK8sRM(system) && name == "kubernetes" {
		return nil
	}
	return errors.Errorf("cannot find resource pool: %s", name)
}

// GetResourcePool returns the validated resource pool name based on the value set in
// the configuration.
func GetResourcePool(
	system *actor.System, poolName string, slots int, command bool,
) (string, error) {
	// If the resource pool isn't set, fill in the default at creation time.
	if poolName == "" {
		if slots == 0 {
			resp := system.Ask(GetCurrentRM(system), GetDefaultAuxResourcePoolRequest{}).Get()
			poolName = resp.(GetDefaultAuxResourcePoolResponse).PoolName
		} else {
			resp := system.Ask(GetCurrentRM(system), GetDefaultComputeResourcePoolRequest{}).Get()
			poolName = resp.(GetDefaultComputeResourcePoolResponse).PoolName
		}
	}

	if err := ValidateResourcePool(system, poolName); err != nil {
		return "", errors.Wrapf(err, "resource pool does not exist: %s", poolName)
	}

	if slots > 0 && command {
		fillable, err := ValidateRPResources(system, poolName, slots)
		if err != nil {
			return "", errors.Wrapf(err, "failed to check resource pool resources: %s", poolName)
		}
		if !fillable {
			return "", errors.New(
				"resource request unfulfillable, please try requesting less slots",
			)
		}
	}
	return poolName, nil
}
