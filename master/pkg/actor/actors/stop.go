package actors

import (
	"github.com/google/uuid"

	"github.com/determined-ai/determined/master/pkg/actor"
)

type stopNotifier struct {
	ref  *actor.Ref
	msg  actor.Message
	done chan struct{}
}

func (a *stopNotifier) Receive(context *actor.Context) error {
	switch context.Message().(type) {
	case actor.PreStart:
		go func() {
			defer close(a.done)
			a.awaitTermination(context)
		}()
	}
	return nil
}

func (a *stopNotifier) awaitTermination(context *actor.Context) {
	_ = a.ref.AwaitTermination()
	context.Ask(context.Self().Parent(), a.msg).Get()
}

// NotifyOnStop asynchronously notifies the context's recipient when the
// provided actor ref has stopped. Returns a channel that is closed when the
// recipient has been notified.
func NotifyOnStop(context *actor.Context, ref *actor.Ref, msg actor.Message) <-chan struct{} {
	done := make(chan struct{})
	context.ActorOf(
		"notify-stop-"+uuid.New().String(),
		&stopNotifier{
			done: done,
			ref:  ref,
			msg:  msg,
		},
	)

	return done
}
