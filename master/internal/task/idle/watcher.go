package idle

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/syncx/waitgroupx"

	log "github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/internal/sproto"

	"github.com/determined-ai/determined/master/internal/proxy"
)

var syslog = log.WithField("component", "idle-watcher")

// ErrIdle indicates that, according to the Watcher configuration, the service is idle.
var ErrIdle = fmt.Errorf("service is inactive")

// TickInterval is the interval at which to check the proxy activity.
var TickInterval = 5 * time.Second

// TimeoutFn is called when the service is idle.
type TimeoutFn func(context.Context, error)

// Watcher watches the proxy activity to handle a task actor idle timeout.
type Watcher struct {
	// System dependencies.
	syslog *log.Entry

	// Configuration.
	cfg    sproto.IdleTimeoutConfig
	action TimeoutFn

	// Mutable internal state.
	mu                   sync.Mutex
	wg                   waitgroupx.Group // TODO(mar): consistent pointer usage.
	lastExplicitActivity *time.Time
}

// New creates a new idle timeout watcher. The action can be triggered until Close is called.
func New(cfg sproto.IdleTimeoutConfig, action TimeoutFn) *Watcher {
	w := &Watcher{
		syslog: syslog.WithField("id", cfg.ServiceID),
		cfg:    cfg,
		action: action,
		wg:     waitgroupx.WithContext(context.Background()),
	}

	w.wg.Go(w.run)

	return w
}

// RecordActivity notes the activity to delay idle timeout.
func (w *Watcher) RecordActivity(instant time.Time) {
	w.mu.Lock()
	w.lastExplicitActivity = &instant
	w.mu.Unlock()
}

// Close closes the idle timeout watcher.
func (w *Watcher) Close() {
	w.wg.Close()
}

func (w *Watcher) run(ctx context.Context) {
	t := time.NewTicker(TickInterval)
	defer t.Stop()
	for {
		select {
		case <-t.C:
		case <-ctx.Done():
			return
		}

		if w.tick(ctx) {
			return
		}
	}
}

func (w *Watcher) tick(ctx context.Context) (done bool) {
	var lastActivity *time.Time
	if w.cfg.UseProxyState {
		service, ok := proxy.DefaultProxy.Summary(w.cfg.ServiceID)
		if !ok {
			return false
		}
		lastActivity = ptrs.Ptr(service.LastRequested)
	}

	if w.cfg.UseRunnerState {
		w.mu.Lock()
		if lastActivity == nil ||
			w.lastExplicitActivity != nil && w.lastExplicitActivity.After(*lastActivity) {
			lastActivity = w.lastExplicitActivity
		}
		w.mu.Unlock()
	}

	if lastActivity != nil {
		w.syslog.WithFields(log.Fields{
			"lastActivity": lastActivity.Format(time.RFC3339),
			"timeout":      lastActivity.Add(w.cfg.TimeoutDuration).Format(time.RFC3339),
		}).Debugf("idle timeout watcher ticked")
	} else {
		w.syslog.Debugf("idle timeout watcher ticked without activity")
	}

	if lastActivity == nil {
		return false
	}

	if time.Now().After(lastActivity.Add(w.cfg.TimeoutDuration)) {
		err := fmt.Errorf("idle for more than %s: %w", w.cfg.TimeoutDuration.Round(time.Second), ErrIdle)
		w.action(ctx, err)
		return true
	}
	return false
}
