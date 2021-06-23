package internal

import (
	"testing"

	"github.com/google/uuid"
	"gotest.tools/assert"
)

// TODO(XXX): Write new rendezvous tests. Old ones were related to websockets.

func TestPreemption(t *testing.T) {
	// Initialize a nil preemption.
	var p *preemption

	// Watch nil should not panic and return an error.
	id := uuid.New()
	_, err := p.watch(watchPreemption{id: id})
	assert.Error(t, err, "no preemption status available nil preemption")

	// All method on nil should not panic.
	p.unwatch(unwatchPreemption{id: id})
	p.preempt()
	p.close()

	// "task" is allocated.
	p = newPreemption()

	// real watcher connects
	id = uuid.New()
	w, err := p.watch(watchPreemption{id: id})
	assert.NilError(t, err)

	// should immediately receive initial status.
	select {
	case <-w.C:
		t.Fatal("received preemption but should not have")
	default:
	}

	// on preemption, it should also receive status.
	p.preempt()

	// should receive updated preemption status.
	select {
	case <-w.C:
	default:
		t.Fatal("did not receive preemption")
	}

	// preempted preemption unwatching should work.
	p.unwatch(unwatchPreemption{id})

	// new post-preemption watch connects
	id = uuid.New()
	w, err = p.watch(watchPreemption{id: id})
	assert.NilError(t, err)

	// should immediately receive initial status and initial status should be preemption.
	select {
	case <-w.C:
	default:
		t.Fatal("preemptionWatcher.C was empty channel (should come with initial status when preempted)")
	}

	// preempted preemption unwatching should work.
	p.unwatch(unwatchPreemption{id})
}
