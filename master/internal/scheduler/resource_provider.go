package scheduler

import (
	"github.com/determined-ai/determined/master/pkg/actor"
	sproto "github.com/determined-ai/determined/master/pkg/scheduler"
)

// ResourceProvider manages the configured resource providers.
// Currently support only one resource provider at a time.
type ResourceProvider struct {
	resourceProvider *actor.Ref
}

// NewResourceProvider creates and instance of ResourceProvider.
func NewResourceProvider(resourceProvider *actor.Ref) *ResourceProvider {
	return &ResourceProvider{
		resourceProvider: resourceProvider,
	}
}

// Receive implements the actor.Actor interface.
func (rp *ResourceProvider) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case AddTask, StartTask, sproto.ContainerStateChanged, SetMaxSlots, SetWeight,
		SetTaskName, TerminateTask, GetTaskSummary, GetTaskSummaries, sproto.ConfigureEndpoints:
		rp.forward(ctx, msg)

	default:
		return actor.ErrUnexpectedMessage(ctx)
	}

	return nil
}

func (rp *ResourceProvider) forward(ctx *actor.Context, msg actor.Message) {
	if ctx.ExpectingResponse() {
		response := ctx.Ask(rp.resourceProvider, msg)
		ctx.Respond(response.Get())
	} else {
		ctx.Tell(rp.resourceProvider, msg)
	}
}
