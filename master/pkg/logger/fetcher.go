package logger

import (
	"github.com/determined-ai/determined/master/internal/api"
)

// Fetcher is a fetcher for in-memory master logs.
type Fetcher struct {
	buffer *LogBuffer
	offset int
}

// NewFetcher returns a new LogBuffer fetcher.
func NewFetcher(buffer *LogBuffer, offset int) *Fetcher {
	return &Fetcher{buffer: buffer, offset: offset}
}

// Fetch implements logs.Fetcher.
func (f *Fetcher) Fetch(limit int, unlimited bool) (api.Batch, error) {
	if unlimited {
		limit = -1
	}
	entries := f.buffer.Entries(f.offset, -1, limit)
	f.offset += len(entries)
	return EntriesBatch(entries), nil
}
