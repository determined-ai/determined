package preemptible

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/determined-ai/determined/master/pkg/syncx/waitgroupx"
)

// ErrPreemptionTimeoutExceeded indicates that an allocation not halt within the expected deadline.
var ErrPreemptionTimeoutExceeded = fmt.Errorf("allocation did not preempt in %s", DefaultTimeout)

// ErrPreemptionDisabled indicates that an alloction is either non-preemptible or not running.
var ErrPreemptionDisabled = fmt.Errorf("allocation is not preemptible")

// DefaultTimeout is the delay before the deadline exceeded callback passed to preempt is called.
var DefaultTimeout = time.Hour

// Preemptible represents the preemption status of an allocation. An allocation is assumed to be
// preempted exactly one time. The object is "nil safe" - it'll gracefully handle calls on a nil
// preemption.
type Preemptible struct {
	mu        sync.Mutex
	wg        waitgroupx.Group
	preempted chan struct{}
	acked     bool
}

// New initializes a Preemption and returns it.
func New() *Preemptible {
	return &Preemptible{
		preempted: make(chan struct{}),
		wg:        waitgroupx.WithContext(context.Background()),
	}
}

// Watch blocks until preemption or the context is canceled. Exits not indication
// preemption return a non-nil error.
func (p *Preemptible) Watch(ctx context.Context) error {
	select {
	case <-p.preempted:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Preempt preempts all watchers, marks us as preempted and begins the preemption deadline,
// after which the timeout callback will be called. The preemption deadline callback can
// fire until Close is called.
func (p *Preemptible) Preempt(timeoutCallback func(err error)) {
	p.mu.Lock()
	defer p.mu.Unlock()

	select {
	case <-p.preempted:
	default:
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
		close(p.preempted)
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

	select {
	case <-p.preempted:
	default:
		close(p.preempted)
	}
}
