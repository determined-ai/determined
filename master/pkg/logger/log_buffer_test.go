package logger

import (
	"fmt"
	"testing"

	"github.com/sirupsen/logrus"
	"gotest.tools/assert"
)

func checkIDRange(entries []*Entry, expectedStartID int, expectedEndID int) error {
	startID := entries[0].ID
	if startID != expectedStartID {
		return fmt.Errorf("unexpected startID: %v != %v", startID, expectedStartID)
	}

	endID := entries[len(entries)-1].ID
	if endID != expectedEndID {
		return fmt.Errorf("unexpected endID: %v != %v", endID, expectedEndID)
	}

	actualLen := len(entries)
	expectedLen := expectedEndID - expectedStartID + 1
	if actualLen != expectedLen {
		return fmt.Errorf("unexpected length of entries: %v != %v", actualLen, expectedLen)
	}

	// Check the order of all IDs.
	for index, entry := range entries {
		expected := index + expectedStartID
		if entry.ID != expected {
			return fmt.Errorf("unexpected entryID: %v != %v", entry.ID, expected)
		}
	}

	return nil
}

func TestLogBuffer(t *testing.T) {
	capacity := 100
	entryCount := 0
	var entries []*Entry
	buffer := NewLogBuffer(capacity)
	writeEntry := func() {
		buffer.write(&Entry{ID: entryCount})
		entryCount++
	}

	assert.Equal(t, len(buffer.Entries(0, -1, -1)), 0)

	writeEntry()
	assert.Equal(t, len(buffer.Entries(0, -1, -1)), 1)
	assert.Equal(t, len(buffer.Entries(-1, -1, -1)), 1)
	assert.Equal(t, len(buffer.Entries(-1, -1, 1)), 1)
	assert.NilError(t, checkIDRange(buffer.Entries(0, -1, -1), 0, 0))
	assert.Equal(t, len(buffer.Entries(1, -1, -1)), 0)
	assert.Equal(t, len(buffer.Entries(2, -1, -1)), 0)

	writeEntry()
	assert.Equal(t, len(buffer.Entries(0, -1, -1)), 2)
	assert.Equal(t, len(buffer.Entries(1, -1, -1)), 1)

	// Write up to one before full capacity.
	for i := 0; i < capacity-3; i++ {
		writeEntry()
	}
	entries = buffer.Entries(0, -1, -1)
	assert.NilError(t, checkIDRange(entries, 0, 98))

	// StartID equal to the biggest entryID in the buffer.
	entries = buffer.Entries(98, -1, -1)
	assert.NilError(t, checkIDRange(entries, 98, 98))

	entries = buffer.Entries(0, -1, 10)
	assert.NilError(t, checkIDRange(entries, 0, 9))

	entries = buffer.Entries(-1, -1, 10)
	assert.NilError(t, checkIDRange(entries, 89, 98))

	// Fill up to capacity once.
	writeEntry()
	entries = buffer.Entries(0, -1, -1)
	assert.NilError(t, checkIDRange(entries, 0, 99))

	// Go one over.
	writeEntry()
	entries = buffer.Entries(0, -1, -1)
	assert.NilError(t, checkIDRange(entries, 1, 100))

	entries = buffer.Entries(1, -1, -1)
	assert.NilError(t, checkIDRange(entries, 1, 100))

	entries = buffer.Entries(-1, -1, 10)
	assert.NilError(t, checkIDRange(entries, 91, 100))

	// 0 should no longer be available.
	entries = buffer.Entries(0, -1, -1)
	assert.NilError(t, checkIDRange(entries, 1, 100))

	entries = buffer.Entries(2, -1, -1)
	assert.NilError(t, checkIDRange(entries, 2, 100))

	entries = buffer.Entries(50, -1, -1)
	assert.NilError(t, checkIDRange(entries, 50, 100))

	// Fill up halfway to the second round.
	for i := 0; i < capacity/2-1; i++ {
		writeEntry()
	}

	entries = buffer.Entries(0, -1, -1)
	assert.NilError(t, checkIDRange(entries, 50, 149))

	entries = buffer.Entries(49, -1, -1)
	assert.NilError(t, checkIDRange(entries, 50, 149))

	entries = buffer.Entries(51, -1, -1)
	assert.NilError(t, checkIDRange(entries, 51, 149))
}

func TestLogBufferEndID(t *testing.T) {
	capacity := 100
	entryCount := 0
	buffer := NewLogBuffer(capacity)
	writeEntry := func(count int) {
		for i := 0; i < count; i++ {
			buffer.write(&Entry{ID: entryCount})
			entryCount++
		}
	}

	assert.Equal(t, len(buffer.Entries(-1, -1, -1)), 0)
	assert.Equal(t, len(buffer.Entries(-1, 0, -1)), 0)

	writeEntry(1)
	assert.Equal(t, len(buffer.Entries(-1, 0, -1)), 0)
	assert.NilError(t, checkIDRange(buffer.Entries(-1, 1, -1), 0, 0))

	// Fill up once.
	writeEntry(capacity - 1)

	assert.NilError(t, checkIDRange(buffer.Entries(-1, -1, -1), 0, 99))
	assert.NilError(t, checkIDRange(buffer.Entries(-1, 10, -1), 0, 9))
	assert.Equal(t, len(buffer.Entries(10, 5, -1)), 0)
	assert.Equal(t, len(buffer.Entries(10, 10, -1)), 0)
	assert.Equal(t, len(buffer.Entries(10, 11, -1)), 1)

	// Write one over.
	writeEntry(1)
	assert.NilError(t, checkIDRange(buffer.Entries(-1, -1, -1), 1, 100))
	assert.NilError(t, checkIDRange(buffer.Entries(-1, 50, 10), 40, 49))
}

