package sproto

import (
	"container/list"
	"sync"
)

type AllocateResponse struct {
	Resources *ResourcesAllocated
	Error     *ResourcesFailure
}

type Watcher[T any] struct {
	mu     sync.Mutex
	cond   *sync.Cond
	inbox  *list.List
	closed bool

	C <-chan T
}

func NewWatcher[T any]() *Watcher[T] {
	c := make(chan T)
	w := &Watcher[T]{C: c}
	go w.run(c)
	return w
}

func (w *Watcher[T]) Send(res T) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.inbox.PushBack(res)
	w.cond.Signal()
}

// TODO(mar): watcher errs.

func (w *Watcher[T]) Close() {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.closed = true
	w.cond.Signal()
}

func (w *Watcher[T]) run(c chan<- T) {
	w.mu.Lock()
	defer w.mu.Unlock()

	for {
		for w.empty() && !w.closed {
			w.cond.Wait()
		}

		if w.closed {
			close(c)
			return
		}

		if !w.empty() {
			next := w.inbox.Remove(w.inbox.Front())
			c <- next.(T)
		}
	}
}

func (w *Watcher[T]) empty() bool {
	return w.inbox.Len() == 0
}

func MergeWatchers[T any](ws ...*Watcher[T]) *Watcher[T] {
	out := NewWatcher[T]()

	var wg sync.WaitGroup
	for _, in := range ws {
		in := in

		wg.Add(1)
		go func() {
			defer wg.Done()
			for t := range in.C {
				out.Send(t)
			}
		}()
	}

	go func() {
		wg.Wait()
		out.Close()
	}()

	return out
}
