package resourcemanagers

import (
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

// Receive implements the actor.Actor interface.
func (rm *ResourceManagers) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case
		sproto.AllocateRequest, sproto.ResourcesReleased,
		sproto.SetGroupMaxSlots, sproto.SetGroupWeight,
		sproto.SetGroupPriority, sproto.GetTaskSummary,
		sproto.GetTaskSummaries, sproto.SetTaskName,
		sproto.GetTaskHandler:
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
