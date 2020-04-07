package actor

// rootActor manages the lifecycle of all actors in a system. Its primary duty is to be the
// top-level parent; any error bubbled up to the root actor will be handled by the configured
// supervision strategy.
type rootActor struct{}

// Receive implements the actor.Actor interface.
func (a rootActor) Receive(context *Context) error {
	switch context.Message().(type) {
	case PreStart, ChildFailed, ChildStopped, PostStop:
		return nil
	}
	context.Log().Warnf("unexpected message sent to root actor (%T): %v",
		context.Message(), context.Message())
	return nil
}
