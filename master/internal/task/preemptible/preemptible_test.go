package preemptible_test

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/master/internal/task/preemptible"
)

func TestPreemption(t *testing.T) {
	// "task" is allocated.
	p := preemptible.New()
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
	var timedOut atomic.Bool
	p.Preempt(func(err error) { timedOut.Store(true) })
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

	p.Acknowledge()
	require.True(t, p.Acknowledged())
	require.False(t, timedOut.Load())
}

func TestTimeout(t *testing.T) {
	preemptible.DefaultTimeout = time.Microsecond
	defer func() { preemptible.DefaultTimeout = time.Hour }()

	// "task" is allocated.
	p := preemptible.New()

	// watcher connects
	_ = p.Watch(uuid.New())

	// on preemption, it should receive status.
	var timedOut atomic.Bool
	p.Preempt(func(err error) { timedOut.Store(true) })

	waitForCondition(t, time.Second, timedOut.Load)

	p.Close()
}

func TestClose(t *testing.T) {
	// "task" is allocated.
	p := preemptible.New()

	// watcher connects
	id := uuid.New()
	w := p.Watch(id)

	// should not immediately receive initial status.
	select {
	case <-w.C:
		t.Fatal("received preemption but should not have")
	default:
	}

	p.Close()

	select {
	case <-w.C:
	default:
		t.Fatal("did not receive preemption")
	}
}

func waitForCondition(
	t *testing.T,
	timeout time.Duration,
	condition func() bool,
) {
	for i := 0; i < int(timeout/preemptible.DefaultTimeout); i++ {
		if condition() {
			return
		}
		time.Sleep(preemptible.DefaultTimeout)
	}
}
