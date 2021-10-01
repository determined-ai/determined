package db

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/proto/pkg/trialv1"
)

// AddTrial adds the trial to the database and sets its ID.
func (db *PgDB) AddTrial(trial *model.Trial) error {
	return db.withTransaction("add_trial", func(tx *sqlx.Tx) error {
		if trial.ID != 0 {
			return errors.Errorf("error adding a trial with non-zero id %v", trial.ID)
		}

		if err := namedGet(tx, &trial.ID, `
INSERT INTO trials
  (task_id, request_id, experiment_id, state, start_time, end_time,
   hparams, warm_start_checkpoint_id, seed)
VALUES (:task_id, :request_id, :experiment_id, :state, :start_time,
        :end_time, :hparams, :warm_start_checkpoint_id, :seed)
RETURNING id`, trial); err != nil {
			// Assume the foreign key constraint is handled by the database.
			return errors.Wrapf(err, "error inserting trial %v", *trial)
		}

		return addTask(tx, &model.Task{
			TaskID:     trial.TaskID,
			TaskType:   model.TaskTypeTrial,
			StartTime:  trial.StartTime,
			LogVersion: model.CurrentTaskLogVersion,
		})
	})
}

// TrialByID looks up a trial by ID, returning an error if none exists.
func (db *PgDB) TrialByID(id int) (*model.Trial, error) {
	var trial model.Trial
	err := db.query(`
SELECT id, task_id, request_id, experiment_id, state, start_time, end_time,
  hparams, warm_start_checkpoint_id, seed
FROM trials
WHERE id = $1`, &trial, id)
	return &trial, errors.Wrapf(err, "error querying for trial %v", id)
}

// TrialByExperimentAndRequestID looks up a trial, returning an error if none exists.
func (db *PgDB) TrialByExperimentAndRequestID(
	experimentID int, requestID model.RequestID,
) (*model.Trial, error) {
	var trial model.Trial
	err := db.query(`
SELECT id, task_id, request_id, experiment_id, state, start_time,
  end_time, hparams, warm_start_checkpoint_id, seed
FROM trials
WHERE experiment_id = $1 AND request_id = $2`, &trial, experimentID, requestID)
	return &trial, errors.Wrapf(err, "error querying for trial %v", requestID)
}

// UpdateTrial updates an existing trial. Fields that are nil or zero are not
// updated.  end_time is set if the trial moves to a terminal state.
func (db *PgDB) UpdateTrial(id int, newState model.State) error {
	trial, err := db.TrialByID(id)
	if err != nil {
		return errors.Wrapf(err, "error finding trial %v to update", id)
	}

	if trial.State == newState {
		return nil
	}

	if !model.TrialTransitions[trial.State][newState] {
		return errors.Errorf("illegal transition %v -> %v for trial %v",
			trial.State, newState, trial.ID)
	}
	toUpdate := []string{"state"}
	trial.State = newState
	if model.TerminalStates[newState] {
		now := time.Now().UTC()
		trial.EndTime = &now
		toUpdate = append(toUpdate, "end_time")
	}

	return db.withTransaction("update_trial", func(tx *sqlx.Tx) error {
		// Only the trial actor updates this row, and it does so in a serialized
		// fashion already, so this transaction is more a matter of atomicity.
		if err := namedExecOne(tx, fmt.Sprintf(`
UPDATE trials
%v
WHERE id = :id`, setClause(toUpdate)), trial); err != nil {
			return errors.Wrapf(err, "error updating (%v) in trial %v",
				strings.Join(toUpdate, ", "), id)
		}

		if model.TerminalStates[newState] && trial.EndTime != nil {
			return completeTask(tx, trial.TaskID, *trial.EndTime)
		}

		return nil
	})
}

// UpdateTrialRunnerState updates a trial runner's state.
func (db *PgDB) UpdateTrialRunnerState(id int, state string) error {
	return db.UpdateTrialRunnerMetadata(id, &trialv1.TrialRunnerMetadata{State: state})
}

// UpdateTrialRunnerMetadata updates a trial's metadata about its runner.
func (db *PgDB) UpdateTrialRunnerMetadata(id int, md *trialv1.TrialRunnerMetadata) error {
	if _, err := db.sql.Exec(`
UPDATE trials
SET runner_state = $2
WHERE id = $1`, id, md.State); err != nil {
		return errors.Wrap(err, "saving trial runner state")
	}
	return nil
}

