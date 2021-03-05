package db

import (
	"time"
)

// InsertTrialProfilerMetricsBatch inserts a batch of metrics into the database.
func (db *PgDB) InsertTrialProfilerMetricsBatch(
	values []float32, batches []int32, timestamps []time.Time, labels []byte,
) error {
	_, err := db.sql.Exec(`
INSERT INTO trial_profiler_metrics (values, batches, ts, labels)
VALUES ($1, $2, $3, $4)
`, values, batches, timestamps, labels)
	return err
}
