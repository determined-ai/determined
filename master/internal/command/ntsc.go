package command

import (
	"fmt"

	"github.com/determined-ai/determined/master/pkg/actor"
)

// MessageNTSC sends a msg to all ntsc managers.
// CHECK: Tell(diff semantics?) or Ask
func MessageNTSC(system *actor.System, msg interface{}) actor.Responses {
	// CHAT: this could also accept an *actor.Context.
	refs := []*actor.Ref{
		// system.Get(actor.Addr(CommandActorPath)),
		system.Get(actor.Addr(NotebookActorPath)),
		// system.Get(actor.Addr(TensorboardActorPath)),
		// system.Get(actor.Addr(ShellActorPath)),
	}

	// CHECK: can we message "notebooks/*" ?
	// FIXME: what if some of the actors aren't up? timeout?
	for _, ref := range refs {
		if ref == nil {
			panic(fmt.Sprintf("ref is nil"))
		}
	}

	return system.AskAll(msg, refs...)
}
