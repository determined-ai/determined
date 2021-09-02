package internal

import (
	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
)

// root manages the lifecycle of all actors in the Determined master and
// defines a supervision strategy specifically for the master.
func root(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart, actor.PostStop:
		return nil
	case actor.ChildFailed:
		switch msg.Child.Address() {
		case sproto.K8sRMAddr, sproto.AgentRMAddr:
			return errors.Wrapf(msg.Error, "resource mananger crashed, exiting")
		case sproto.PodsAddr, sproto.AgentsAddr:
			return errors.New("pods or agents manager crashed, exiting")
		}
		return nil
	case actor.ChildStopped:
		switch msg.Child.Address() {
		case sproto.K8sRMAddr, sproto.AgentRMAddr:
			return errors.New("resource manager stopped, exiting")
		case sproto.PodsAddr, sproto.AgentsAddr:
			return errors.New("pods or agents manager stopped, exiting")
		}
		return nil
	}
	ctx.Log().Warnf("unexpected message sent to root actor (%T): %v",
		ctx.Message(), ctx.Message())
	return nil
}
