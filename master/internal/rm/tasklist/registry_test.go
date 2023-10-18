package tasklist

import (
	"testing"
	"time"

	"gotest.tools/assert"
)

func TestRegistry(t *testing.T) {
	registry := NewRegistry[string, string]()

	val, ok := registry.Load("key1")
	assert.Assert(t, !ok)
	assert.Equal(t, val, "")

	val, ok = registry.Load("key2")
	assert.Assert(t, !ok)
	assert.Equal(t, val, "")

	assert.Assert(t, registry.Add("key1", "value1"))
	assert.Assert(t, registry.Add("key2", "value2"))

	val, ok = registry.Load("key1")
	assert.Assert(t, ok)
	assert.Equal(t, val, "value1")

	val, ok = registry.Load("key2")
	assert.Assert(t, ok)
	assert.Equal(t, val, "value2")

	deletedKey1a := make(chan bool)
	registry.OnDelete("key1", func() {
		_, exists := registry.Load("key1")
		assert.Assert(t, !exists)
		close(deletedKey1a)
	})
	deletedKey1b := make(chan bool)
	registry.OnDelete("key1", func() {
		_, exists := registry.Load("key1")
		assert.Assert(t, !exists)
		close(deletedKey1b)
	})

	deletedKey2 := make(chan bool)
	registry.OnDelete("key2", func() {
		_, exists := registry.Load("key2")
		assert.Assert(t, !exists)
		close(deletedKey2)
	})

	assert.Assert(t, registry.Delete("key1"))

	select {
	case <-deletedKey1a:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for delete callback")
	}
	select {
	case <-deletedKey1b:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for delete callback")
	}

	select {
	case <-deletedKey2:
		t.Fatal("delete key2 callback should not have been called")
	case <-time.After(time.Second):
	}

	val, ok = registry.Load("key1")
	assert.Assert(t, !ok)
	assert.Equal(t, val, "")

	val, ok = registry.Load("key2")
	assert.Assert(t, ok)
	assert.Equal(t, val, "value2")
}
