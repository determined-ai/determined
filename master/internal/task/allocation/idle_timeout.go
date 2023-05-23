package allocation

import (
	"context"
	"sync"
	"time"

	"github.com/determined-ai/determined/master/pkg/syncx/waitgroupx"

	log "github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/internal/sproto"

	"github.com/determined-ai/determined/master/internal/proxy"

	"github.com/determined-ai/determined/master/pkg/actor"
)

type (
	// IdleTimeoutWatcherTick is the incoming message that should be handled.
	IdleTimeoutWatcherTick struct{}
	// IdleWatcherNoteActivity notes the activity to delay idle timeout.
	IdleWatcherNoteActivity struct {
		LastActivity time.Time
	}
)

var idleTimeoutWatcherTickInterval = 5 * time.Second

// IdleTimeoutWatcher watches the proxy activity to handle a task actor idle timeout.
type IdleTimeoutWatcher struct {
	// System dependencies.
	log    *log.Entry
	system *actor.System

	// Configuration.
	name   string
	cfg    *sproto.IdleTimeoutConfig
	action func()

	// Mutable internal state.
	mu                   sync.Mutex
	wg                   waitgroupx.Group // TODO(mar): consistent pointer usage.
	lastExplicitActivity *time.Time
}

// NewIdleTimeoutWatcher creates a new idle timeout watcher.
func NewIdleTimeoutWatcher(
	name string,
	cfg *sproto.IdleTimeoutConfig,
	system *actor.System,
	action func(),
) *IdleTimeoutWatcher {
	iw := &IdleTimeoutWatcher{
		log:    log.WithField("component", "idle-watcher").WithField("id", name),
		system: system,

		name:   name,
		cfg:    cfg,
		action: action,

		wg: waitgroupx.WithContext(context.Background()),
	}

	iw.wg.Go(iw.run)

	return iw
}

func (iw *IdleTimeoutWatcher) RecordActivity(instant time.Time) {
	iw.mu.Lock()
	iw.lastExplicitActivity = &instant
	iw.mu.Unlock()
}

func (iw *IdleTimeoutWatcher) Close() {
	iw.wg.Close()
}

func (iw *IdleTimeoutWatcher) run(ctx context.Context) {
	t := time.NewTicker(idleTimeoutWatcherTickInterval)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			if iw.tick() {
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

func (iw *IdleTimeoutWatcher) tick() (done bool) {
	var lastActivity *time.Time
	if iw.cfg.UseProxyState {
		// TODO(mar): is this map send safe?
		services := iw.system.AskAt(actor.Addr("proxy"), proxy.GetSummary{}).Get().(map[string]proxy.Service)
		service, ok := services[iw.cfg.ServiceID]
		if !ok {
			return false
		}
		lastActivity = &service.LastRequested
	}

	if iw.cfg.UseRunnerState {
		iw.mu.Lock()
		if iw.lastExplicitActivity != nil && iw.lastExplicitActivity.After(*lastActivity) {
			*lastActivity = *iw.lastExplicitActivity
		}
		iw.mu.Unlock()
	}

	iw.log.WithFields(log.Fields{
		"lastActivity": lastActivity.Format(time.RFC3339),
		"timeout":      lastActivity.Add(iw.cfg.TimeoutDuration).Format(time.RFC3339),
	}).Debug("idle timeout watcher ticked")

	if time.Now().After(lastActivity.Add(iw.cfg.TimeoutDuration)) {
		iw.action()
		return true
	}
	return false
}
