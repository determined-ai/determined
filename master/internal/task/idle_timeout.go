package task

import (
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"

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
	sproto.IdleTimeoutConfig
	TickInterval time.Duration
	Action       func(ctx *actor.Context)

	lastExplicitActivity *time.Time
}

// NewIdleTimeoutWatcher creates a new idle timeout watcher.
func NewIdleTimeoutWatcher(name string, cfg *sproto.IdleTimeoutConfig) *IdleTimeoutWatcher {
	return &IdleTimeoutWatcher{
		TickInterval:      5 * time.Second,
		IdleTimeoutConfig: *cfg,
		Action: func(ctx *actor.Context) {
			ctx.Log().Infof("killing %s due to inactivity", name)
			ctx.Tell(ctx.Self(),
				sproto.AllocationSignalWithReason{
					AllocationSignal: sproto.TerminateAllocation,
					InformationalReason: fmt.Sprintf(
						"inactivity for more than %s",
						cfg.TimeoutDuration.Round(time.Second)),
				})
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
		if p.UseProxyState {
			service := proxy.DefaultProxy.GetService(p.ServiceID)
			if service == nil {
				return nil
			}
			lastActivity = ptrs.Ptr(service.LastRequested)
		}

		if p.UseRunnerState {
			if lastActivity == nil ||
				p.lastExplicitActivity != nil && p.lastExplicitActivity.After(*lastActivity) {
				lastActivity = p.lastExplicitActivity
			}
		}

		if p.Debug {
			ctx.Log().WithFields(log.Fields{
				"lastActivity": lastActivity.Format(time.RFC3339),
				"timeout":      lastActivity.Add(p.TimeoutDuration).Format(time.RFC3339),
			}).Infof("idle timeout watcher ticked")
		}

		if lastActivity == nil {
			actors.NotifyAfter(ctx, p.TickInterval, IdleTimeoutWatcherTick{})
			return nil
		}

		if time.Now().After(lastActivity.Add(p.TimeoutDuration)) {
			p.Action(ctx)
			return nil
		}

		actors.NotifyAfter(ctx, p.TickInterval, IdleTimeoutWatcherTick{})

	case IdleWatcherNoteActivity:
		p.lastExplicitActivity = ptrs.Ptr(msg.LastActivity)

	default:
		return actor.ErrUnexpectedMessage(ctx)
	}

	return nil
}
