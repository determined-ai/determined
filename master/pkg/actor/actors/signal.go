package actors

import (
	"os"
	"os/signal"

	"github.com/google/uuid"

	"github.com/determined-ai/determined/master/pkg/actor"
)

type signalActor struct {
	listener chan os.Signal
}

func (s *signalActor) Receive(ctx *actor.Context) error {
	switch ctx.Message().(type) {
	case actor.PreStart:
		go func() {
			for sig := range s.listener {
				ctx.Tell(ctx.Self().Parent(), sig)
			}
		}()
	case actor.PostStop:
		signal.Stop(s.listener)
		close(s.listener)
	}
	return nil
}

// NotifyOnSignal relays incoming signals to the caller.  If no signals are provided, all incoming
// signals will be relayed. Otherwise, just the provided signals will.
func NotifyOnSignal(ctx *actor.Context, signals ...os.Signal) {
	listener := make(chan os.Signal, 100)
	signal.Notify(listener, signals...)
	ctx.ActorOf("notify-on-signal-"+uuid.New().String(), &signalActor{listener: listener})
}
