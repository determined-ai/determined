package internal

import (
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
)

const clusterCrashMessage = `Detected a resource manager crashed. Please manually restart the
Determined master. It is recommended you check that stray resources are cleaned up properly.`

// root manages the lifecycle of all actors in the Determined master and
// defines a supervision strategy specifically for the master.
func root(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart, actor.PostStop:
		return nil
	case actor.ChildFailed:
		switch msg.Child.Address() {
		case sproto.K8sRMAddr, sproto.AgentRMAddr:
			ctx.Log().WithField("crash", msg).Errorf(clusterCrashMessage)
		case sproto.PodsAddr, sproto.AgentsAddr:
			ctx.Log().WithField("crash", msg).Errorf(clusterCrashMessage)
		}
		return nil
	case actor.ChildStopped:
		switch msg.Child.Address() {
		case sproto.K8sRMAddr, sproto.AgentRMAddr:
			ctx.Log().WithField("crash", msg).Errorf(clusterCrashMessage)
		case sproto.PodsAddr, sproto.AgentsAddr:
			ctx.Log().WithField("crash", msg).Errorf(clusterCrashMessage)
		}
		return nil
	}
	ctx.Log().Warnf("unexpected message sent to root actor: %v", ctx.Message())
	return nil
}
