package logger

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/determined-ai/determined/master/internal/logs"

	"github.com/sirupsen/logrus"
)

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func computeSlice(startID int, endID int, limit int, totalEntries int, capacity int) (int, int) {
	if endID < -1 || startID < -1 || limit < -1 {
		return 0, 0
	}

	// Handle special values.
	if endID == -1 {
		endID = totalEntries
	}
	if limit == -1 {
		limit = capacity
	}

	selectTail := startID == -1

	// Limit values to appropriate bounds.
	startID = max(startID, max(0, totalEntries-capacity))
	endID = min(endID, totalEntries)
	if startID >= endID {
		return 0, 0
	}
	limit = min(limit, endID-startID)

	// Select the newest entries if the limit is taking effect and no startID was provided.
	if selectTail {
		startID = endID - limit
	}

	return startID % capacity, limit
}

func logrusMessageAndData(entry *logrus.Entry) string {
	if len(entry.Data) == 0 {
		return entry.Message
	}

	// Stringify the fields in a sorted order.
	keys := make([]string, 0, len(entry.Data))
	for key := range entry.Data {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var fields []string
	for _, key := range keys {
		strValue := fmt.Sprintf("%v", entry.Data[key])
		fields = append(
			fields,
			fmt.Sprintf("%s=%q", key, strValue),
		)
	}

	return entry.Message + "  " + strings.Join(fields, " ")
}

// Entry captures the interesting attributes of logrus.Entry.
type Entry struct {
	ID      int          `json:"id"`
	Message string       `json:"message"`
	Time    time.Time    `json:"time"`
	Level   logrus.Level `json:"level"`
}

// EntriesBatch is a batch of logger.Entry.
type EntriesBatch []*Entry

// Size implements logs.Batch.
func (eb EntriesBatch) Size() int {
	return len(eb)
}

// ForEach implements logs.Batch.
func (eb EntriesBatch) ForEach(f func(logs.Record) error) error {
	for _, e := range eb {
		if err := f(e); err != nil {
			return err
		}
	}
	return nil
}

// LogBuffer is an in-memory buffer based logger.
type LogBuffer struct {
	lock         sync.RWMutex
	buffer       []*Entry
	totalEntries int
}

// NewLogBuffer creates a new LogBuffer.
func NewLogBuffer(capacity int) *LogBuffer {
	return &LogBuffer{
		buffer: make([]*Entry, capacity),
	}
}

func (lb *LogBuffer) write(entry *Entry) {
	lb.lock.Lock()
	defer lb.lock.Unlock()
	// Write to the head of the buffer.
	entry.ID = lb.totalEntries
	lb.buffer[lb.totalEntries%len(lb.buffer)] = entry
	lb.totalEntries++
}

// Entries retrieves a snapshot of the newest logged entries.
//
//   - startID: Beginning of the range of IDs to include (inclusive).
//   - endID: End of the range of IDs to include (exclusive).
//   - limit: Maximum number of entries to return.
//
// Special cases:
//
//   - startID == -1: Don't limit the minimum ID.
//   - endID == -1: Don't limit the maximum ID.
//   - limit == -1: Don't limit the number of entries.
func (lb *LogBuffer) Entries(startID int, endID int, limit int) []*Entry {
	lb.lock.RLock()
	defer lb.lock.RUnlock()

	startIndex, entryCount :=
		computeSlice(startID, endID, limit, lb.totalEntries, len(lb.buffer))
	if entryCount <= 0 {
		return nil
	}

	// Copy the pointers to entries from the underlying lb.buffer into a new slice to return.
	// We can avoid copying the contents of the entries since they are not modified by this
	// object.
	entries := make([]*Entry, entryCount)
	copiedCount := copy(entries, lb.buffer[startIndex:])
	// Fill in the rest of entries beginning from the start of lb.buffer.
	copy(entries[copiedCount:], lb.buffer)

	return entries
}

// Len returns the total number of entries written to the buffer.
func (lb *LogBuffer) Len() int {
	lb.lock.RLock()
	defer lb.lock.RUnlock()
	return lb.totalEntries
}

// Fire implements the logrus.Hook interface.
func (lb *LogBuffer) Fire(entry *logrus.Entry) error {
	lb.write(&Entry{
		Message: logrusMessageAndData(entry),
		Time:    entry.Time,
		Level:   entry.Level,
	})
	return nil
}

// Levels implements the logrus.Hook interface.
func (lb *LogBuffer) Levels() []logrus.Level {
	return logrus.AllLevels
}
