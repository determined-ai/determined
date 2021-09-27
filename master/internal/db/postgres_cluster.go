package db

import (
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// GetClusterID queries the master uuid in the database, first adding it if it doesn't exist.
func (db *PgDB) GetClusterID() (string, error) {
	newUUID := uuid.New().String()

	if _, err := db.sql.Exec(`
INSERT INTO cluster_id (cluster_id) SELECT ($1)
WHERE NOT EXISTS ( SELECT * FROM cluster_id );
`, newUUID); err != nil {
		return "", errors.Wrapf(err, "error initializing cluster_id in cluster_id table")
	}

	var uuidVal []string

	if err := db.sql.Select(&uuidVal, `SELECT cluster_id FROM cluster_id`); err != nil {
		return "", errors.Wrapf(err, "error reading cluster_id from cluster_id table")
	}
	if len(uuidVal) != 1 {
		return "", errors.Errorf(
			"expecting exactly one cluster_id from cluster_id table, %d values found", len(uuidVal),
		)
	}
	return uuidVal[0], nil
}

// PeriodicTelemetryInfo returns anonymous information about the usage of the current
// Determined cluster.
func (db *PgDB) PeriodicTelemetryInfo() ([]byte, error) {
	return db.rawQuery(`
SELECT jsonb_build_object(
    'num_users', (SELECT count(*) FROM users),
    'num_experiments', (SELECT count(*) FROM experiments),
    'num_trials', (SELECT count(*) FROM trials),
    'experiment_states', (SELECT jsonb_agg(t) FROM
                           (SELECT state, count(*)
                            FROM experiments GROUP BY state) t)
);
`)
}

// UpdateResourceAllocationAggregation updates the aggregated resource allocation table.
func (db *PgDB) UpdateResourceAllocationAggregation() error {
	var lastDatePtr *time.Time
	err := db.sql.QueryRow(
		`SELECT date_trunc('day', max(date)) FROM resource_aggregates`,
	).Scan(&lastDatePtr)
	if err != nil {
		return errors.Wrap(err, "failed to find last aggregate")
	}

	// The values periodStart takes on are all midnight UTC (because of date_trunc) for each day that
	// is to be aggregated.
	var periodStart time.Time
	if lastDatePtr == nil {
		var firstDatePtr *time.Time
		err := db.sql.QueryRow(
			`SELECT date_trunc('day', min(start_time)) FROM allocations`,
		).Scan(&firstDatePtr)
		if err != nil {
			return errors.Wrap(err, "failed to find first step")
		}
		if firstDatePtr == nil {
			// No steps found; nothing to do.
			return nil
		}

		periodStart = firstDatePtr.UTC()
	} else {
		periodStart = lastDatePtr.UTC().AddDate(0, 0, 1)
	}

	// targetDate is some time during the day before today, which is the last full day that has ended
	// and can therefore be aggregated; the Before check means that the last value of periodStart is
	// midnight at the beginning of that day.
	targetDate := time.Now().UTC().AddDate(0, 0, -1)
	for ; periodStart.Before(targetDate); periodStart = periodStart.AddDate(0, 0, 1) {
		t0 := time.Now()

		if _, err := db.sql.Exec(
			db.queries.getOrLoad("update_aggregated_allocation"), periodStart,
		); err != nil {
			return errors.Wrap(err, "failed to add aggregate")
		}

		log.Infof(
			"aggregated resource allocation statistics for %v in %s",
			periodStart, time.Since(t0),
		)
	}

	return nil
}
