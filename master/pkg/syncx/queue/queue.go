package queue

import (
	"context"
	"sync"
)

// Queue is a thread-safe queue.
type Queue[T any] struct {
	mu    sync.Mutex
	cond  *sync.Cond // used to wait for elements in the queue
	elems []T
}

// New creates a new queue.
func New[T any]() *Queue[T] {
	q := &Queue[T]{elems: make([]T, 0)}
	q.cond = sync.NewCond(&q.mu)
	return q
}

// Put adds an element to the queue.
func (q *Queue[T]) Put(t T) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.empty() {
		q.cond.Broadcast()
	}
	q.elems = append(q.elems, t)
}

// Get removes and returns an element from the queue. If the queue is empty, then Get will block
// until an element is available.
func (q *Queue[T]) Get() T {
	q.mu.Lock()
	defer q.mu.Unlock()

	for q.empty() {
		q.cond.Wait()
	}
	res := q.elems[0]
	q.elems = q.elems[1:]
	return res
}

// GetWithContext removes and returns an element from the queue. If the queue is empty, then Get will block
// until an element is available or the context is canceled. This routine is relatively expensive---don't use
// it for a high throughput queue. Instead, you should probably just use a channel.
func (q *Queue[T]) GetWithContext(ctx context.Context) (T, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-ctx.Done():
			q.cond.Broadcast()
		case <-done:
		}
	}()

	for q.empty() && ctx.Err() == nil {
		q.cond.Wait()
	}
	if ctx.Err() != nil {
		var t T
		return t, ctx.Err()
	}
	res := q.elems[0]
	q.elems = q.elems[1:]
	return res, nil
}

// Len returns the number of elements in the queue.
func (q *Queue[T]) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()

	return len(q.elems)
}

func (q *Queue[T]) empty() bool {
	return len(q.elems) == 0
}