func TestComputeSlice(t *testing.T) {
	capacity := 3
	var startIndex, length int
	_, length = computeSlice(-1, -1, -1, 0, capacity)
	assert.Equal(t, length, 0)
	startIndex, length = computeSlice(-1, -1, -1, 1, capacity)
	assert.Equal(t, startIndex, 0)
	assert.Equal(t, length, 1)

	// Zero limit should not return any entries.
	_, length = computeSlice(-1, -1, 0, 1, capacity)
	assert.Equal(t, length, 0)

	// Negative limit other than -1 should not return any entries.
	_, length = computeSlice(-1, -1, -2, 1, capacity)
	assert.Equal(t, length, 0)

	startIndex, length = computeSlice(0, 1, -1, 1, 1)
	assert.Equal(t, startIndex, 0)
	assert.Equal(t, length, 1)

	startIndex, length = computeSlice(0, 1, -1, 1, 2)
	assert.Equal(t, startIndex, 0)
	assert.Equal(t, length, 1)

	startIndex, length = computeSlice(1, 2, -1, 2, 4)
	assert.Equal(t, startIndex, 1)
	assert.Equal(t, length, 1)

	startIndex, length = computeSlice(0, 1, -1, 1, 3)
	assert.Equal(t, startIndex, 0)
	assert.Equal(t, length, 1)

	startIndex, length = computeSlice(0, 1, -1, 1, 4)
	assert.Equal(t, startIndex, 0)
	assert.Equal(t, length, 1)

	startIndex, length = computeSlice(0, 4, -1, 4, 4)
	assert.Equal(t, startIndex, 0)
	assert.Equal(t, length, 4)

	startIndex, length = computeSlice(1, 2, -1, 2, 1)
	assert.Equal(t, startIndex, 0)
	assert.Equal(t, length, 1)

	startIndex, length = computeSlice(1, 2, -1, 2, 2)
	assert.Equal(t, startIndex, 1)
	assert.Equal(t, length, 1)

	startIndex, length = computeSlice(0, 1, -1, 2, 4)
	assert.Equal(t, startIndex, 0)
	assert.Equal(t, length, 1)

	startIndex, length = computeSlice(0, 1, -1, 3, 3)
	assert.Equal(t, startIndex, 0)
	assert.Equal(t, length, 1)

	startIndex, length = computeSlice(1, 2, -1, 4, 4)
	assert.Equal(t, startIndex, 1)
	assert.Equal(t, length, 1)

	startIndex, length = computeSlice(1, 3, -1, 2, 1)
	assert.Equal(t, startIndex, 0)
	assert.Equal(t, length, 1)

	startIndex, length = computeSlice(1, 3, -1, 2, 2)
	assert.Equal(t, startIndex, 1)
	assert.Equal(t, length, 1)

	startIndex, length = computeSlice(1, 3, -1, 2, 3)
	assert.Equal(t, startIndex, 1)
	assert.Equal(t, length, 1)

	startIndex, length = computeSlice(1, 4, -1, 3, 4)
	assert.Equal(t, startIndex, 1)
	assert.Equal(t, length, 2)
	startIndex, length = computeSlice(2, 4, -1, 3, 3)
	assert.Equal(t, startIndex, 2)
	assert.Equal(t, length, 1)

	startIndex, length = computeSlice(3, 4, -1, 4, 4)
	assert.Equal(t, startIndex, 3)
	assert.Equal(t, length, 1)
}

func TestEntryFieldInclusion(t *testing.T) {
	buffer := NewLogBuffer(10)
	logger := logrus.StandardLogger()

	fields := map[string]interface{}{"keyA": "valA", "keyB": "valB"}
	originalEntry := logger.WithFields(fields)
	originalEntry.Message = "important message"

	assert.NilError(t, buffer.Fire(originalEntry))

	savedEntry := buffer.Entries(-1, -1, -1)[0]
	assert.Equal(t, savedEntry.Message, originalEntry.Message+`  keyA="valA" keyB="valB"`)

	fieldsB := map[string]interface{}{"keyA": `my great "quote"`}
	originalEntry = logger.WithFields(fieldsB)
	originalEntry.Message = "another message"

	assert.NilError(t, buffer.Fire(originalEntry))

	savedEntry = buffer.Entries(-1, -1, -1)[1]
	assert.Equal(t, savedEntry.Message, originalEntry.Message+`  keyA="my great \"quote\""`)
}
