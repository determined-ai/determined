package db

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/trialv1"
)

// InsertTrialProfilerMetricsBatch inserts a batch of metrics into the database.
func (db *PgDB) InsertTrialProfilerMetricsBatch(
	values []float32, batches []int32, timestamps []time.Time, labels []byte,
) error {
	_, err := db.sql.Exec(`
INSERT INTO trial_profiler_metrics (values, batches, ts, ts_range, labels)
VALUES ($1, $2, $3, tstzrange($4, $5, '[]'), $6)
`, values, batches, timestamps, timestamps[0], timestamps[len(timestamps)-1], labels)
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

const summaryWindowSeconds = 3 * 60

// GetTrialProfilerMetricSummary gets a summary of profiler metrics.
func (db *PgDB) GetTrialProfilerMetricSummary(
	ctx context.Context, labels *trialv1.TrialProfilerMetricLabels, // nolint:interfacer
) (*trialv1.TrialProfilerMetricSummary, error) {
	labelsJSON, err := protojson.Marshal(labels)
	if err != nil {
		return nil, errors.Wrap(err, "marshaling labels")
	}

	var average float32
	var stable bool
	if err := db.sql.QueryRowxContext(ctx, fmt.Sprintf(`
WITH latest AS (
	SELECT upper(m.ts_range) AS ts
	FROM trial_profiler_metrics m
	WHERE m.labels @> $1::jsonb
	ORDER BY id DESC
	LIMIT 1
)
SELECT
  coalesce(avg(q.v), 0) AS avg,
  -- classic signal-to-noise ratio + is obviously increasing/decreasing
  coalesce((
		((avg(q.v) ^ 2.0) / (stddev(q.v) ^ 2.0) > 5.0)
		AND (regr_slope(q.v, rn) < 0.5)
	), false) AS stable
FROM (
  SELECT
    row_number() OVER () as rn,
    unnest(m.values) AS v
  FROM trial_profiler_metrics m, latest
  WHERE m.labels @> $1::jsonb
    AND m.ts_range && tstzrange(latest.ts - interval '%d seconds', latest.ts, '[]')
) q`, summaryWindowSeconds), labelsJSON).Scan(&average, &stable); err != nil {
		return nil, errors.Wrapf(err, "querying summary data for %s", labelsJSON)
	}

	return &trialv1.TrialProfilerMetricSummary{
		Average: average,
		Stable:  stable,
	}, nil
}
