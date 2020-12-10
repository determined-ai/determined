package resourcemanagers

import (
	"crypto/tls"
	"time"

	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
)

const (
	actionCoolDown = 500 * time.Millisecond
)

// schedulerTick periodically triggers the scheduler to act.
type schedulerTick struct{}

// ResourceManagers manages the configured resource providers.
// Currently support only one resource manager at a time.
type ResourceManagers struct {
	ref *actor.Ref
}

// NewResourceManagers creates an instance of ResourceManagers.
func NewResourceManagers(
	system *actor.System, config *ResourceConfig, cert *tls.Certificate,
) *ResourceManagers {
	var ref *actor.Ref
	switch {
	case config.ResourceManager.AgentRM != nil:
		ref, _ = system.ActorOf(
			actor.Addr("agentRM"),
			newAgentResourceManager(config, cert),
		)

	case config.ResourceManager.KubernetesRM != nil:
		ref, _ = system.ActorOf(
			actor.Addr("kubernetesRM"),
			newKubernetesResourceManager(config.ResourceManager.KubernetesRM),
		)

	default:
		panic("no expected resource manager config is defined")
	}

	return &ResourceManagers{ref: ref}
}

// Receive implements the actor.Actor interface.
func (rm *ResourceManagers) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case
		AllocateRequest, ResourcesReleased,
		sproto.SetGroupMaxSlots, sproto.SetGroupWeight,
		sproto.SetGroupPriority, GetTaskSummary,
		GetTaskSummaries, SetTaskName:
		rm.forward(ctx, msg)

	default:
		return actor.ErrUnexpectedMessage(ctx)
	}

	return nil
}

func (rm *ResourceManagers) forward(ctx *actor.Context, msg actor.Message) {
	if ctx.ExpectingResponse() {
		response := ctx.Ask(rm.ref, msg)
		ctx.Respond(response.Get())
	} else {
		ctx.Tell(rm.ref, msg)
	}
}

// GetResourceManagerType returns the type of resourceManager being used.
func GetResourceManagerType(rmConfig *ResourceManagerConfig) string {
	switch {
	case rmConfig.AgentRM != nil:
		return "agentsRM"

	case rmConfig.KubernetesRM != nil:
		return "kubernetesRM"

	default:
		panic("no expected resource manager config is defined")
	}
}
