package db

import (
	"context"

	"github.com/jmoiron/sqlx"
	"github.com/uptrace/bun"

	"github.com/pkg/errors"
)

// ExperimentSnapshot returns the snapshot for the specified experiment.
func (db *PgDB) ExperimentSnapshot(experimentID int) ([]byte, int, error) {
	ret := struct {
		Version int    `db:"version"`
		Content []byte `db:"content"`
	}{}
	if err := db.query(`
SELECT version, content
FROM experiment_snapshots
WHERE experiment_id = $1
ORDER BY id DESC
LIMIT 1`, &ret, experimentID); errors.Cause(err) == ErrNotFound {
		return nil, 0, nil
	} else if err != nil {
		return nil, 0, errors.Wrapf(err, "error querying for experiment snapshot (%d)", experimentID)
	}
	return ret.Content, ret.Version, nil
}

// SaveSnapshot saves a searcher and trial snapshot together.
func (db *PgDB) SaveSnapshot(
	experimentID int, version int, experimentSnapshot []byte,
) error {
	if _, err := db.sql.Exec(`
INSERT INTO experiment_snapshots (experiment_id, content, version)
VALUES ($1, $2, $3)
ON CONFLICT (experiment_id)
DO UPDATE SET
  updated_at = now(),
  content = EXCLUDED.content,
  version = EXCLUDED.version`, experimentID, experimentSnapshot, version); err != nil {
		return errors.Wrap(err, "failed to upsert experiment snapshot")
	}
	return nil
}

// DeleteSnapshotsForExperiment deletes all snapshots for one given experiment.
func (db *PgDB) DeleteSnapshotsForExperiment(experimentID int) error {
	return db.withTransaction("delete snapshots", db.deleteSnapshotsForExperiment(experimentID))
}

func (db *PgDB) deleteSnapshotsForExperiment(experimentID int) func(tx *sqlx.Tx) error {
	return func(tx *sqlx.Tx) error {
		if _, err := tx.Exec(`
DELETE FROM experiment_snapshots
WHERE experiment_id = $1`, experimentID); err != nil {
			return errors.Wrap(err, "failed to delete experiment snapshots")
		}
		return nil
	}
}

// DeleteSnapshotsForExperiments deletes all snapshots for multiple given experiments.
func (db *PgDB) DeleteSnapshotsForExperiments(experimentIDs []int) func(ctx context.Context,
	tx *bun.Tx) error {
	return func(ctx context.Context, tx *bun.Tx) error {
		var snapIDs []int
		if _, err := tx.NewDelete().Model(&snapIDs).Table("experiment_snapshots").
			Where("experiment_id IN (?)", bun.In(experimentIDs)).
			Returning("id").
			Exec(ctx); err != nil {
			return errors.Wrap(err, "failed to delete experiments snapshots")
		}
		return nil
	}
}

// DeleteSnapshotsForTerminalExperiments deletes all snapshots for
// terminal state experiments from the database.
func (db *PgDB) DeleteSnapshotsForTerminalExperiments() error {
	if _, err := db.sql.Exec(`
DELETE FROM experiment_snapshots
WHERE experiment_id IN (
	SELECT id
	FROM experiments
	WHERE state IN ('COMPLETED', 'CANCELED', 'ERROR'))`); err != nil {
		return errors.Wrap(err, "failed to delete experiment snapshots")
	}
	return nil
}
