package scheduler

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

// ResourceProviders manages the configured resource providers.
// Currently support only one resource provider at a time.
type ResourceProviders struct {
	resourceProvider *actor.Ref
}

// NewResourceProviders creates an instance of ResourceProviders.
func NewResourceProviders(resourceProvider *actor.Ref) *ResourceProviders {
	return &ResourceProviders{
		resourceProvider: resourceProvider,
	}
}

// Receive implements the actor.Actor interface.
func (rp *ResourceProviders) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case sproto.ContainerStateChanged,
		AssignRequest, ResourceReleased,
		SetMaxSlots, SetWeight,
		GetTaskSummary, GetTaskSummaries,
		sproto.ConfigureEndpoints, sproto.GetEndpointActorAddress:
		rp.forward(ctx, msg)

	default:
		return actor.ErrUnexpectedMessage(ctx)
	}

	return nil
}

func (rp *ResourceProviders) forward(ctx *actor.Context, msg actor.Message) {
	if ctx.ExpectingResponse() {
		response := ctx.Ask(rp.resourceProvider, msg)
		ctx.Respond(response.Get())
	} else {
		ctx.Tell(rp.resourceProvider, msg)
	}
}
