package preemptible_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/master/internal/task/preemptible"
)

func TestPreemption(t *testing.T) {
	// "task" is allocated.
	p := preemptible.New()
	defer p.Close()

	// watcher connects
	errs := make(chan error, 1)
	go func() {
		errs <- p.Watch(context.Background())
		close(errs)
	}()

	// should not immediately receive initial status.
	select {
	case <-errs:
		t.Fatal("received preemption but should not have")
	default:
	}

	// on preemption, it should receive status.
	var timedOut atomic.Bool
	p.Preempt(func(err error) { timedOut.Store(true) })
	select {
	case <-errs:
	case <-time.After(time.Second):
		t.Fatal("did not receive preemption")
	}

	// new post-preemption watch connects
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err := p.Watch(ctx)
	if err != nil {
		t.Fatal("watch on preempted object was not instant")
	}

	p.Acknowledge()
	require.True(t, p.Acknowledged())
	require.False(t, timedOut.Load())
}

func TestTimeout(t *testing.T) {
	preemptible.DefaultTimeout = time.Microsecond
	defer func() { preemptible.DefaultTimeout = time.Hour }()

	p := preemptible.New()

	var timedOut atomic.Bool
	p.Preempt(func(err error) { timedOut.Store(true) })

	waitForCondition(t, time.Second, timedOut.Load)

	p.Close()
}

func TestClose(t *testing.T) {
	// "task" is allocated.
	p := preemptible.New()

	// watcher connects
	errs := make(chan error, 1)
	go func() {
		errs <- p.Watch(context.Background())
		close(errs)
	}()

	// should not immediately receive initial status.
	select {
	case <-errs:
		t.Fatal("received preemption but should not have")
	default:
	}

	p.Close()

	select {
	case <-errs:
	case <-time.After(time.Second):
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
