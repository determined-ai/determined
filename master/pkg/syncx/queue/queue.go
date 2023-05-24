package queue

import (
	"sync"
)

// Opt is a functional option for Queue.
type Opt[T any] (func(*Queue[T]))

// WithMaxSize sets the maximum size of the queue. If the queue is full, then Put will block until
// space is available.
func WithMaxSize[T any](maxSize int) Opt[T] {
	return func(q *Queue[T]) {
		q.maxSize = maxSize
		q.elems = make([]T, 0, maxSize)
	}
}

// Queue is a thread-safe queue.
type Queue[T any] struct {
	maxSize int // 0 means no limit
	mu      *sync.Mutex
	putCond *sync.Cond // used to wait for space in the queue
	getCond *sync.Cond // used to wait for elements in the queue
	elems   []T
}

// New creates a new queue.
func New[T any](opts ...Opt[T]) *Queue[T] {
	var mu sync.Mutex
	q := &Queue[T]{
		mu:      &mu,
		putCond: sync.NewCond(&mu),
		getCond: sync.NewCond(&mu),
		elems:   make([]T, 0),
	}

	for _, opt := range opts {
		opt(q)
	}
	return q
}

// Put adds an element to the queue. If the queue was configured with a maximum size, then Put will
// block until space is available.
func (q *Queue[T]) Put(t T) {
	q.mu.Lock()
	defer q.mu.Unlock()

	for q.full() {
		q.putCond.Wait()
	}
	if q.empty() {
		q.getCond.Broadcast()
	}
	q.elems = append(q.elems, t)
}

// Get removes and returns an element from the queue. If the queue is empty, then Get will block
// until an element is available.
func (q *Queue[T]) Get() T {
	q.mu.Lock()
	defer q.mu.Unlock()

	for q.empty() {
		q.getCond.Wait()
	}
	if q.full() {
		q.putCond.Broadcast()
	}
	res := q.elems[0]
	q.elems = q.elems[1:]
	return res
}

// TryGet removes and returns an element from the queue. If the queue is empty, then TryGet will
// return false.
func (q *Queue[T]) TryGet() (T, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.empty() {
		var t T
		return t, false
	}
	if q.full() {
		q.putCond.Broadcast()
	}
	res := q.elems[0]
	q.elems = q.elems[1:]
	return res, true
}

// Len returns the number of elements in the queue.
func (q *Queue[T]) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()

	return len(q.elems)
}

func (q Queue[T]) empty() bool {
	return len(q.elems) == 0
}

func (q *Queue[T]) full() bool {
	return q.maxSize > 0 && len(q.elems) == q.maxSize
}
