package actors

import (
	"time"

	"github.com/google/uuid"

	"github.com/determined-ai/determined/master/pkg/actor"
)

type timer struct {
	*time.Timer

	recipient *actor.Ref
	msg       actor.Message
}

// Receive implements the actor.Actor interface.
func (t *timer) Receive(ctx *actor.Context) error {
	switch ctx.Message().(type) {
	case actor.PreStart:
		go t.awaitTimer(ctx)
	case actor.PostStop:
		t.Stop()
	}
	return nil
}

func (t *timer) awaitTimer(ctx *actor.Context) {
	<-t.C // Wait for the timer to tick.
	ctx.Tell(t.recipient, t.msg)
	ctx.Self().Stop()
}

// NotifyAfter asynchronously notifies the context's recipient with the provided message when
// after the provided duration.
func NotifyAfter(ctx *actor.Context, d time.Duration, msg actor.Message) (*actor.Ref, bool) {
	addr := actor.Addr("notify-timer-" + uuid.New().String())
	return ctx.Self().System().ActorOf(addr,
		&timer{Timer: time.NewTimer(d), recipient: ctx.Self(), msg: msg})
}
