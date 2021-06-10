package db

import (
	"time"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/trialv1"
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

// GetTrialProfilerMetricsBatches gets a batch of profiler metric batches from the database.
func (db *PgDB) GetTrialProfilerMetricsBatches(
	labelsJSON []byte, offset, limit int,
) (model.TrialProfilerMetricsBatchBatch, error) {
	rows, err := db.sql.Queryx(`
SELECT
    m.values AS values,
    m.batches AS batches,
    m.ts AS timestamps,
    m.labels AS labels
FROM trial_profiler_metrics m
WHERE m.labels @> $1::jsonb
ORDER by m.id
OFFSET $2 LIMIT $3`, labelsJSON, offset, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pBatches []*trialv1.TrialProfilerMetricsBatch
	for rows.Next() {
		var batch model.TrialProfilerMetricsBatch
		if err := rows.StructScan(&batch); err != nil {
			return nil, errors.Wrap(err, "querying profiler metric batch")
		}

		pBatch, err := batch.ToProto()
		if err != nil {
			return nil, errors.Wrap(err, "converting batch to protobuf")
		}

		pBatches = append(pBatches, pBatch)
	}
	return pBatches, nil
}
