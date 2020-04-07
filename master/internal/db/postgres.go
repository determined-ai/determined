package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/golang-migrate/migrate"
	postgresM "github.com/golang-migrate/migrate/database/postgres"
	_ "github.com/golang-migrate/migrate/source/file" // Load migrations from files.
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // Use pq Postgres driver.
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/pkg/model"
)

// PgDB represents a Postgres database connection.  The type definition is needed to define methods.
type PgDB struct {
	sql *sqlx.DB
}

// ConnectPostgres connects to a Postgres database.
func ConnectPostgres(url string) (*PgDB, error) {
	numTries := 0
	for {
		sql, err := sqlx.Connect("postgres", url)
		if err == nil {
			return &PgDB{sql}, nil
		}

		numTries++
		if numTries >= 15 {
			return nil, errors.Wrapf(err, "could not connect to database after %v tries", numTries)
		}
		time.Sleep(4 * time.Second)
	}
}

const (
	// uniqueViolation is the error code that Postgres uses to indicate that an attempted insert/update
	// violates a uniqueness constraint.  Obtained from:
	// https://www.postgresql.org/docs/10/errcodes-appendix.html
	uniqueViolation = "23505"
)

// Migrate runs the migrations from the specified directory URL.
func (db *PgDB) Migrate(migrationURL string) error {
	driver, err := postgresM.WithInstance(db.sql.DB, &postgresM.Config{})
	if err != nil {
		return errors.Wrap(err, "error constructing Postgres migration driver")
	}
	m, err := migrate.NewWithDatabaseInstance(migrationURL, "postgres", driver)
	if err != nil {
		return errors.Wrapf(err, "error constructing Postgres migration using %s", migrationURL)
	}

	migrateVersion, _, merr := m.Version()
	if merr != nil {
		if merr != migrate.ErrNilVersion {
			return errors.Wrap(merr, "error loading golang-migrate version")
		}
		log.Info("unable to find golang-migrate version")
	} else {
		log.Infof("found golang-migrate version %v", migrateVersion)
	}

	if err = m.Up(); err != migrate.ErrNoChange {
		return errors.Wrap(err, "error applying migrations")
	}

	return nil
}

// Close closes the underlying pq connection.
func (db *PgDB) Close() error {
	return db.sql.Close()
}

// namedGet is a convenience method for a named query for a single value.
func (db *PgDB) namedGet(dest interface{}, query string, arg interface{}) error {
	nstmt, err := db.sql.PrepareNamed(query)
	if err != nil {
		return errors.Wrapf(err, "error preparing query %s", query)
	}
	if sErr := nstmt.QueryRowx(arg).Scan(dest); sErr != nil {
		err = errors.Wrapf(sErr, "error scanning query %s", query)
	}
	if cErr := nstmt.Close(); cErr != nil && err != nil {
		err = errors.Wrap(cErr, "error closing named DB statement")
	}

	return err
}

// namedExecOne is a convenience method for a NamedExec that should affect only one row.
func (db *PgDB) namedExecOne(query string, arg interface{}) error {
	res, err := db.sql.NamedExec(query, arg)
	if err != nil {
		return errors.Wrapf(err, "error in query %v \narg %v", query, arg)
	}
	num, err := res.RowsAffected()
	if err != nil {
		return errors.Wrapf(
			err,
			"error checking rows affected for query %v\n arg %v",
			query, arg)
	}
	if num != 1 {
		return errors.Errorf("error: %v rows affected on query %v \narg %v", num, query, arg)
	}
	return nil
}

func queryBinds(fields []string) []string {
	binds := make([]string, 0, len(fields))
	for _, field := range fields {
		binds = append(binds, ":"+field)
	}
	return binds
}

func setClause(fields []string) string {
	sets := make([]string, 0, len(fields))
	binds := queryBinds(fields)
	for i, field := range fields {
		sets = append(sets, fmt.Sprintf("%v = %v", field, binds[i]))
	}
	return fmt.Sprintf("SET\n%v", strings.Join(sets, ",\n"))
}

func (db *PgDB) rawQuery(q string, args ...interface{}) ([]byte, error) {
	var ret []byte
	if err := db.sql.QueryRowx(q, args...).Scan(&ret); err == sql.ErrNoRows {
		return nil, errors.WithStack(ErrNotFound)
	} else if err != nil {
		return nil, errors.WithStack(err)
	}
	return ret, nil
}

