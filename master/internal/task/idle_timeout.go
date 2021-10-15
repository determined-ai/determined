package task

import (
	"time"

	"github.com/determined-ai/determined/master/internal/sproto"

	"github.com/determined-ai/determined/master/internal/proxy"
	"github.com/determined-ai/determined/master/pkg/ptrs"

	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/actor/actors"
)

type (
	// IdleTimeoutWatcherTick is the incoming message that should be handled.
	IdleTimeoutWatcherTick struct{}
	// IdleWatcherNoteActivity notes the activity to delay idle timeout.
	IdleWatcherNoteActivity struct {
		LastActivity time.Time
	}
)

// IdleTimeoutWatcher watches the proxy activity to handle a task actor idle timeout.
type IdleTimeoutWatcher struct {
	TickInterval   time.Duration
	Timeout        time.Duration
	ServiceID      string
	UseProxy       bool
	UseRunnerState bool
	Action         func(ctx *actor.Context)

	lastExplicitActivity *time.Time
}

// NewIdleTimeoutWatcher creates a new idle timeout watcher.
func NewIdleTimeoutWatcher(name string, cfg *sproto.IdleTimeoutConfig) *IdleTimeoutWatcher {
	return &IdleTimeoutWatcher{
		TickInterval:   5 * time.Second,
		Timeout:        cfg.TimeoutDuration,
		UseProxy:       cfg.UseProxyState,
		UseRunnerState: cfg.UseRunnerState,
		ServiceID:      cfg.ServiceID,
		Action: func(ctx *actor.Context) {
			ctx.Log().Infof("killing %s due to inactivity", name)
			ctx.Tell(ctx.Self(), Terminate)
		},
	}
}

// PreStart should be called on task actor PreStart.
func (p *IdleTimeoutWatcher) PreStart(ctx *actor.Context) {
	actors.NotifyAfter(ctx, p.TickInterval, IdleTimeoutWatcherTick{})
}

// ReceiveMsg should be called on receiving related messages.
func (p *IdleTimeoutWatcher) ReceiveMsg(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case IdleTimeoutWatcherTick:
		var lastActivity *time.Time
		if p.UseProxy {
			proxyRef := ctx.Self().System().Get(actor.Addr("proxy"))
			services := ctx.Ask(proxyRef, proxy.GetSummary{}).Get().(map[string]proxy.Service)
			service, ok := services[p.ServiceID]
			if !ok {
				return nil
			}
			lastActivity = ptrs.TimePtr(service.LastRequested)
		}

		if p.UseRunnerState {
			if lastActivity == nil ||
				p.lastExplicitActivity != nil && p.lastExplicitActivity.After(*lastActivity) {
				lastActivity = p.lastExplicitActivity
			}
		}

		if lastActivity == nil {
			actors.NotifyAfter(ctx, p.TickInterval, IdleTimeoutWatcherTick{})
			return nil
		}

		if time.Now().After(lastActivity.Add(p.Timeout)) {
			p.Action(ctx)
			return nil
		}

		actors.NotifyAfter(ctx, p.TickInterval, IdleTimeoutWatcherTick{})

	case IdleWatcherNoteActivity:
		p.lastExplicitActivity = ptrs.TimePtr(msg.LastActivity)

	default:
		return actor.ErrUnexpectedMessage(ctx)
	}

	return nil
}
