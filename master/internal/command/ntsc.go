package command

import "github.com/determined-ai/determined/master/pkg/actor"

func MessageNTSC(system *actor.System, msg interface{}) actor.Responses {
	// CHAT: this could also accept an *actor.Context.
	refs := []*actor.Ref{
		system.Get(actor.Addr(CommandActorPath)),
		system.Get(actor.Addr(NotebookActorPath)),
		system.Get(actor.Addr(TensorboardActorPath)),
		system.Get(actor.Addr(ShellActorPath)),
	}

	// filter out nil refs
	refs = refs[:0]
	for _, ref := range refs {
		if ref != nil {
			refs = append(refs, ref)
		}
	}

	// FIXME: what if some of the actors aren't up? timeout?
	return system.AskAll(msg, refs...)
}
