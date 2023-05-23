package allocation

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/syncx/waitgroupx"
)

var (
	preemptionTimeoutDuration = time.Hour
	errNoPreemptionStatus     = errors.New("no preemption status available for unallocated task")
)

type (
	// Preemption represents the preemption status of an allocation. An alllocation is assumed to be
	// preempted exactly one time. The object is "nil safe" - it'll gracefully handle calls on a nil
	// preemption.
	Preemption struct {
		mu sync.Mutex
		wg waitgroupx.Group

		preempted bool
		acked     bool
		watchers  map[uuid.UUID]chan<- struct{}
	}

	// PreemptionWatcher contains a channel which can be polled for a preemption signal.
	PreemptionWatcher struct{ C <-chan struct{} }
)

// NewPreemption returns a new preemption struct.
func NewPreemption() *Preemption {
	return &Preemption{
		watchers: map[uuid.UUID]chan<- struct{}{},
		wg:       waitgroupx.WithContext(context.Background()),
	}
}

// Watch sets a watcher up to listen for preemption signals and returns it.
func (p *Preemption) Watch(id uuid.UUID) PreemptionWatcher {
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

	return PreemptionWatcher{C: w}
}

// Unwatch unregisters a preemption watcher.
func (p *Preemption) Unwatch(id uuid.UUID) {
	if p == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()

	delete(p.watchers, id)
}

// Preempt preempts all watchers and sets the allocation as preempted for all future.
func (p *Preemption) Preempt(timeout func(err error)) {
	if p == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.preempted {
		p.wg.Go(func(ctx context.Context) {
			t := time.NewTimer(preemptionTimeoutDuration)
			defer t.Stop()

			select {
			case <-t.C:
				timeout(ErrPreemptionTimeoutExceeded)
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
func (p *Preemption) Acknowledge() error {
	if p == nil {
		return errNoPreemptionStatus
	}
	p.mu.Lock()
	defer p.mu.Unlock()

	p.acked = true
	return nil
}

// Acknowledged returns if preemption has been acknowledged.
func (p *Preemption) Acknowledged() bool {
	if p == nil {
		return false
	}
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.acked
}

// Close closes the preemption object.
func (p *Preemption) Close() {
	if p == nil {
		return
	}

	p.mu.Lock()
	p.preempted = true
	for id, w := range p.watchers {
		w <- struct{}{}
		close(w)
		delete(p.watchers, id)
	}
	p.mu.Unlock()

	p.wg.Close()
}