// TrialDetailsRaw returns a trial as a JSON string. This includes checkpoints and
// validations for every step, plus aggregated training metrics and full validation metrics.
func (db *PgDB) TrialDetailsRaw(id int) ([]byte, error) {
	// Find the desired metric names and construct parts of the query appropriately.
	var metricNames []string
	if err := db.sql.Select(&metricNames, `
SELECT jsonb_object_keys(s.metrics->'batch_metrics'->0)
FROM (
    SELECT metrics
    FROM steps
    WHERE trial_id = $1 AND state = 'COMPLETED'
    LIMIT 1
) s
`, id); err != nil {
		return nil, errors.Wrapf(err, "failed to get metric names for trial %d", id)
	}

	averageMetrics := make([]string, 0, len(metricNames))
	for _, name := range metricNames {
		averageMetrics = append(averageMetrics,
			fmt.Sprintf(`avg(try_float8_cast(value->>'%s')) AS "%s"`, name, name),
		)
	}

	// We want to average the per-batch training metrics into per-step metrics.
	// Newer runners compute the averages in the metrics already. For legacy
	// data, we compute the averages on the fly.
	//
	// Ideally, we'd like to just cast the metric values (i.e., ::float8) but
	// there may be non-scalar training metrics, in which case casts will cause
	// an error in Postgres. We use the try_float8_cast function defined
	// earlier, which makes the whole query considerably slower, but can handle
	// that case (by returning NULL instead of aborting).
	queryTemplate := `
WITH const AS (
    SELECT config->'searcher'->>'metric' AS metric_name,
           (SELECT
               CASE
                   WHEN coalesce((config->'searcher'
                                        ->>'smaller_is_better')::boolean, true)
                   THEN 1
                   ELSE -1
               END) AS sign
    FROM experiments WHERE id IN (SELECT experiment_id FROM trials WHERE id = $1)
)
SELECT row_to_json(r1)::text
FROM (
    SELECT t.end_time, t.experiment_id, t.hparams, t.id, t.seed, t.start_time, t.state,
           t.warm_start_checkpoint_id,
           (SELECT coalesce(max(s.total_batches), 0)
            FROM steps s
            WHERE s.trial_id = t.id AND s.state = 'COMPLETED'
           ) AS total_batches_processed,
           (SELECT coalesce(jsonb_agg(row_to_json(r2) ORDER BY r2.id ASC), '[]'::jsonb)
            FROM (
                SELECT s.end_time, s.id, s.state, s.total_batches,
                       (SELECT CASE
                           WHEN s.metrics->'avg_metrics' IS NOT NULL THEN
                               (s.metrics->'avg_metrics')::json
                           ELSE (SELECT row_to_json(r3)
                                 FROM
                                    (SELECT %s
                                     FROM jsonb_array_elements(s.metrics->'batch_metrics')
                                    ) r3)
                        END) AS avg_metrics,
                       (SELECT row_to_json(r4)
                        FROM (
                            SELECT v.end_time, v.id, v.metrics, v.state,
                            FROM validations v
                            WHERE v.trial_id = t.id AND v.total_batches = s.total_batches
                                  AND v.metrics IS NOT NULL
                        ) r4
                       ) AS validation,
                       (SELECT row_to_json(r5)
                        FROM (
                            SELECT c.id, c.trial_id, c.total_batches, c.state,
                                   c.end_time, c.uuid, c.resources, c.metadata,
                                   (v.metrics->'validation_metrics'
                                             ->>const.metric_name)::float8 AS validation_metric
                            FROM checkpoints c LEFT JOIN validations v
                            ON c.trial_id = v.trial_id AND c.total_batches = v.total_batches, const
                            WHERE c.trial_id = t.id AND c.total_batches = s.total_batches
                       ) r5) AS checkpoint
                FROM steps s
                WHERE s.trial_id = t.id
            ) r2
           ) AS steps
   FROM trials t
   WHERE t.id = $1
) r1;`

	return db.rawQuery(fmt.Sprintf(queryTemplate, strings.Join(averageMetrics, ",")), id)
}

// TrialRunIDAndRestarts returns the run id and restart count for a trial.
func (db *PgDB) TrialRunIDAndRestarts(trialID int) (int, int, error) {
	var runID, restart int
	if err := db.sql.QueryRowx(`
SELECT run_id, restarts
FROM trials
WHERE id = $1`, trialID).Scan(&runID, &restart); err != nil {
		return 0, 0, errors.Wrap(err, "failed to scan trial restart count")
	}
	return runID, restart, nil
}

// UpdateTrialRunID sets the trial's run ID.
func (db *PgDB) UpdateTrialRunID(id, runID int) error {
	if _, err := db.sql.Exec(`
UPDATE trials
SET run_id = $2
WHERE id = $1`, id, runID); err != nil {
		return errors.Wrap(err, "updating trial run id")
	}
	return nil
}

