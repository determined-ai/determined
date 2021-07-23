package internal

import (
	"testing"

	"github.com/google/uuid"
	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/pkg/model"
)

func TestPreemption(t *testing.T) {
	// Initialize a nil preemption.
	var p preemption

	// Watch nil should not panic and return an error.
	id := uuid.New()
	_, err := p.watch(model.NewAllocationID(uuid.New().String()), id)
	assert.ErrorContains(t, err, "stale task")

	// All method on nil should not panic.
	p.unwatch(id)
	p.preempt()
	p.close()

	// "task" is allocated.
	t1 := model.NewAllocationID(uuid.New().String())
	p = newPreemption(t1)

	// real watcher connects
	id = uuid.New()
	w, err := p.watch(t1, id)
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
	p.unwatch(id)

	// new post-preemption watch connects
	id = uuid.New()
	w, err = p.watch(t1, id)
	assert.NilError(t, err)

	// should immediately receive initial status and initial status should be preemption.
	select {
	case <-w.C:
	default:
		t.Fatal("preemptionWatcher.C was empty channel (should come with initial status when preempted)")
	}

	// preempted preemption unwatching should work.
	p.unwatch(id)
}
