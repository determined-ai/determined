package sproto

import (
	"github.com/determined-ai/determined/master/pkg/actor"
)

var (
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