// UpdateTrialRestarts sets the trial's restart count.
func (db *PgDB) UpdateTrialRestarts(id, restartCount int) error {
	if _, err := db.sql.Exec(`
UPDATE trials
SET restarts = $2
WHERE id = $1`, id, restartCount); err != nil {
		return errors.Wrap(err, "updating trial restarts")
	}
	return nil
}

// AddTrainingMetrics adds a completed step to the database with the given training metrics.
// If these training metrics occur before any others, a rollback is assumed and later
// training and validation metrics are cleaned up.
func (db *PgDB) AddTrainingMetrics(ctx context.Context, m *trialv1.TrialMetrics) error {
	return db.withTransaction("add training metrics", func(tx *sqlx.Tx) error {
		if err := checkTrialRunID(ctx, tx, m.TrialId, m.TrialRunId); err != nil {
			return err
		}

		if _, err := tx.ExecContext(ctx, `
UPDATE raw_steps SET archived = true
WHERE trial_id = $1
  AND trial_run_id < $2
  AND total_batches >= $3;
`, m.TrialId, m.TrialRunId, m.LatestBatch); err != nil {
			return errors.Wrap(err, "archiving training metrics")
		}

		if _, err := tx.ExecContext(ctx, `
UPDATE raw_validations SET archived = true
WHERE trial_id = $1
  AND trial_run_id < $2
  AND total_batches > $3;
`, m.TrialId, m.TrialRunId, m.LatestBatch); err != nil {
			return errors.Wrap(err, "archiving validations")
		}

		if _, err := tx.ExecContext(ctx, `
UPDATE raw_checkpoints SET archived = true
WHERE trial_id = $1
  AND trial_run_id < $2
  AND total_batches > $3;
`, m.TrialId, m.TrialRunId, m.LatestBatch); err != nil {
			return errors.Wrap(err, "archiving checkpoints")
		}

		if _, err := tx.NamedExecContext(ctx, `
INSERT INTO raw_steps
	(trial_id, trial_run_id, state,
	 end_time, metrics, total_batches)
VALUES
	(:trial_id, :trial_run_id, :state,
	 now(), :metrics, :total_batches)
`, model.TrialMetrics{
			TrialID:    int(m.TrialId),
			TrialRunID: int(m.TrialRunId),
			State:      model.CompletedState,
			Metrics: map[string]interface{}{
				"avg_metrics":   m.Metrics,
				"batch_metrics": m.BatchMetrics,
			},
			TotalBatches: int(m.LatestBatch),
		}); err != nil {
			return errors.Wrap(err, "inserting training metrics")
		}
		return nil
	})
}

// AddValidationMetrics adds a completed validation to the database with the given
// validation metrics. If these validation metrics occur before any others, a rollback
// is assumed and later metrics are cleaned up from the database.
func (db *PgDB) AddValidationMetrics(
	ctx context.Context, m *trialv1.TrialMetrics,
) error {
	return db.withTransaction("add validation metrics", func(tx *sqlx.Tx) error {
		if err := checkTrialRunID(ctx, tx, m.TrialId, m.TrialRunId); err != nil {
			return err
		}

		if _, err := tx.ExecContext(ctx, `
UPDATE raw_validations SET archived = true
WHERE trial_id = $1
  AND trial_run_id < $2
  AND total_batches >= $2;
`, m.TrialId, m.LatestBatch); err != nil {
			return errors.Wrap(err, "archiving validations")
		}

		if err := db.ensureStep(
			ctx, tx, int(m.TrialId), int(m.TrialRunId), int(m.LatestBatch),
		); err != nil {
			return err
		}

		if _, err := tx.NamedExecContext(ctx, `
INSERT INTO raw_validations
	(trial_id, trial_run_id, state, end_time,
	 metrics, total_batches)
VALUES
	(:trial_id, :trial_run_id, :state, now(),
	 :metrics, :total_batches)
`, model.TrialMetrics{
			TrialID:    int(m.TrialId),
			TrialRunID: int(m.TrialRunId),
			State:      model.CompletedState,
			Metrics: map[string]interface{}{
				"validation_metrics": m.Metrics,
			},
			TotalBatches: int(m.LatestBatch),
		}); err != nil {
			return errors.Wrap(err, "inserting validation metrics")
		}

		if err := setTrialBestValidation(tx, int(m.TrialId)); err != nil {
			return errors.Wrap(err, "updating trial best validation")
		}

		return nil
	})
}

