package preemptible

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/determined-ai/determined/master/pkg/syncx/waitgroupx"
)

// ErrPreemptionTimeoutExceeded indicates that an allocation not halt within the expected deadline.
var ErrPreemptionTimeoutExceeded = fmt.Errorf("allocation did not preempt in %s", DefaultTimeout)

// ErrPreemptionDisabled indicates that an alloction is either non-preemptible or not running.
var ErrPreemptionDisabled = fmt.Errorf("allocation is not preemptible")

// DefaultTimeout is the delay before the deadline exceeded callback passed to preempt is called.
var DefaultTimeout = time.Hour

// Watcher signals all gather completion via a channel which is closed upon said completion.
// TODO(DET-9565): Use of this watcher pattern here is unnecessary.
type Watcher struct{ C <-chan struct{} }

// Preemptible represents the preemption status of an allocation. An allocation is assumed to be
// preempted exactly one time. The object is "nil safe" - it'll gracefully handle calls on a nil
// preemption.
type Preemptible struct {
	mu sync.Mutex
	wg waitgroupx.Group

	preempted bool
	acked     bool
	watchers  map[uuid.UUID]chan<- struct{}
}

// New initializes a Preemption and returns it.
func New() *Preemptible {
	return &Preemptible{
		watchers: map[uuid.UUID]chan<- struct{}{},
		wg:       waitgroupx.WithContext(context.Background()),
	}
}

// Watch sets a watcher up to listen for preemption signals and returns it.
// TODO(DET-9565): Callers maintaining this ID is unnecessary.
func (p *Preemptible) Watch(id uuid.UUID) Watcher {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Size 1; at most a single message can be sent and we don't want to block.
	w := make(chan struct{}, 1)

	p.watchers[id] = w
	if p.preempted {
		w <- struct{}{}
		close(w)
		delete(p.watchers, id)
	}

	return Watcher{C: w}
}

// Unwatch unregisters a preemption watcher.
func (p *Preemptible) Unwatch(id uuid.UUID) {
	p.mu.Lock()
	defer p.mu.Unlock()

	delete(p.watchers, id)
}

// Preempt preempts all watchers, marks us as preempted and begins the preemption deadline,
// after which the timeout callback will be called. The preemption deadline callback can
// fire until Close is called.
func (p *Preemptible) Preempt(timeoutCallback func(err error)) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.preempted {
		p.wg.Go(func(ctx context.Context) {
			// don't acquire a lock in here without changing close to not lock while it waits.
			t := time.NewTimer(DefaultTimeout)
			defer t.Stop()

			select {
			case <-t.C:
				timeoutCallback(ErrPreemptionTimeoutExceeded)
			case <-ctx.Done():
			}
		})
	}

	p.preempted = true
	for id, w := range p.watchers {
		w <- struct{}{}
		close(w)
		delete(p.watchers, id)
	}
}

// Acknowledge acknowledges preemption.
func (p *Preemptible) Acknowledge() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.acked = true
}

// Acknowledged returns if preemption has been acknowledged.
func (p *Preemptible) Acknowledged() bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.acked
}

// Close cancels the preemption timeout callbacks if they haven't started and signals all watchers.
func (p *Preemptible) Close() {
	p.wg.Close()

	p.mu.Lock()
	defer p.mu.Unlock()

	p.preempted = true
	for id, w := range p.watchers {
		w <- struct{}{}
		close(w)
		delete(p.watchers, id)
	}
}
