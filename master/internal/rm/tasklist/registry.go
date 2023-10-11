package tasklist

import (
	"fmt"
	"sync"
)

// Registry is a thread-safe map of key value pairs that supports callbacks on delete.
type Registry[K comparable, V any] struct {
	mu   sync.Mutex
	data map[K]entry[V]
}

type entry[V any] struct {
	value V
	done  chan bool
}

// NewRegistry creates a new Registry.
func NewRegistry[K comparable, V any]() *Registry[K, V] {
	return &Registry[K, V]{
		data: make(map[K]entry[V]),
	}
}

// Load returns the value stored for the given key and whether the key was found.
func (r *Registry[K, V]) Load(key K) (V, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	e, ok := r.data[key]
	return e.value, ok
}

// Add adds the given key value pair to the registry. If the key already exists, an error is
// returned.
func (r *Registry[K, V]) Add(key K, value V) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.data[key]; ok {
		return fmt.Errorf("key %v already exists", key)
	}
	r.data[key] = entry[V]{
		value: value,
		done:  make(chan bool),
	}
	return nil
}

// Delete deletes the given key from the registry. If the key does not exist, an error is returned.
func (r *Registry[K, V]) Delete(key K) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	e, ok := r.data[key]
	if !ok {
		return fmt.Errorf("key %v does not exist", key)
	}
	close(e.done)
	delete(r.data, key)
	return nil
}

// OnDelete registers a callback to be called when the given key is deleted. If the key does not
// exist, the callback is called async.
func (r *Registry[K, V]) OnDelete(key K, callback func()) {
	r.mu.Lock()
	defer r.mu.Unlock()

	e, ok := r.data[key]
	go func() {
		if ok {
			<-e.done
		}
		callback()
	}()
}
