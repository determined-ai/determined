package db

import (
	"database/sql"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/model"
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

// TrialSnapshot returns the snapshot for the specified trial.
func (db *PgDB) TrialSnapshot(
	experimentID int, requestID model.RequestID,
) ([]byte, int, error) {
	ret := struct {
		Version int    `db:"version"`
		Content []byte `db:"content"`
	}{}
	if err := db.query(`
SELECT version, content
FROM trial_snapshots
WHERE experiment_id = $1 AND request_id = $2
ORDER BY id DESC
LIMIT 1`, &ret, experimentID, requestID); errors.Cause(err) == ErrNotFound {
		return nil, 0, nil
	} else if err != nil {
		return nil, 0, errors.Wrapf(
			err, "error querying for trial snapshot (%d, %d)", experimentID, requestID)
	}
	return ret.Content, ret.Version, nil
}

// SaveSnapshot saves a searcher and trial snapshot together.
func (db *PgDB) SaveSnapshot(
	experimentID int, trialID int, requestID model.RequestID,
	version int, experimentSnapshot []byte, trialSnapshot []byte,
) error {
	return db.withTransaction("save snapshot", func(tx *sql.Tx) error {
		if _, err := tx.Exec(`
INSERT INTO experiment_snapshots (experiment_id, content, version)
VALUES ($1, $2, $3)
ON CONFLICT (experiment_id)
DO UPDATE SET
  updated_at = now(),
  content = EXCLUDED.content,
  version = EXCLUDED.version`, experimentID, experimentSnapshot, version); err != nil {
			return errors.Wrap(err, "failed to upsert experiment snapshot")
		}

		if trialSnapshot != nil {
			if _, err := tx.Exec(`
INSERT INTO trial_snapshots (experiment_id, trial_id, request_id, content, version)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (trial_id)
DO UPDATE SET
  updated_at = now(),
  content = EXCLUDED.content,
  version = EXCLUDED.version`,
				experimentID, trialID, requestID, trialSnapshot, version); err != nil {
				return errors.Wrap(err, "failed to upsert trial snapshot")
			}
		}
		return nil
	})
}

// DeleteSnapshotsForExperiment deletes all snapshots for the given experiment.
func (db *PgDB) DeleteSnapshotsForExperiment(experimentID int) error {
	return db.withTransaction("delete snapshots", db.deleteSnapshotsForExperiment(experimentID))
}

func (db *PgDB) deleteSnapshotsForExperiment(experimentID int) func(tx *sql.Tx) error {
	return func(tx *sql.Tx) error {
		if _, err := tx.Exec(`
DELETE FROM experiment_snapshots
WHERE experiment_id = $1`, experimentID); err != nil {
			return errors.Wrap(err, "failed to delete experiment snapshots")
		}
		if _, err := tx.Exec(`
DELETE FROM trial_snapshots
WHERE experiment_id = $1`, experimentID); err != nil {
			return errors.Wrap(err, "failed to delete trial snapshots")
		}
		return nil
	}
}

// DeleteSnapshotsForTerminalExperiments deletes all snapshots for
// terminal state experiments from the database.
func (db *PgDB) DeleteSnapshotsForTerminalExperiments() error {
	return db.withTransaction("delete snapshots", func(tx *sql.Tx) error {
		if _, err := tx.Exec(`
DELETE FROM experiment_snapshots
WHERE experiment_id IN (
	SELECT id
	FROM experiments
	WHERE state IN ('COMPLETED', 'CANCELED', 'ERROR'))`); err != nil {
			return errors.Wrap(err, "failed to delete experiment snapshots")
		}
		if _, err := tx.Exec(`
DELETE FROM trial_snapshots
WHERE experiment_id IN (
	SELECT id
	FROM experiments
	WHERE state IN ('COMPLETED', 'CANCELED', 'ERROR'))`); err != nil {
			return errors.Wrap(err, "failed to delete trial snapshots")
		}
		return nil
	})
}
