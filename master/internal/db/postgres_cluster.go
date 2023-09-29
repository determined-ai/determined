package db

import (
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

// BCryptCost is a stopgap until we implement sane master-configuration.
const BCryptCost = 15

// GetOrCreateClusterID queries the master uuid in the database, adding one if it doesn't exist.
func (db *PgDB) GetOrCreateClusterID() (string, error) {
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

// UpdateClusterHeartBeat updates the clusterheartbeat column in the cluster_id table.
func (db *PgDB) UpdateClusterHeartBeat(currentClusterHeartbeat time.Time) error {
	_, err := db.sql.Exec(`UPDATE cluster_id SET cluster_heartbeat = $1`, currentClusterHeartbeat)
	return errors.Wrap(err, "updating cluster heartbeat")
}

// PeriodicTelemetryInfo returns anonymous information about the usage of the current
// Determined cluster.
func (db *PgDB) PeriodicTelemetryInfo() ([]byte, error) {
	return db.rawQuery(`
SELECT jsonb_build_object(
    'num_users', (SELECT count(*) FROM users),
    'num_experiments', (SELECT count(*) FROM experiments),
    'num_trials', (SELECT count(*) FROM trials),
	'num_jobs', (SELECT count(*) FROM jobs),
	'num_tasks', (SELECT count(*) FROM tasks),
	'num_allocations', (SELECT count(*) FROM allocations),
    'num_allocations_today', (
		SELECT count(*) FROM allocations WHERE DATE(start_time) >= CURRENT_DATE
	),
	'num_workspaces', (SELECT count(*) FROM workspaces),
	'num_projects', (SELECT count(*) FROM projects),
	'avg_projects_per_workspace', (
		(SELECT count(*) FROM projects WHERE id > 1)::float
		/ (SELECT greatest(count(*), 1) FROM workspaces WHERE id > 1)
	),
	'avg_experiments_per_project', (
		(SELECT count(*) FROM experiments WHERE project_id > 1)::float
		/ (SELECT greatest(count(*), 1) FROM projects WHERE id > 1)
	),
	'notes_gt_zero', (
		(SELECT count(*) FROM projects WHERE id > 1 AND jsonb_array_length(notes) > 0)::float
		/ (SELECT greatest(count(*), 1) FROM projects WHERE id > 1)
	),
	'notes_gt_one', (
		(SELECT count(*) FROM projects WHERE id > 1 AND jsonb_array_length(notes) > 1)::float
		/ (SELECT greatest(count(*), 1) FROM projects WHERE id > 1)
	)
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
			db.queries.GetOrLoad("update_aggregated_allocation"), periodStart,
		); err != nil {
			return errors.Wrap(err, "failed to add aggregate allocation")
		}

		if _, err := db.sql.Exec(
			db.queries.GetOrLoad("update_aggregated_queued_time"), periodStart,
		); err != nil {
			return errors.Wrap(err, "failed to add aggregate queued time")
		}

		log.Infof(
			"aggregated resource allocation statistics for %v in %s",
			periodStart, time.Since(t0),
		)
	}
	return nil
}

// SetInitialPassword initializes 'admin' and 'determined' user accounts with given password.
func (db *PgDB) SetInitialPassword(password string) error {
	passwordHash, err := bcrypt.GenerateFromPassword(
		[]byte(password),
		BCryptCost,
	)
	if err != nil {
		return err
	}

	_, err = db.sql.Exec(
		`UPDATE users SET password_hash = $1
	 	WHERE username = 'admin' OR username = 'determined';`,
		passwordHash)

	return errors.Wrapf(err, "error initializing admin and determined passwords in users table")
}
