package dispatcherrm

import (
	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

type dispatcherAgents struct {
	ref *actor.Ref
}

func newDispatcherAgents(ref *actor.Ref) *dispatcherAgents {
	return &dispatcherAgents{ref}
}

func (a *dispatcherAgents) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart:
		// do nothing
	case *apiv1.DisableAgentRequest, *apiv1.EnableAgentRequest:
		ctx.Respond(ctx.Ask(a.ref, msg).Get())
	default:
		ctx.Respond(errors.Errorf("dispatcher agents received an unexpected message %T", msg))
	}
	return nil
}
