package db

import "github.com/determined-ai/determined/proto/pkg/trialv1"

func (db *PgDB) InsertTrialProfilerMetrics(b *trialv1.TrialProfilerMetricsBatch) error {
	_, err := db.sql.Exec(`
INSERT INTO trial_profiler_metrics (values, batches, timestamps, labels)
VALUES ($1, $2, $3, $4)
`, b.Values, b.Batches, b.Timestamps, b.Labels)
	return err
}
