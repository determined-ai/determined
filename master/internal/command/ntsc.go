package command

import (
	"fmt"

	"github.com/determined-ai/determined/master/pkg/actor"
)

// MessageNTSC sends a msg to all ntsc managers.
func MessageNTSC(system *actor.System, msg interface{}) actor.Responses {
	refs := []*actor.Ref{
		// system.Get(actor.Addr(CommandActorPath)),
		system.Get(actor.Addr(NotebookActorPath)),
		// system.Get(actor.Addr(TensorboardActorPath)),
		// system.Get(actor.Addr(ShellActorPath)),
	}

	for _, ref := range refs {
		if ref == nil {
			panic(fmt.Sprintf("ref is nil"))
		}
	}

	return system.AskAll(msg, refs...)
}
