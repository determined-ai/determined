package idler

import (
	"context"
	"sync"
	"time"

	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/syncx/waitgroupx"

	log "github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/internal/sproto"

	"github.com/determined-ai/determined/master/internal/proxy"
)

var syslog = log.WithField("component", "idle-watcher")

// TickInterval is the interval at which to check the proxy activity.
var TickInterval = 5 * time.Second

// Watcher watches the proxy activity to handle a task actor idle timeout.
type Watcher struct {
	// System dependencies.
	log *log.Entry

	// Configuration.
	cfg    *sproto.IdleTimeoutConfig
	action func()

	// Mutable internal state.
	mu                   sync.Mutex
	wg                   waitgroupx.Group // TODO(mar): consistent pointer usage.
	lastExplicitActivity *time.Time
}

// New creates a new idle timeout watcher. The action can be triggered until Close is called.
func New(cfg *sproto.IdleTimeoutConfig, action func()) *Watcher {
	w := &Watcher{
		log:    syslog.WithField("id", cfg.ServiceID),
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
			if w.tick() {
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

func (w *Watcher) tick() (done bool) {
	var lastActivity *time.Time
	if w.cfg.UseProxyState {
		service := proxy.DefaultProxy.GetService(w.cfg.ServiceID)
		if service == nil {
			return false
		}
		lastActivity = ptrs.Ptr(service.LastRequested)
	}

	if w.cfg.UseRunnerState {
		w.mu.Lock()
		if lastActivity == nil || w.lastExplicitActivity != nil && w.lastExplicitActivity.After(*lastActivity) {
			lastActivity = w.lastExplicitActivity
		}
		w.mu.Unlock()
	}

	w.log.WithFields(log.Fields{
		"lastActivity": lastActivity.Format(time.RFC3339),
		"timeout":      lastActivity.Add(w.cfg.TimeoutDuration).Format(time.RFC3339),
	}).Debugf("idle timeout watcher ticked")

	if lastActivity == nil {
		return false
	}

	if time.Now().After(lastActivity.Add(w.cfg.TimeoutDuration)) {
		w.action()
		return true
	}
	return false
}
