package task

import (
	"time"

	"github.com/determined-ai/determined/master/internal/proxy"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/actor/actors"
	"github.com/determined-ai/determined/master/pkg/model"
)

const tickInterval = 5 * time.Second

// ProxyIdleTimeoutWatcherTick is the incoming message that should be handled.
type ProxyIdleTimeoutWatcherTick struct{}

// ProxyIdleTimeoutWatcher watches the proxy activity to handle a task actor idle timeout.
type ProxyIdleTimeoutWatcher struct {
	proxyRef *actor.Ref

	TaskID      string
	Description string
	Timeout     model.Duration
	KillMessage actor.Message
}

// PreStart should be called on task actor PreStart.
func (p *ProxyIdleTimeoutWatcher) PreStart(ctx *actor.Context) {
	p.proxyRef = ctx.Self().System().Get(actor.Addr("proxy"))

	actors.NotifyAfter(ctx, tickInterval, ProxyIdleTimeoutWatcherTick{})
}

// ReceiveMsg should be called on receiving related messages.
func (p *ProxyIdleTimeoutWatcher) ReceiveMsg(ctx *actor.Context) error {
	switch ctx.Message().(type) {
	case ProxyIdleTimeoutWatcherTick:
		services := ctx.Ask(p.proxyRef, proxy.GetSummary{}).Get().(map[string]proxy.Service)

		service, ok := services[p.TaskID]
		if !ok {
			return nil
		}

		if time.Now().After(service.LastRequested.Add(time.Duration(p.Timeout))) {
			ctx.Log().Infof("killing %s due to inactivity", p.Description)
			ctx.Ask(ctx.Self(), p.KillMessage)
		}

		actors.NotifyAfter(ctx, tickInterval, ProxyIdleTimeoutWatcherTick{})

	default:
		return actor.ErrUnexpectedMessage(ctx)
	}

	return nil
}
