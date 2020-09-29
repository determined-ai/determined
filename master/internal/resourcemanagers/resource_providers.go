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
// Currently support only one resource provider at a time.
type ResourceManagers struct {
	ref *actor.Ref
}

// NewResourceManagers creates an instance of ResourceManagers.
func NewResourceManagers(ref *actor.Ref) *ResourceManagers {
	return &ResourceManagers{
		ref: ref,
	}
}

// Receive implements the actor.Actor interface.
func (rp *ResourceManagers) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case
		AllocateRequest, ResourcesReleased,
		sproto.SetGroupMaxSlots, sproto.SetGroupWeight,
		GetTaskSummary, GetTaskSummaries,
		sproto.ConfigureEndpoints, sproto.GetEndpointActorAddress:
		rp.forward(ctx, msg)

	default:
		return actor.ErrUnexpectedMessage(ctx)
	}

	return nil
}

func (rp *ResourceManagers) forward(ctx *actor.Context, msg actor.Message) {
	if ctx.ExpectingResponse() {
		response := ctx.Ask(rp.ref, msg)
		ctx.Respond(response.Get())
	} else {
		ctx.Tell(rp.ref, msg)
	}
}
