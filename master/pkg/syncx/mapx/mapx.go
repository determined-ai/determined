package mapx

import "sync"

// New creates an empty map.
func New[K comparable, V any]() Map[K, V] {
	return Map[K, V]{inner: make(map[K]V)}
}

// Map is a generic, thread-safe map to supersede usages of sync.Map.
type Map[K comparable, V any] struct {
	mu    sync.RWMutex
	inner map[K]V
}

// Load the value corresponding to k.
func (m *Map[K, V]) Load(k K) (V, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	v, ok := m.inner[k]
	return v, ok
}

// Delete the value corresponding to k, idempotently.
func (m *Map[K, V]) Delete(k K) (V, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	v, ok := m.inner[k]
	delete(m.inner, k)
	return v, ok
}

// Store the (k,v) pair.
func (m *Map[K, V]) Store(k K, v V) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.inner[k] = v
}

// Len of the map (number of stored pairs).
func (m *Map[K, V]) Len() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.inner)
}

// WithLock runs the given function on the underlying map with a write lock.
func (m *Map[K, V]) WithLock(f func(m map[K]V)) {
	m.mu.Lock()
	defer m.mu.Unlock()

	f(m.inner)
}

// Clear the map.
func (m *Map[K, V]) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for k := range m.inner {
		delete(m.inner, k)
	}
}

// Values returns the list of values in the map.
func (m *Map[K, V]) Values() []V {
	m.mu.RLock()
	defer m.mu.RUnlock()
	values := make([]V, 0, len(m.inner))
	for k := range m.inner {
		values = append(values, m.inner[k])
	}
	return values
}
