package fetchers

import (
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/logs"
	"github.com/determined-ai/determined/master/pkg/model"
)

const (
	batchSize = 1000
)

// PostgresTrialLogsFetcher is a fetcher for postgres-backed trial logs.
type PostgresTrialLogsFetcher struct {
	db      *db.PgDB
	trialID int
	offset  int
}

// NewPostgresTrialLogsFetcher returns a new PostgresTrialLogsFetcher.
func NewPostgresTrialLogsFetcher(db *db.PgDB, trialID, offset int) *PostgresTrialLogsFetcher {
	return &PostgresTrialLogsFetcher{db: db, trialID: trialID, offset: offset}
}

// Fetch implements logs.Fetcher
func (p *PostgresTrialLogsFetcher) Fetch(limit int, unlimited bool) (logs.Batch, error) {
	switch {
	case unlimited || limit > batchSize:
		limit = batchSize
	case limit <= 0:
		return nil, nil
	}

	var b []*model.TrialLog
	err := p.db.Query("stream_logs", &b, p.trialID, p.offset, limit)

	if len(b) != 0 {
		p.offset = b[len(b)-1].ID
	}

	return model.TrialLogBatch(b), err
}
