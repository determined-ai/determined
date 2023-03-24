package command

import (
	"fmt"

	"github.com/determined-ai/determined/master/pkg/actor"
)

// TellNTSC sends a msg to all ntsc managers.
func TellNTSC(system *actor.System, msg interface{}) {
	ntscAddresses := []actor.Address{
		actor.Addr(CommandActorPath),
		actor.Addr(NotebookActorPath),
		actor.Addr(TensorboardActorPath),
		actor.Addr(ShellActorPath),
	}

	for _, addr := range ntscAddresses {
		ref := system.Get(addr)
		if ref == nil {
			panic(fmt.Sprintf("unable to find actor at %s", addr))
		}
		system.Tell(ref, msg)
	}
}
