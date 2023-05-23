package allocation

import (
	"testing"

	"github.com/google/uuid"
)

func TestPreemptionNil(t *testing.T) {
	// Initialize a nil preemption.
	var p *Preemption

	// Watch nil should not panic and return an error.
	id := uuid.New()
	_ = p.Watch(id)

	// All method on nil should not panic.
	p.Unwatch(id)
	p.Preempt(func(error) {})
	p.Close()
}

func TestPreemption(t *testing.T) {
	// "task" is allocated.
	preemptionTimeoutDuration = 0
	p := NewPreemption()
	defer p.Close()

	// watcher connects
	id := uuid.New()
	w := p.Watch(id)

	// should not immediately receive initial status.
	select {
	case <-w.C:
		t.Fatal("received preemption but should not have")
	default:
	}

	// on preemption, it should receive status.
	p.Preempt(func(err error) {})
	select {
	case <-w.C:
	default:
		t.Fatal("did not receive preemption")
	}

	// unwatching preemption should do no harm.
	p.Unwatch(id)

	// new post-preemption watch connects
	id = uuid.New()
	w = p.Watch(id)

	// should immediately receive initial status and initial status should be preemption.
	select {
	case <-w.C:
	default:
		t.Fatal("PreemptionWatcher.C was empty channel (should come with initial status when preempted)")
	}

	// again, unwatching preemption should do no harm.
	p.Unwatch(id)
}
