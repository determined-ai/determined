package preemptible

import (
	"github.com/google/uuid"

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

// Watch sets a watcher up to listen for preemption signals and returns it.
// ID must be a globally unique identifier for the preemptible.
func Watch(id string, wID uuid.UUID) (Watcher, error) {
	p, ok := preemptibles.Load(id)
	if !ok {
		return Watcher{}, ErrPreemptionDisabled
	}
	return p.Watch(wID), nil
}

// Unwatch removes a preemption watcher.
// ID must be a globally unique identifier for the preemptible.
func Unwatch(id string, wID uuid.UUID) {
	p, ok := preemptibles.Load(id)
	if !ok {
		return
	}
	p.Unwatch(wID)
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
func Preempt(id string, timeoutCallback TimeoutFn) {
	p, ok := preemptibles.Load(id)
	if !ok {
		return
	}
	p.Preempt(timeoutCallback)
}
