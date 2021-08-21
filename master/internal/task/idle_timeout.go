package task

import (
	"time"

	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/actor/actors"
)

// IdleTimeoutWatcherTick is the incoming message that should be handled.
type IdleTimeoutWatcherTick struct{}

// IdleTimeoutWatcher watches the proxy activity to handle a task actor idle timeout.
type IdleTimeoutWatcher struct {
	TickInterval    time.Duration
	Timeout         time.Duration
	GetLastActivity func(ctx *actor.Context) *time.Time
	Action          func(ctx *actor.Context)
}

// PreStart should be called on task actor PreStart.
func (p *IdleTimeoutWatcher) PreStart(ctx *actor.Context) {
	actors.NotifyAfter(ctx, p.TickInterval, IdleTimeoutWatcherTick{})
}

// ReceiveMsg should be called on receiving related messages.
func (p *IdleTimeoutWatcher) ReceiveMsg(ctx *actor.Context) error {
	switch ctx.Message().(type) {
	case IdleTimeoutWatcherTick:
		lastActivity := p.GetLastActivity(ctx)
		if lastActivity == nil {
			return nil
		}

		if time.Now().After(lastActivity.Add(p.Timeout)) {
			p.Action(ctx)
			return nil
		}

		actors.NotifyAfter(ctx, p.TickInterval, IdleTimeoutWatcherTick{})

	default:
		return actor.ErrUnexpectedMessage(ctx)
	}

	return nil
}