// ensureStep inserts a noop step if no step exists at the batch index of the validation.
// This is used to make sure there is at least a dummy step for each validation or checkpoint,
// in the event one comes without (e.g. perform_initial_validation).
func (db *PgDB) ensureStep(
	ctx context.Context, tx *sqlx.Tx, trialID, trialRunID, latestBatch int,
) error {
	if _, err := tx.NamedExecContext(ctx, `
INSERT INTO raw_steps
	(trial_id, trial_run_id, state,
	 end_time, metrics, total_batches)
VALUES
	(:trial_id, :trial_run_id, :state,
	 :end_time, :metrics, :total_batches)
ON CONFLICT (trial_id, trial_run_id, total_batches)
DO NOTHING
`, model.TrialMetrics{
		TrialID:    trialID,
		TrialRunID: trialRunID,
		State:      model.CompletedState,
		EndTime:    ptrs.TimePtr(time.Now().UTC()),
		Metrics: map[string]interface{}{
			"avg_metrics":   struct{}{},
			"batch_metrics": []struct{}{},
		},
		TotalBatches: latestBatch,
	}); err != nil {
		return errors.Wrap(err, "inserting training metrics")
	}
	return nil
}

// AddCheckpointMetadata persists metadata for a completed checkpoint to the database.
func (db *PgDB) AddCheckpointMetadata(
	ctx context.Context, m *trialv1.CheckpointMetadata,
) error {
	return db.withTransaction("add checkpoint metadata", func(tx *sqlx.Tx) error {
		if err := checkTrialRunID(ctx, tx, m.TrialId, m.TrialRunId); err != nil {
			return err
		}

		if _, err := tx.ExecContext(ctx, `
UPDATE raw_checkpoints SET archived = true
WHERE trial_id = $1
  AND trial_run_id < $2
  AND total_batches >= $3;
`, m.TrialId, m.TrialRunId, m.LatestBatch); err != nil {
			return errors.Wrap(err, "archiving checkpoints")
		}

		if err := db.ensureStep(
			ctx, tx, int(m.TrialId), int(m.TrialRunId), int(m.LatestBatch),
		); err != nil {
			return err
		}

		if _, err := tx.NamedExecContext(ctx, `
INSERT INTO raw_checkpoints
	(trial_id, trial_run_id, state, end_time, total_batches,
	 uuid, resources, framework, format, determined_version)
VALUES
	(:trial_id, :trial_run_id, :state, now(), :total_batches,
	 :uuid, :resources, :framework, :format, :determined_version)
`, model.Checkpoint{
			TrialID:           int(m.TrialId),
			TrialRunID:        int(m.TrialRunId),
			State:             model.CompletedState,
			TotalBatches:      int(m.LatestBatch),
			UUID:              &m.Uuid,
			Resources:         model.JSONObjFromMapStringInt64(m.Resources),
			Framework:         m.Framework,
			Format:            m.Format,
			DeterminedVersion: m.DeterminedVersion,
		}); err != nil {
			return errors.Wrap(err, "inserting checkpoint metadata")
		}
		return nil
	})
}

func checkTrialRunID(ctx context.Context, tx *sqlx.Tx, trialID, runID int32) error {
	var cRunID int
	switch err := tx.QueryRowxContext(ctx, `
SELECT run_id
FROM trials
WHERE id = $1
`, trialID).Scan(&cRunID); {
	case err != nil:
		return errors.Wrap(err, "querying current run")
	case int(runID) != cRunID:
		return api.AsValidationError("invalid run id, %d (reported) != %d (expected)", runID, cRunID)
	default:
		return nil
	}
}

// ValidationByTotalBatches looks up a validation by trial and step ID,
// returning nil if none exists.
func (db *PgDB) ValidationByTotalBatches(trialID, totalBatches int) (*model.TrialMetrics, error) {
	var validation model.TrialMetrics
	if err := db.query(`
SELECT id, trial_id, total_batches, state, end_time, metrics
FROM validations
WHERE trial_id = $1
AND total_batches = $2`, &validation, trialID, totalBatches); errors.Cause(err) == ErrNotFound {
		return nil, nil
	} else if err != nil {
		return nil, errors.Wrapf(err, "error querying for validation (%v, %v)",
			trialID, totalBatches)
	}
	return &validation, nil
}