// query executes a query returning a single row and unmarshals the result into
// obj.
func (db *PgDB) query(q string, obj interface{}, args ...interface{}) error {
	if err := db.sql.QueryRowx(q, args...).StructScan(obj); err == sql.ErrNoRows {
		return errors.WithStack(ErrNotFound)
	} else if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

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

// AddExperiment adds the experiment to the database and sets its ID.
func (db *PgDB) AddExperiment(experiment *model.Experiment) error {
	if experiment.ID != 0 {
		return errors.Errorf("error adding an experiment with non-zero id %v", experiment.ID)
	}
	err := db.namedGet(&experiment.ID, `
INSERT INTO experiments
(state, config, model_definition, start_time, end_time, archived,
 git_remote, git_commit, git_committer, git_commit_date, owner_id)
VALUES (:state, :config, :model_definition, :start_time, :end_time, :archived,
        :git_remote, :git_commit, :git_committer, :git_commit_date, :owner_id)
RETURNING id`, experiment)
	if err != nil {
		return errors.Wrapf(err, "error inserting experiment %v", *experiment)
	}
	return nil
}

// ExperimentByID looks up an experiment by ID in a database, returning an error if none exists.
func (db *PgDB) ExperimentByID(id int) (*model.Experiment, error) {
	var experiment model.Experiment

	if err := db.query(`
SELECT id, state, config, model_definition, start_time, end_time, archived,
       git_remote, git_commit, git_committer, git_commit_date, owner_id
FROM experiments
WHERE id = $1`, &experiment, id); err != nil {
		return nil, err
	}

	return &experiment, nil
}

// ExperimentByTrialID looks up an experiment by a trial ID in the
// database, returning an error if the experiment doesn't exist.
func (db *PgDB) ExperimentByTrialID(id int) (*model.Experiment, error) {
	experiment := model.Experiment{}
	return &experiment, db.sql.QueryRowx(`
SELECT e.id, e.state, e.config, e.model_definition, e.start_time, e.end_time,
e.archived, e.git_remote, e.git_commit, e.git_committer, e.git_commit_date
FROM experiments e, trials t  WHERE t.id = $1 AND e.id = t.experiment_id`,
		id).StructScan(&experiment)
}

// NonTerminalExperiments finds all experiments in the database whose states are not terminal.
func (db *PgDB) NonTerminalExperiments() ([]*model.Experiment, error) {
	rows, err := db.sql.Queryx(`
SELECT id, state, config, model_definition, start_time, end_time, archived,
       git_remote, git_commit, git_committer, git_commit_date, owner_id
FROM experiments
WHERE state IN ('ACTIVE', 'PAUSED', 'STOPPING_CANCELED', 'STOPPING_COMPLETED', 'STOPPING_ERROR')`)
	if err == sql.ErrNoRows {
		return nil, errors.WithStack(ErrNotFound)
	} else if err != nil {
		return nil, errors.Wrap(err, "querying for active experiments")
	}

	defer rows.Close()

	var exps []*model.Experiment
	for rows.Next() {
		var exp model.Experiment
		if err = rows.StructScan(&exp); err != nil {
			return nil, errors.Wrap(err, "reading experiments")
		}
		exps = append(exps, &exp)
	}
	return exps, nil
}

// SaveExperimentConfig saves the current experiment config to the database.
func (db *PgDB) SaveExperimentConfig(experiment *model.Experiment) error {
	query := `
UPDATE experiments
SET config=:config
WHERE id = :id`
	return db.namedExecOne(query, experiment)
}

// SaveExperimentState saves the current experiment state to the database.
func (db *PgDB) SaveExperimentState(experiment *model.Experiment) error {
	query := `
UPDATE experiments
SET state=:state, end_time=:end_time
WHERE id = :id`
	return db.namedExecOne(query, experiment)
}

// SaveExperimentArchiveStatus saves the current experiment archive status to the database.
func (db *PgDB) SaveExperimentArchiveStatus(experiment *model.Experiment) error {
	if !model.TerminalStates[experiment.State] {
		return errors.Errorf("cannot set archived for experiment in state %v", experiment.State)
	}

	query := `
UPDATE experiments
SET archived=:archived
WHERE id = :id`
	return db.namedExecOne(query, experiment)
}

// DeleteExperiment deletes an existing experiment.
func (db *PgDB) DeleteExperiment(id int) error {
	tx, err := db.sql.Begin()
	if err != nil {
		return errors.Wrap(err, "error starting transaction")
	}
	defer func() {
		if tx == nil {
			return
		}

		if rErr := tx.Rollback(); rErr != nil {
			log.Errorf("error during rollback: %v", rErr)
		}
	}()

	// This delete cascades to checkpoints and validations.
	_, err = tx.Exec(`
DELETE FROM steps
WHERE trial_id IN (SELECT id FROM trials WHERE experiment_id = $1)
`, id)
	if err != nil {
		return errors.Wrapf(err, "error deleting steps for experiment %v", id)
	}
	_, err = tx.Exec(`
DELETE FROM trial_logs
WHERE trial_id IN (SELECT id FROM trials WHERE experiment_id = $1);
`, id)
	if err != nil {
		return errors.Wrapf(err, "error deleting trial logs for experiment %v", id)
	}
	_, err = tx.Exec(`
DELETE FROM trials
WHERE experiment_id = $1;
`, id)
	if err != nil {
		return errors.Wrapf(err, "error deleting trials for experiment %v", id)
	}
	_, err = tx.Exec(`
DELETE FROM searcher_events
WHERE experiment_id = $1;
`, id)
	if err != nil {
		return errors.Wrapf(err, "error deleting events for experiment %v", id)
	}
	result, err := tx.Exec(`
DELETE FROM experiments
WHERE id = $1
`, id)
	if err != nil {
		return errors.Wrapf(err, "error deleting experiment %v", id)
	}
	num, err := result.RowsAffected()
	if err != nil {
		return errors.Wrapf(err, "error in RowsAffected when deleting experiment %v", id)
	}
	if num != 1 {
		return errors.Errorf("error deleting non-existing experiment %v", id)
	}
	err = tx.Commit()
	if err != nil {
		return errors.Wrapf(err, "error committing delete from experiment %v", id)
	}

	tx = nil

	return nil
}

// SaveExperimentProgress stores the progress for an experiment in the database.
func (db *PgDB) SaveExperimentProgress(id int, progress *float64) error {
	res, err := db.sql.Exec(`UPDATE experiments SET progress = $1 WHERE id = $2`, progress, id)
	if err != nil {
		return errors.Wrap(err, "saving experiment progress")
	}
	if numRows, err := res.RowsAffected(); err != nil {
		return errors.Wrap(err, "checking affected rows for saving experiment progress")
	} else if numRows != 1 {
		return errors.Errorf("saving experiment %d's progress affected %d rows instead of 1", id, numRows)
	}
	return nil
}

// ForEachSearcherEvent calls a callback for each searcher event of an experiment.
func (db *PgDB) ForEachSearcherEvent(id int, callback func(model.SearcherEvent) error) error {
	rows, err := db.sql.Queryx(`
SELECT id, experiment_id, event_type, content
FROM searcher_events
WHERE experiment_id = $1
ORDER BY id ASC`, id)
	if err == sql.ErrNoRows {
		return errors.WithStack(ErrNotFound)
	} else if err != nil {
		return errors.Wrapf(err, "querying for searcher events of experiment %v", id)
	}

	defer rows.Close()

	for rows.Next() {
		var event model.SearcherEvent

		if err = rows.StructScan(&event); err != nil {
			return errors.Wrapf(err, "scanning for event in row for experiment %v", id)
		}

		if err = callback(event); err != nil {
			return errors.Wrapf(err, "running searcher event callback for experiment %v", id)
		}
	}
	return nil
}

// ExperimentConfig returns the full config object for an experiment.
func (db *PgDB) ExperimentConfig(id int) (*model.ExperimentConfig, error) {
	expConfigBytes, err := db.rawQuery(`
SELECT config
FROM experiments
WHERE id = $1`, id)
	if err != nil {
		return nil, err
	}
	var expConfig model.ExperimentConfig
	if err = json.Unmarshal(expConfigBytes, &expConfig); err != nil {
		return nil, errors.WithStack(err)
	}
	return &expConfig, nil
}

// ExperimentTotalStepTime returns the total time for all steps of the experiment with the given ID
// (i.e., the sum of start_time - end_time). Any step with a NULL end_time does not contribute.
func (db *PgDB) ExperimentTotalStepTime(id int) (time.Duration, error) {
	var seconds float64
	if err := db.sql.Get(&seconds, `
SELECT coalesce(extract(epoch FROM sum(steps.end_time - steps.start_time)), 0)
FROM steps, trials
WHERE trials.experiment_id = $1 AND steps.trial_id = trials.id
`, id); err != nil {
		return 0, errors.Wrapf(err, "querying for experiment total step time")
	}
	return time.Duration(seconds * float64(time.Second)), nil
}

// ExperimentModelDefinitionRaw returns the zipped model definition for an experiment as a byte
// array.
func (db *PgDB) ExperimentModelDefinitionRaw(id int) ([]byte, error) {
	return db.rawQuery(`
SELECT model_definition
FROM experiments
WHERE id = $1`, id)
}

// ExperimentCheckpointsToGCRaw returns a JSON string describing checkpoints that should be GCed
// according to the given GC policy parameters. If the delete parameter is true, the returned
// checkpoints are also marked as deleted in the database.
func (db *PgDB) ExperimentCheckpointsToGCRaw(
	id int,
	experimentBest, trialBest, trialLatest *int,
	delete bool,
) ([]byte, error) {
	// The string for the CTEs that we need whether or not we're not deleting the results. The
	// "selected_checkpoints" table contains the checkpoints to return as rows, so that we can easily
	// set the corresponding checkpoints to deleted in a separate CTE if we're deleting.
	ctes := `
WITH const AS (
    SELECT config->'searcher'->>'metric' AS metric_name,
           (CASE
                WHEN coalesce((config->'searcher'->>'smaller_is_better')::boolean, true)
                THEN 1
                ELSE -1
            END) AS sign,
           coalesce($2, (config->'checkpoint_storage'->>'save_experiment_best')::int)
               AS experiment_best,
           coalesce($3, (config->'checkpoint_storage'->>'save_trial_best')::int)
               AS trial_best,
           coalesce($4, (config->'checkpoint_storage'->>'save_trial_latest')::int)
               AS trial_latest
    FROM experiments WHERE id = $1
), selected_checkpoints AS (
    SELECT *
    FROM (
        SELECT *,
               -- The order includes the id to prevent different rows from having the same
               -- rank, which could cause more than the desired number of checkpoints to be
               -- left out of the result set. Also, any rows with null validation values
               -- will sort to the end, thereby not affecting the ranks of rows with
               -- non-null validations, and will be filtered out later.
               rank() OVER (
                   ORDER BY
                       const.sign * (step->'validation'->'metrics'->'validation_metrics'
                                     ->>const.metric_name)::float8 ASC NULLS LAST, id ASC
               ) AS experiment_rank,
               rank() OVER (
                   PARTITION BY trial_id
                   ORDER BY
                       const.sign * (step->'validation'->'metrics'->'validation_metrics'
                                     ->>const.metric_name)::float8 ASC NULLS LAST, id ASC
               ) AS trial_rank,
               rank() OVER (
                   PARTITION BY trial_id
                   ORDER BY step_id DESC
               ) AS trial_order_rank
        FROM (
            SELECT c.id, c.trial_id, c.step_id, c.state, c.start_time, c.end_time, c.uuid,
                   c.resources, c.labels,
                   (SELECT row_to_json(s)
                    FROM (
                        SELECT s.end_time, s.id, s.start_time, s.state, s.trial_id,
                               (SELECT row_to_json(v)
                                FROM (
                                    SELECT v.end_time, v.id, v.metrics, v.start_time,
                                           v.state, v.step_id, v.trial_id
                                    FROM validations v
                                    WHERE v.trial_id = t.id AND v.step_id = s.id
                                ) v
                               ) AS validation
                        FROM steps s
                        WHERE s.id = c.step_id AND s.trial_id = c.trial_id
                    ) s
                   ) AS step,
                   -- We later filter out any checkpoints with any corresponding warm start
                   -- trials, so we can just put an empty list here. (TODO(dzhu): This is
                   -- here for backwards compatibility with Python, but could maybe be
                   -- removed.)
                   '[]'::jsonb AS warm_start_trials
            FROM checkpoints c, trials t, const
            WHERE c.state = 'COMPLETED' AND c.trial_id = t.id AND t.experiment_id = $1
        ) _, const
    ) c, const
    WHERE (const.experiment_best IS NOT NULL
               OR const.trial_best IS NOT NULL
               OR const.trial_latest IS NOT NULL)
          AND (SELECT COUNT(*) FROM trials t WHERE t.warm_start_checkpoint_id = c.id) = 0
          AND c.trial_order_rank > const.trial_latest
          AND ((c.experiment_rank > const.experiment_best
                AND c.trial_rank > const.trial_best)
               OR (c.step->'validation'->'metrics'->'validation_metrics'->>const.metric_name
                   IS NULL))
)`

	if delete {
		ctes += `, do_delete AS (
    UPDATE checkpoints
    SET state = 'DELETED'
    FROM selected_checkpoints
    WHERE checkpoints.id = selected_checkpoints.id
)
`
	}

	query := `
SELECT row_to_json(x)
FROM (
    SELECT const.metric_name,
           (SELECT coalesce(
                       jsonb_agg(to_jsonb(selected_checkpoints.*)
                           #- '{experiment_rank}' #- '{trial_rank}' #- '{trial_order_rank}'
                       ORDER BY id ASC), '[]'::jsonb)
            FROM selected_checkpoints
           ) AS checkpoints
    FROM const
) x
`

	return db.rawQuery(ctes+query, id, experimentBest, trialBest, trialLatest)
}

// AddTrial adds the trial to the database and sets its ID.
func (db *PgDB) AddTrial(trial *model.Trial) error {
	if trial.ID != 0 {
		return errors.Errorf("error adding a trial with non-zero id %v", trial.ID)
	}
	// Assume the foreign key constraint is handled by the database.
	err := db.namedGet(&trial.ID, `
INSERT INTO trials
(experiment_id, state, start_time, end_time, hparams, warm_start_checkpoint_id, seed)
VALUES (:experiment_id, :state, :start_time, :end_time, :hparams, :warm_start_checkpoint_id, :seed)
RETURNING id`, trial)
	if err != nil {
		return errors.Wrapf(err, "error inserting trial %v", *trial)
	}
	return nil
}

// TrialByID looks up a trial by ID, returning an error if none exists.
func (db *PgDB) TrialByID(id int) (*model.Trial, error) {
	trial := model.Trial{}
	if err := db.query(`
SELECT id, experiment_id, state, start_time, end_time, hparams, warm_start_checkpoint_id, seed
FROM trials
WHERE id = $1`, &trial, id); err != nil {
		return nil, errors.Wrapf(err, "error querying for trial %v", id)
	}
	return &trial, nil
}

// UpdateTrial updates an existing trial. Fields that are nil or zero are not
// updated.  end_time is set if the trial moves to a terminal state.
func (db *PgDB) UpdateTrial(id int, newState model.State) error {
	if len(newState) == 0 {
		return nil
	}
	trial, err := db.TrialByID(id)
	if err != nil {
		return errors.Wrapf(err, "error finding trial %v to update", id)
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
	err = db.namedExecOne(fmt.Sprintf(`
UPDATE trials
%v
WHERE id = :id`, setClause(toUpdate)), trial)
	if err != nil {
		return errors.Wrapf(err, "error updating (%v) in trial %v",
			strings.Join(toUpdate, ", "), id)
	}
	return nil
}

// RollBackTrial deletes from the database all steps, checkpoints, and validations for the trial
// that correspond to steps past lastStep.
func (db *PgDB) RollBackTrial(id int, lastStep int) error {
	// This delete cascades to checkpoints and validations.
	_, err := db.sql.Exec(`
DELETE FROM steps
WHERE trial_id = $1 AND id > $2
`, id, lastStep)
	if err != nil {
		return errors.Wrapf(err, "error rolling back trial %v to step %v", id, lastStep)
	}
	return nil
}

// AddTrialLogs adds a list of *model.TrialLog objects to the database with automatic IDs.
func (db *PgDB) AddTrialLogs(logs []*model.TrialLog) error {
	if len(logs) == 0 {
		return nil
	}

	var text strings.Builder
	text.WriteString("INSERT INTO trial_logs (trial_id, message) VALUES")

	args := make([]interface{}, 0, len(logs)*2)

	for i, log := range logs {
		// Add an argument to the SQL statement of the form: ($1, $2)
		if i > 0 {
			text.WriteString(",")
		}
		fmt.Fprintf(&text, " ($%d, $%d)", i*2+1, i*2+2)

		args = append(args, log.TrialID, model.RawString(log.Message))
	}

	if _, err := db.sql.Exec(text.String(), args...); err != nil {
		return errors.Wrapf(err, "error inserting %d trial logs", len(logs))
	}

	return nil
}

// TrialLogsRaw returns the logs for a trial as a JSON string. TODO(dzhu): With GraphQL, this should
// now only be used for raw log text downloads; the query can be simplified accordingly.
func (db *PgDB) TrialLogsRaw(
	id int,
	greaterThan, lessThan *int,
	limit *int,
) ([]*model.LogMessage, error) {
	innerQuery := `
SELECT id, message
FROM trial_logs
WHERE trial_id = $1 AND (id > $2 OR $2 IS NULL) AND (id < $3 OR $3 IS NULL)
`
	var rows *sqlx.Rows
	var err error

	if limit != nil {
		rows, err = db.sql.Queryx(fmt.Sprintf(`
SELECT * FROM (
	%s
	ORDER BY id DESC LIMIT $4
) r2
ORDER BY id ASC`, innerQuery), id, greaterThan, lessThan, *limit)
	} else {
		rows, err = db.sql.Queryx(fmt.Sprintf(`
%s
ORDER BY id ASC
`, innerQuery), id, greaterThan, lessThan)
	}

	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, errors.Wrap(err, "querying trial logs")
	}
	defer rows.Close()

	var logs []*model.LogMessage
	for rows.Next() {
		var msg model.LogMessage
		if err = rows.StructScan(&msg); err != nil {
			return nil, errors.Wrap(err, "scanning row")
		}
		logs = append(logs, &msg)
	}

	return logs, nil
}

// AddStep adds the step to the database.
func (db *PgDB) AddStep(step *model.Step) error {
	if !step.IsNew() {
		return errors.Errorf("unexpected state for new step: %v", step)
	}
	trial, err := db.TrialByID(step.TrialID)
	if err != nil {
		return errors.Wrapf(err, "error finding trial %v for new step", step.TrialID)
	}
	if trial.State != model.ActiveState {
		return errors.Errorf("can't add step to trial %v with state %v", trial.ID, trial.State)
	}
	err = db.namedExecOne(`
INSERT INTO steps
(trial_id, id, state, start_time, end_time)
VALUES (:trial_id, :id, :state, :start_time, :end_time)`, step)
	if err != nil {
		return errors.Wrapf(err, "error inserting step %v", *step)
	}
	return nil
}

// StepByID looks up a step by (TrialID, StepID) pair, returning an error if none exists.
func (db *PgDB) StepByID(trialID, stepID int) (*model.Step, error) {
	var step model.Step
	if err := db.query(`
SELECT trial_id, id, state, start_time, end_time, metrics
FROM steps
WHERE trial_id = $1 AND id = $2`, &step, trialID, stepID); err != nil {
		return nil, errors.Wrapf(err, "error querying for step %v, %v", trialID, stepID)
	}
	return &step, nil
}

// UpdateStep updates an existing step. Fields that are nil or zero are not
// updated.  end_time is set if the step moves to a terminal state.
func (db *PgDB) UpdateStep(
	trialID, stepID int, newState model.State, metrics model.JSONObj) error {
	if len(newState) == 0 && len(metrics) == 0 {
		return nil
	}
	step, err := db.StepByID(trialID, stepID)
	if err != nil {
		return errors.Wrapf(err, "error finding step (%v, %v) to update", trialID, stepID)
	}
	toUpdate := []string{}
	if len(newState) != 0 {
		if !model.StepTransitions[step.State][newState] {
			return errors.Errorf("illegal transition %v -> %v for step (%v, %v)",
				step.State, newState, step.TrialID, step.ID)
		}
		step.State = newState
		toUpdate = append(toUpdate, "state")
		if model.TerminalStates[newState] {
			now := time.Now().UTC()
			step.EndTime = &now
			toUpdate = append(toUpdate, "end_time")
		}
	}
	if len(metrics) != 0 {
		if len(step.Metrics) != 0 {
			return errors.Errorf("step (%v, %v) already has metrics", trialID, stepID)
		}
		step.Metrics = metrics
		toUpdate = append(toUpdate, "metrics")
	}
	err = db.namedExecOne(fmt.Sprintf(`
UPDATE steps
%v
WHERE trial_id = :trial_id
AND id = :id`, setClause(toUpdate)), step)
	if err != nil {
		return errors.Wrapf(err, "error updating (%v) in step (%v, %v)",
			strings.Join(toUpdate, ", "), step.TrialID, step.ID)
	}
	return nil
}

// AddValidation adds the validation to the database and sets its ID.
func (db *PgDB) AddValidation(validation *model.Validation) error {
	if !validation.IsNew() {
		return errors.Errorf("unexpected state for new validation: %v", validation)
	}
	trial, err := db.TrialByID(validation.TrialID)
	if err != nil {
		return errors.Wrapf(err, "error finding trial %v for new validation", validation.TrialID)
	}
	if trial.State != model.ActiveState {
		return errors.Errorf("can't add validation to trial %v with state %v", trial.ID, trial.State)
	}
	step, err := db.StepByID(validation.TrialID, validation.StepID)
	if err != nil {
		return errors.Wrapf(err,
			"error finding step (%v, %v) to add validation", validation.TrialID, validation.StepID)
	}
	if step.State != model.CompletedState {
		return errors.Errorf("unexpected state %v for trial %v step %v",
			step.State, validation.TrialID, validation.StepID)
	}
	var count int
	err = db.namedGet(&count, `
SELECT COUNT(*)
FROM validations
WHERE trial_id = :trial_id
AND step_id = :step_id`, validation)
	if err != nil {
		return errors.Wrapf(err, "error checking at-most-one validation %v", *validation)
	}
	if count > 0 {
		return errors.Errorf("duplicate validation for trial %v step %v",
			validation.TrialID, validation.StepID)
	}
	err = db.namedGet(&validation.ID, `
INSERT INTO validations
(trial_id, step_id, state, start_time, end_time)
VALUES (:trial_id, :step_id, :state, :start_time, :end_time)
RETURNING id`, validation)
	if err != nil {
		return errors.Wrapf(err, "error inserting validation %v", *validation)
	}
	return nil
}

// ValidationByStep looks up a validation by trial and step ID, returning nil if none exists.
func (db *PgDB) ValidationByStep(trialID, stepID int) (*model.Validation, error) {
	var validation model.Validation
	if err := db.query(`
SELECT id, trial_id, step_id, state, start_time, end_time, metrics
FROM validations
WHERE trial_id = $1
AND step_id = $2`, &validation, trialID, stepID); errors.Cause(err) == ErrNotFound {
		return nil, nil
	} else if err != nil {
		return nil, errors.Wrapf(err, "error querying for validation (%v, %v)",
			trialID, stepID)
	}
	return &validation, nil
}

// UpdateValidation updates an existing validation. Fields that are nil or zero
// are not updated. end_time is set if the validation moves to a terminal
// state.
func (db *PgDB) UpdateValidation(trialID, stepID int, newState model.State, metrics model.JSONObj,
) error {
	if len(newState) == 0 && len(metrics) == 0 {
		return nil
	}
	validation, err := db.ValidationByStep(trialID, stepID)
	if err != nil {
		return errors.Wrapf(err, "error querying for validation (%v, %v) to update",
			trialID, stepID)
	}
	if validation == nil {
		return errors.Wrapf(err, "can't update missing validation (%v, %v)",
			trialID, stepID)
	}
	toUpdate := []string{}
	if len(newState) != 0 {
		if !model.StepTransitions[validation.State][newState] {
			return errors.Errorf("illegal transition %v -> %v for validation %v",
				validation.State, newState, validation.ID)
		}
		validation.State = newState
		toUpdate = append(toUpdate, "state")
		if model.TerminalStates[newState] {
			now := time.Now().UTC()
			validation.EndTime = &now
			toUpdate = append(toUpdate, "end_time")
		}
	}
	if len(metrics) != 0 {
		if len(validation.Metrics) != 0 {
			return errors.Errorf("validation (%v, %v) already has metrics",
				trialID, stepID)
		}
		validation.Metrics = metrics
		toUpdate = append(toUpdate, "metrics")
	}
	err = db.namedExecOne(fmt.Sprintf(`
UPDATE validations
%v
WHERE id = :id`, setClause(toUpdate)), validation)
	if err != nil {
		return errors.Wrapf(err, "error updating (%v) in validation (%v, %v)",
			strings.Join(toUpdate, ", "), trialID, stepID)
	}
	return nil
}

// AddCheckpoint adds the checkpoint to the database and sets its ID.
func (db *PgDB) AddCheckpoint(checkpoint *model.Checkpoint) error {
	if !checkpoint.IsNew() {
		return errors.Errorf("unexpected state for new checkpoint: %v", checkpoint)
	}
	step, err := db.StepByID(checkpoint.TrialID, checkpoint.StepID)
	if err != nil {
		return errors.Wrapf(err,
			"error finding step (%v, %v) for new checkpoint", checkpoint.TrialID, checkpoint.StepID)
	}
	if step.State != model.CompletedState {
		return errors.Errorf("unexpected state %v for trial %v step %v",
			step.State, checkpoint.TrialID, checkpoint.StepID)
	}
	var count int
	err = db.namedGet(&count, `
SELECT COUNT(*)
FROM checkpoints
WHERE trial_id = :trial_id
AND step_id = :step_id`, checkpoint)
	if err != nil {
		return errors.Wrapf(err, "error checking at-most-one checkpoint %v", *checkpoint)
	}
	if count > 0 {
		return errors.Errorf("duplicate checkpoint for trial %v step %v",
			checkpoint.TrialID, checkpoint.StepID)
	}
	err = db.namedGet(&checkpoint.ID, `
INSERT INTO checkpoints
(trial_id, step_id, state, start_time)
VALUES (:trial_id, :step_id, :state, :start_time)
RETURNING id`, checkpoint)
	if err != nil {
		return errors.Wrapf(err, "error inserting checkpoint %v", *checkpoint)
	}
	return nil
}

// CheckpointByStep looks up a checkpoint by trial and step ID, returning nil if none exists.
func (db *PgDB) CheckpointByStep(trialID, stepID int) (*model.Checkpoint, error) {
	var checkpoint model.Checkpoint
	if err := db.query(`
SELECT id, trial_id, step_id, state, start_time, end_time, uuid, resources, labels
FROM checkpoints
WHERE trial_id = $1
AND step_id = $2`, &checkpoint, trialID, stepID); errors.Cause(err) == ErrNotFound {
		return nil, nil
	} else if err != nil {
		return nil, errors.Wrapf(err, "error querying for checkpoint (%v, %v)",
			trialID, stepID)
	}
	return &checkpoint, nil
}

// CheckpointByUUID looks up a checkpoint by UUID, returning nil if none exists.
func (db *PgDB) CheckpointByUUID(id uuid.UUID) (*model.Checkpoint, error) {
	var checkpoint model.Checkpoint
	if err := db.query(`
SELECT id, trial_id, step_id, state, start_time, end_time, uuid, resources, labels
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
SELECT id, trial_id, step_id, state, start_time, end_time, uuid, resources, labels
FROM checkpoints
WHERE trial_id = $1 AND state = 'COMPLETED'
ORDER BY step_id DESC
LIMIT 1`, &checkpoint, trialID); errors.Cause(err) == ErrNotFound {
		return nil, nil
	} else if err != nil {
		return nil, errors.Wrapf(err, "error querying for latest trial checkpoint (%v)", trialID)
	}
	return &checkpoint, nil
}

// UpdateCheckpoint updates an existing checkpoint. Fields that are nil or zero
// are not updated. end_time is set if the checkpoint moves to a terminal
// state.
func (db *PgDB) UpdateCheckpoint(
	trialID, stepID int,
	newState model.State,
	uuid string,
	resources model.JSONObj,
	labels model.JSONObj,
) error {
	if len(newState) == 0 && len(uuid) == 0 && len(resources) == 0 && len(labels) == 0 {
		return nil
	}

	checkpoint, err := db.CheckpointByStep(trialID, stepID)
	if err != nil {
		return errors.Wrapf(err, "error querying for checkpoint (%v, %v) to update",
			trialID, stepID)
	}
	if checkpoint == nil {
		return errors.Wrapf(err, "can't update missing checkpoint (%v, %v)",
			trialID, stepID)
	}

	toUpdate := []string{}
	if len(newState) != 0 {
		if !model.CheckpointTransitions[checkpoint.State][newState] {
			return errors.Errorf("illegal transition %v -> %v for checkpoint %v",
				checkpoint.State, newState, checkpoint.ID)
		}
		checkpoint.State = newState
		toUpdate = append(toUpdate, "state")
		if model.TerminalStates[newState] {
			now := time.Now().UTC()
			checkpoint.EndTime = &now
			toUpdate = append(toUpdate, "end_time")
		}
	}
	if len(uuid) != 0 {
		if checkpoint.UUID != nil && len(*checkpoint.UUID) != 0 {
			return errors.Errorf("checkpoint (%v, %v) already has UUID",
				trialID, stepID)
		}
		checkpoint.UUID = &uuid
		toUpdate = append(toUpdate, "uuid")
	}
	if len(resources) != 0 {
		if len(checkpoint.Resources) != 0 {
			return errors.Errorf("checkpoint (%v, %v) already has resources",
				trialID, stepID)
		}
		checkpoint.Resources = resources
		toUpdate = append(toUpdate, "resources")
	}
	if len(labels) != 0 {
		if len(checkpoint.Labels) != 0 {
			return errors.Errorf("checkpoint (%v, %v) already has labels",
				trialID, stepID)
		}
		checkpoint.Labels = labels
		toUpdate = append(toUpdate, "labels")
	}
	err = db.namedExecOne(fmt.Sprintf(`
UPDATE checkpoints
%v
WHERE id = :id`, setClause(toUpdate)), checkpoint)
	if err != nil {
		return errors.Wrapf(err, "error updating (%v) in checkpoint (%v, %v)",
			strings.Join(toUpdate, ", "), trialID, stepID)
	}
	return nil
}

// AddSearcherEvents adds the searcher events to the database.
func (db *PgDB) AddSearcherEvents(events []*model.SearcherEvent) error {
	if len(events) == 0 {
		return nil
	}

	var text strings.Builder
	_, _ = text.WriteString(
		"INSERT INTO searcher_events (experiment_id, event_type, content) VALUES",
	)

	args := make([]interface{}, 0, len(events)*3)

	for i, event := range events {
		// Add an argument to the SQL statement of the form: ($1, $2, $3)
		if i > 0 {
			_, _ = text.WriteString(",")
		}
		_, _ = text.WriteString(" ($")
		_, _ = text.WriteString(strconv.Itoa(i*3 + 1))
		_, _ = text.WriteString(", $")
		_, _ = text.WriteString(strconv.Itoa(i*3 + 2))
		_, _ = text.WriteString(", $")
		_, _ = text.WriteString(strconv.Itoa(i*3 + 3))
		_, _ = text.WriteString(")")

		args = append(args, event.ExperimentID)
		args = append(args, event.EventType)
		args = append(args, event.Content)
	}

	if _, err := db.sql.Exec(text.String(), args...); err != nil {
		return errors.Wrapf(err, "error inserting %d searcher events", len(events))
	}

	return nil
}

// DeleteSearcherEvents deletes all searcher events for a specific experiment from the database.
func (db *PgDB) DeleteSearcherEvents(expID int) error {
	res, err := db.sql.Exec("DELETE FROM searcher_events WHERE experiment_id = $1", expID)
	if err != nil {
		return errors.Wrapf(err, "error in deleting searcher events for experiment %v", expID)
	}

	num, err := res.RowsAffected()
	if err != nil {
		log.Errorf(
			"RowsAffected failed in deleting searcher events for experiment %v, error: %v", expID, err)
		return nil
	}
	log.Debugf("deleted total %v searcher events for experiment %v", num, expID)
	return nil
}

// DeleteSearcherEventsForTerminalStateExperiments deletes all searcher events for
// terminal state experiments from the database. This is used to clean up searcher
// events if master crashes before deleting searcher events.
func (db *PgDB) DeleteSearcherEventsForTerminalStateExperiments() error {
	res, err := db.sql.Exec(`
DELETE FROM searcher_events
WHERE experiment_id IN (
	SELECT id
	FROM experiments
	WHERE state IN ('COMPLETED', 'CANCELED', 'ERROR'))`)
	if err != nil {
		return err
	}

	num, err := res.RowsAffected()
	if err != nil {
		log.Errorf(
			"RowsAffected failed in deleting searcher events for terminal state experiments. error: %v", err)
		return nil
	}
	log.Debugf("deleted total %v searcher events for terminal state experiments", num)
	return nil
}

// TelemetryInformation returns an anonymous summary of information about the usage of Determined.
func (db *PgDB) TelemetryInformation() ([]byte, error) {
	return db.rawQuery(`
SELECT jsonb_build_object(
    'num_users', (SELECT count(*) FROM users),
    'num_experiments', (SELECT count(*) FROM experiments),
    'num_trials', (SELECT count(*) FROM trials),
    'experiment_aggregates', jsonb_build_object(
        'state', frequencies(array(SELECT state FROM experiments)),
        'cpu_image', frequencies(array(SELECT config->'environment'->'image'->'cpu'
                                       FROM experiments)),
        'gpu_image', frequencies(array(SELECT config->'environment'->'image'->'gpu'
                                       FROM experiments)),
        'num_trials', frequencies(array(SELECT count(*)
                                        FROM trials GROUP BY experiment_id)),
        'num_hparams', frequencies(array(SELECT (SELECT count(*)
                                                 FROM jsonb_object_keys(config->'hyperparameters'))
                                         FROM experiments)),
        'batches_per_step', frequencies(array(SELECT config->'batches_per_step' FROM experiments)),
        'slots_per_trial', frequencies(array(SELECT config->'resources'->'slots_per_trial'
                                             FROM experiments)),
        'searcher_name', frequencies(array(SELECT config->'searcher'->'name' FROM experiments))
    ),
    'trial_aggregates', jsonb_build_object(
        'state', frequencies(array(SELECT state FROM trials)),
        'completed_steps', frequencies(array(SELECT count(*) FROM steps
                                             WHERE state = 'COMPLETED' GROUP BY trial_id))
    )
);
`)
}

// AddAuthTokenKeypair adds the new auth token keypair.
func (db *PgDB) AddAuthTokenKeypair(tokenKeypair *model.AuthTokenKeypair) error {
	return db.namedExecOne(`
INSERT INTO auth_token_keypair (public_key, private_key)
VALUES (:public_key, :private_key)`, *tokenKeypair)
}

// AuthTokenKeypair gets the existing auth token keypair.
func (db *PgDB) AuthTokenKeypair() (*model.AuthTokenKeypair, error) {
	var tokenKeypair model.AuthTokenKeypair
	switch err := db.query("SELECT * FROM auth_token_keypair", &tokenKeypair); {
	case errors.Cause(err) == ErrNotFound:
		return nil, nil
	case err != nil:
		return nil, err
	default:
		return &tokenKeypair, nil
	}
}
