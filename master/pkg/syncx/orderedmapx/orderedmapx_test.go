package orderedmapx

import (
	"fmt"
	"testing"

	"gotest.tools/assert"
)

type KeyAndValue struct {
	key   string
	value string
}

var testData = []KeyAndValue{
	{
		key:   "key1",
		value: "value1",
	},
	{
		key:   "key2",
		value: "value2",
	},
	{
		key:   "key3",
		value: "value3",
	},
	{
		key:   "key4",
		value: "value4",
	},
}

func Test_GetAndDelete(t *testing.T) {
	m := New[string, string]()

	// Add some data to the map.
	for _, data := range testData {
		m.Put(data.key, data.value)
	}

	// Verify the map contains expected number of items.
	assert.Equal(t, m.Length(), len(testData))

	// Verify that "GetFirst()" returns the first (i.e., oldest) element in the
	// map.
	for index, data := range testData {
		assert.Equal(t, m.Length() > 0, true, "List is empty")

		val, ok := m.GetAndDelete()

		assert.Equal(t, ok, true, fmt.Sprintf("Element #%d was not found", index))

		assert.Equal(t, val, data.value)

		assert.Equal(t, m.Length(), len(testData)-(index+1))
	}

	assert.Equal(t, m.Length(), 0, "Expected map length to be zero")
}

func Test_Put(t *testing.T) {
	m := New[string, string]()

	assert.Equal(t, m.Length(), 0, "Expected map length to be zero")

	// Add the key in the map for the first time.
	m.Put("key", "value")

	assert.Equal(t, m.Length(), 1, "Expected map length to be one")

	val, ok := m.Get("key")

	// Should be "true" since the key exists in the map.
	assert.Equal(t, ok, true)

	// Should have returned the value that we added to the map.
	assert.Equal(t, val, "value")

	// Try to add the same key to the map with a different value.
	m.Put("key", "new value")

	assert.Equal(t, m.Length(), 1, "Expected map length to be one")

	val, ok = m.Get("key")

	// Should be "true" since the key exists in the map.
	assert.Equal(t, ok, true)

	// Should have returned the new value.
	assert.Equal(t, val, "new value")

	// Delete the key from the map.
	ok = m.Delete("key")

	assert.Equal(t, m.Length(), 0, "Expected map length to be zero")

	// Should be "true" since the key exists in the map.
	assert.Equal(t, ok, true)

	// Delete the key from the map again.
	ok = m.Delete("key")

	// Should be "false" since the should no longer be in the map.
	assert.Equal(t, ok, false)
}

func Test_PutIfAbsent(t *testing.T) {
	m := New[string, string]()

	assert.Equal(t, m.Length(), 0, "Expected map length to be zero")

	// Add the key in the map for the first time.
	val, ok := m.PutIfAbsent("key", "value")

	assert.Equal(t, m.Length(), 1, "Expected map length to be one")

	// Should be "true", since the key and its value were added to the map,
	// because the key did not already exist in the map.
	assert.Equal(t, ok, true)

	// Should have returned the value that we added to the map.
	assert.Equal(t, val, "value")

	// Try to add the same key to the map.
	val, ok = m.PutIfAbsent("key", "new value")

	assert.Equal(t, m.Length(), 1, "Expected map length to be one")

	// Should be "false", since the key already existed in the map and its
	// value was not updated.
	assert.Equal(t, ok, false)

	// Should have returned the value that was already in the map.
	assert.Equal(t, val, "value")
}