// CheckpointByTotalBatches looks up a checkpoint by trial and total batch,
// returning nil if none exists.
func (db *PgDB) CheckpointByTotalBatches(trialID, totalBatches int) (*model.Checkpoint, error) {
	var checkpoint model.Checkpoint
	if err := db.query(`
SELECT id, trial_id, total_batches, state, end_time, uuid, resources, metadata
FROM checkpoints
WHERE trial_id = $1
AND total_batches = $2`, &checkpoint, trialID, totalBatches); errors.Cause(err) == ErrNotFound {
		return nil, nil
	} else if err != nil {
		return nil, errors.Wrapf(err, "error querying for checkpoint (%v, %v)",
			trialID, totalBatches)
	}
	return &checkpoint, nil
}

// CheckpointByUUID looks up a checkpoint by UUID, returning nil if none exists.
func (db *PgDB) CheckpointByUUID(id uuid.UUID) (*model.Checkpoint, error) {
	var checkpoint model.Checkpoint
	if err := db.query(`
SELECT id, trial_id, total_batches, state, end_time, uuid, resources, metadata
FROM checkpoints
WHERE uuid = $1`, &checkpoint, id.String()); errors.Cause(err) == ErrNotFound {
		return nil, nil
	} else if err != nil {
		return nil, errors.Wrapf(err, "error querying for checkpoint (%v)", id.String())
	}
	return &checkpoint, nil
}

// LatestCheckpointForTrial finds the latest completed checkpoint for a trial, returning nil if
// none exists.
func (db *PgDB) LatestCheckpointForTrial(trialID int) (*model.Checkpoint, error) {
	var checkpoint model.Checkpoint
	if err := db.query(`
SELECT
	id, trial_id, state, end_time, uuid, resources, metadata, format,
	framework, determined_version, total_batches, trial_run_id
FROM checkpoints
WHERE trial_id = $1 AND state = 'COMPLETED'
ORDER BY total_batches DESC
LIMIT 1`, &checkpoint, trialID); errors.Cause(err) == ErrNotFound {
		return nil, nil
	} else if err != nil {
		return nil, errors.Wrapf(err, "error querying for latest trial checkpoint (%v)", trialID)
	}
	return &checkpoint, nil
}

// UpdateCheckpointMetadata updates an existing checkpoint with the metadata
// attached to the checkpoint passed into the method.
func (db *PgDB) UpdateCheckpointMetadata(checkpoint *model.Checkpoint) error {
	if checkpoint == nil {
		return errors.Errorf("checkpoint cannot be nil does not exist")
	}

	toUpdate := []string{"metadata"}

	err := db.namedExecOne(fmt.Sprintf(`
UPDATE checkpoints
%v
WHERE id = :id`, setClause(toUpdate)), checkpoint)
	if err != nil {
		return errors.Wrapf(err, "error updating (%v) in checkpoint (%v)",
			strings.Join(toUpdate, ", "), checkpoint.UUID)
	}
	return nil
}

// TrialState returns the current state of the given trial.
func (db *PgDB) TrialState(trialID int) (model.State, error) {
	var state model.State
	err := db.sql.QueryRow(`
SELECT state
FROM trials
WHERE id = $1
`, trialID).Scan(&state)
	return state, err
}

// TrialStatus returns the current status of the given trial, including the end time
// without returning all its hparams and other unneeded details. Called in paths hotter
// than TrialByID allows.
func (db *PgDB) TrialStatus(trialID int) (model.State, *time.Time, error) {
	status := struct {
		State   model.State `db:"state"`
		EndTime *time.Time  `db:"end_time"`
	}{}
	err := db.query(`
SELECT state, end_time
FROM trials
WHERE id = $1
`, &status, trialID)
	return status.State, status.EndTime, err
}

// setTrialBestValidation sets `public.trials.best_validation_id` to the `id` of the row in
// `public.validations` corresponding to the trial's best validation.
func setTrialBestValidation(tx *sqlx.Tx, id int) error {
	_, err := tx.Exec(`
WITH const AS (
    SELECT t.id as trial_id,
           config->'searcher'->>'metric' AS metric_name,
           (SELECT
               CASE WHEN coalesce((config->'searcher'->>'smaller_is_better')::boolean, true)
			   THEN 1
			   ELSE -1 END) AS sign
    FROM experiments e
    INNER JOIN trials t ON t.experiment_id = e.id
  	WHERE t.id = $1
), best_validation AS (
	SELECT
		v.id AS id,
		const.sign * (v.metrics->'validation_metrics'->>const.metric_name)::float8 AS metric
	FROM validations v, const
	WHERE v.trial_id = $1
	ORDER BY metric ASC
	LIMIT 1
)
UPDATE trials t
SET best_validation_id = (SELECT bv.id FROM best_validation bv)
WHERE t.id = $1;
`, id)
	return errors.Wrapf(err, "error updating best validation for trial %d", id)
}
