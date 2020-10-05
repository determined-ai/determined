package fetchers

import (
	"github.com/determined-ai/determined/master/internal/logs"
	"github.com/determined-ai/determined/master/pkg/logger"
)

// MasterLogsFetcher is a fetcher for in-memory master logs.
type MasterLogsFetcher struct {
	buffer *logger.LogBuffer
	offset int
}

// NewMasterLogsFetcher returns a new MasterLogsFetcher.
func NewMasterLogsFetcher(buffer *logger.LogBuffer, offset int) *MasterLogsFetcher {
	return &MasterLogsFetcher{buffer: buffer, offset: offset}
}

// Fetch implements logs.Fetcher.
func (f *MasterLogsFetcher) Fetch(limit int, unlimited bool) (logs.Batch, error) {
	if unlimited {
		limit = -1
	}
	entries := f.buffer.Entries(f.offset, -1, limit)
	f.offset += len(entries)
	return logger.EntriesBatch(entries), nil
}
