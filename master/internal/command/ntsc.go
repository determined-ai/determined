package command

import (
	"fmt"

	"github.com/determined-ai/determined/master/pkg/actor"
)

func askChildren(ctx *actor.Context, msg interface{}) map[*actor.Ref]error {
	responses := ctx.AskAll(msg, ctx.Children()...).GetAll()
	issues := make(map[*actor.Ref]error, 0)
	// pick out the errors and report them
	for ref, resp := range responses {
		switch r := resp.(type) {
		case error:
			issues[ref] = r
		}
	}
	return issues
}

// MessageNTSC sends a msg to all ntsc managers.
func MessageNTSC(system *actor.System, msg interface{}) actor.Responses {
	refs := []*actor.Ref{
		system.Get(actor.Addr(CommandActorPath)),
		system.Get(actor.Addr(NotebookActorPath)),
		system.Get(actor.Addr(TensorboardActorPath)),
		system.Get(actor.Addr(ShellActorPath)),
	}

	for _, ref := range refs {
		if ref == nil {
			panic(fmt.Sprintf("ref is nil"))
		}
	}

	return system.AskAll(msg, refs...)
}
