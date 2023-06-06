package preemptible

import (
	"context"

	"github.com/determined-ai/determined/master/pkg/syncx/mapx"
)

var preemptibles = mapx.New[string, *Preemptible]()

// Register a preempitble to default service.
// ID must be a globally unique identifier for the preemptible.
func Register(id string) {
	preemptibles.Store(id, New())
}

// Unregister removes a preemptible from the service.
// ID must be a globally unique identifier for the preemptible.
func Unregister(id string) {
	p, ok := preemptibles.Delete(id)
	if !ok {
		return
	}
	p.Close()
}

// Watch blocks until preemption or the context is canceled. Exits not indicative
// preemption return a non-nil error.
// ID must be a globally unique identifier for the preemptible.
func Watch(ctx context.Context, id string) error {
	p, ok := preemptibles.Load(id)
	if !ok {
		return ErrPreemptionDisabled
	}
	return p.Watch(ctx)
}

// Acknowledge the receipt of a preemption signal.
// ID must be a globally unique identifier for the preemptible.
func Acknowledge(id string) {
	p, ok := preemptibles.Load(id)
	if !ok {
		return
	}
	p.Acknowledge()
}

// Acknowledged returns whether a preemption signal has been acknowledged.
// ID must be a globally unique identifier for the preemptible.
func Acknowledged(id string) bool {
	p, ok := preemptibles.Load(id)
	if !ok {
		return false
	}
	return p.Acknowledged()
}

// Preempt preempts all watchers, marks us as preempted and begins the preemption deadline.
// The preemption deadline callback can fire until Close is called.
// ID must be a globally unique identifier for the preemptible.
func Preempt(id string, timeoutCallback func(error)) {
	p, ok := preemptibles.Load(id)
	if !ok {
		return
	}
	p.Preempt(timeoutCallback)
}
