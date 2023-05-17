// Package orderedmapx is a thread safe and ordered implementation of a standard
// map. The elements can be retrieved in the same order in which they were
// added. orderedmapx allows elements to be located without having to
// sequentially search the list to find the element we're looking for.
package orderedmapx

import (
	"container/list"
	"sync"
)

type mapElement[K comparable, V any] struct {
	key   K
	value V
}

// Map is a thread safe and ordered implementation of standard map.
// K is the type of key and V is the type of value.
type Map[K comparable, V any] struct {
	mp   map[K]*list.Element
	mu   sync.RWMutex
	dll  *list.List
	cond *sync.Cond
}

// New returns an initialized Map[K, V].
func New[K comparable, V any]() *Map[K, V] {
	m := new(Map[K, V])
	m.mp = make(map[K]*list.Element)
	m.dll = list.New()
	m.cond = sync.NewCond(&m.mu)
	return m
}

// Put adds a key to the map with the specified value.  If the key already
// exists, then the existing key's value is replaced with the new value.
func (m *Map[K, V]) Put(key K, val V) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if e, ok := m.mp[key]; ok {
		// Replace existing key's value with the new value.
		e.Value = mapElement[K, V]{key: key, value: val}
		return
	}

	// Add the new key and its value.
	m.mp[key] = m.dll.PushFront(mapElement[K, V]{key: key, value: val})
	m.cond.Signal()
}

// PutIfAbsent returns the existing value if the key already exists in the map
// and sets the second return value to false to indicate that the value was not
// updated because the key already existed in the map. Otherwise, if the key
// did not already exist in the map, it returns the new value and sets the
// second return value to true to indicate that the key and its value were
// added to the map.
func (m *Map[K, V]) PutIfAbsent(key K, value V) (V, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if e, ok := m.mp[key]; ok {
		me := e.Value.(mapElement[K, V])
		return me.value, false
	}

	m.mp[key] = m.dll.PushFront(mapElement[K, V]{key: key, value: value})
	m.cond.Signal()

	return value, true
}

// Delete deletes the entry for the specified key from the map. Returns true
// if the key existed in the map and was deleted; false otherwise.
func (m *Map[K, V]) Delete(key K) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	e, ok := m.mp[key]
	if !ok {
		return false
	}

	m.dll.Remove(e)
	delete(m.mp, key)
	return true
}

// Length will return the length of Map.
func (m *Map[k, V]) Length() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.dll.Len()
}

// Get returns the value for the specified key if the key exists in the map.
// The second return value is a boolean, which will be set to true if the key
// was found in the map, or false if the key was not found in the map.
func (m *Map[K, V]) Get(key K) (V, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	v, ok := m.mp[key]
	if !ok {
		var value V
		return value, ok
	}

	me := v.Value.(mapElement[K, V])
	return me.value, ok
}

// GetAndDelete removes the first (i.e., oldest) element from the list and
// returns it. If the list is empty, the call will block until a new element
// is added.  The second return value will be set to true if the list was not
// empty; otherwise it will be set to false if the list is empty.
func (m *Map[K, V]) GetAndDelete() (V, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Wait for "Put()" or "PutIfAbsent()" to add an element to the list.
	// As per https://pkg.go.dev/sync#Cond.Wait, Wait atomically unlocks
	// m.mu and suspends execution of the calling goroutine. After later
	// resuming execution, Wait locks m.mu before returning.
	for m.dll.Len() == 0 {
		m.cond.Wait()
	}

	// Get the first element from the list. "Back()" returns the oldest
	// element in the list and "Front()" returns the newest.
	firstElement := m.dll.Back()

	if firstElement == nil {
		var value V
		return value, false
	}

	// The value that we store in the list is the key to the map.
	mapElem := firstElement.Value.(mapElement[K, V])

	// Remove the first element from the list.
	m.dll.Remove(firstElement)

	// Delete the entry from the map.
	delete(m.mp, mapElem.key)

	return mapElem.value, true
}
