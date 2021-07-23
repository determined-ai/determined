package db

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/golang-migrate/migrate"
	postgresM "github.com/golang-migrate/migrate/database/postgres"
	_ "github.com/golang-migrate/migrate/source/file" // Load migrations from files.
	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v4/stdlib" // Import Postgres driver.
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/schemas"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/proto/pkg/trialv1"
)

// PgDB represents a Postgres database connection.  The type definition is needed to define methods.
type PgDB struct {
	tokenKeys *model.AuthTokenKeypair
	sql       *sqlx.DB
	queries   *staticQueryMap
}

// ConnectPostgres connects to a Postgres database.
func ConnectPostgres(url string) (*PgDB, error) {
	numTries := 0
	for {
		sql, err := sqlx.Connect("pgx", url)
		if err == nil {
			return &PgDB{sql: sql, queries: &staticQueryMap{queries: make(map[string]string)}}, err
		}
		numTries++
		if numTries >= 15 {
			return nil, errors.Wrapf(err, "could not connect to database after %v tries", numTries)
		}
		toWait := 4 * time.Second
		time.Sleep(toWait)
		log.WithError(err).Warnf("failed to connect to postgres, trying again in %s", toWait)
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
	log.Infof("running DB migrations from %s; this might take a while...", migrationURL)
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

	if err = m.Up(); err != nil && err != migrate.ErrNoChange {
		return errors.Wrap(err, "error applying migrations")
	}
	log.Info("DB migrations completed")
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

// query executes a query returning a single row and unmarshals the result into an obj.
func (db *PgDB) query(q string, obj interface{}, args ...interface{}) error {
	if err := db.sql.QueryRowx(q, args...).StructScan(obj); err == sql.ErrNoRows {
		return errors.WithStack(ErrNotFound)
	} else if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

// query executes a query returning a single row and unmarshals the result into a slice.
func (db *PgDB) queryRows(query string, v interface{}, args ...interface{}) error {
	parser := func(rows *sqlx.Rows, val interface{}) error { return rows.StructScan(val) }
	return db.queryRowsWithParser(query, parser, v, args...)
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

// ExperimentWithTrialSummariesRaw returns a JSON string containing information for an experiment,
// including extra computed information for each trial.
func (db *PgDB) ExperimentWithTrialSummariesRaw(id int) ([]byte, error) {
	return db.rawQuery(`
WITH const AS (
    SELECT config->'searcher'->>'metric' AS metric_name,
           (SELECT
               CASE
                   WHEN coalesce((config->'searcher'
                                        ->>'smaller_is_better')::boolean, true)
                   THEN 1
                   ELSE -1
               END) AS sign
    FROM experiments WHERE id = $1
)
SELECT row_to_json(e)
FROM (
    SELECT e.id, e.state, e.config, e.start_time, e.end_time,
           e.archived, e.git_remote, e.git_commit,
           e.git_committer, e.git_commit_date, e.progress,
           -- Get the trials belonging to this experiment, along with additional "num_steps",
           -- "latest_validation_metrics", and "num_completed_checkpoints" columns.
           (SELECT coalesce(jsonb_agg(t ORDER BY id ASC), '[]'::jsonb)
            FROM (
                SELECT t.end_time, t.experiment_id, t.hparams, t.id, t.seed, t.start_time, t.state,
                       t.warm_start_checkpoint_id,
                       (SELECT count(*)
                        FROM steps s
                        WHERE s.trial_id = t.id
                       ) AS num_steps,
                       (SELECT coalesce(max(s.total_batches), 0)
                        FROM steps s
                        WHERE s.trial_id = t.id AND s.state = 'COMPLETED'
                       ) AS total_batches_processed,
                       (SELECT v.metrics
                        FROM validations v
                        WHERE v.trial_id = t.id AND v.state = 'COMPLETED'
                        ORDER BY v.total_batches DESC
                        LIMIT 1
                       ) AS latest_validation_metrics,
                       (SELECT count(*)
                        FROM checkpoints c
                        WHERE c.trial_id = t.id AND c.state = 'COMPLETED'
                       ) AS num_completed_checkpoints,
                       (SELECT (v.metrics->'validation_metrics'->>const.metric_name)::float8
                        FROM validations v, const
                        WHERE v.trial_id = t.id
                          AND v.state = 'COMPLETED'
                        ORDER BY const.sign * (v.metrics->'validation_metrics'
                                                        ->>const.metric_name)::float8 ASC
                        LIMIT 1
                       ) AS best_validation_metric,
                       (SELECT row_to_json(bc)
                        FROM (
                            SELECT c.id, c.uuid, c.trial_id, c.total_batches, c.state,
                                   c.start_time, c.end_time, c.resources, c.metadata,
                                   (v.metrics->'validation_metrics'
                                             ->>const.metric_name)::float8 AS validation_metric
                            FROM checkpoints c LEFT JOIN validations v
                            ON c.trial_id = v.trial_id AND c.total_batches = v.total_batches, const
                            WHERE c.trial_id = t.id
                              AND c.state = 'COMPLETED'
                              AND v.state = 'COMPLETED'
                            ORDER BY const.sign * (v.metrics->'validation_metrics'
                                                            ->>const.metric_name)::float8 ASC
                            LIMIT 1
                        ) bc
                       ) AS best_available_checkpoint
                   FROM trials t
                WHERE t.experiment_id = e.id
            ) t
           ) AS trials,
			(
				SELECT to_json(u) FROM (SELECT id, username FROM users WHERE id = e.owner_id
				) u
			) as owner,
           -- Compute "validation_history" (end time, trial, and metric value for each validation
           -- that was better than all previous validations).
           (WITH vals AS (
                SELECT v.trial_id, v.end_time, v.state,
                       (v.metrics->'validation_metrics'->>(e.config->'searcher'->>'metric'))::float8
                        AS validation_error
                FROM validations v, trials t
                WHERE v.trial_id = t.id and t.experiment_id = e.id and v.state = 'COMPLETED'
            )
            SELECT coalesce(jsonb_agg(v), '[]'::jsonb)
            FROM (
                SELECT n.trial_id, n.end_time, n.validation_error
                FROM (
                  SELECT v.trial_id, v.end_time, v.validation_error,
                        min(const.sign * v.validation_error)
                          OVER (ORDER BY v.end_time ASC
                                ROWS BETWEEN UNBOUNDED PRECEDING AND 1 PRECEDING)
                          AS prev_min_error
                  FROM vals v,
                       trials t,
                       const
                  WHERE v.trial_id = t.id
                    AND v.state = 'COMPLETED'
                    AND t.experiment_id = e.id
                ) n, const
                WHERE const.sign * n.validation_error < n.prev_min_error
                   OR n.prev_min_error IS NULL
                ORDER BY n.end_time asc
           ) v) as validation_history
    FROM experiments e
    WHERE e.id = $1
) e
`, id)
}

// ExperimentWithSummaryMetricsRaw returns a JSON string containing information
// for one experiment with just summary metrics for all steps instead of all
// metrics.
func (db *PgDB) ExperimentWithSummaryMetricsRaw(id int) ([]byte, error) {
	var metricNames []string
	if err := db.sql.Select(&metricNames, `
SELECT jsonb_object_keys(to_jsonb(r)->'metrics'->'batch_metrics'->0)
FROM (
    SELECT s.metrics
    FROM steps s, trials t, experiments e
    WHERE s.state = 'COMPLETED'
        AND s.trial_id = t.id
        AND t.experiment_id = $1
    LIMIT 1
) r
`, id); err != nil {
		return nil, errors.Wrapf(err, "failed to get metric names for experiment %d", id)
	}

	averageMetrics := make([]string, 0, len(metricNames))
	for _, name := range metricNames {
		// Slow conversion (i.e., try_float8_cast versus ::float8) is ok here
		// because we are only using this in legacy metrics where precomputed
		// averages are not present.
		averageMetrics = append(averageMetrics,
			fmt.Sprintf(`avg(try_float8_cast(value->>'%s')) AS "%s"`, name, name))
	}

	queryTemplate := `
SELECT row_to_json(e)
FROM (
    SELECT e.archived, e.config, e.end_time, e.git_commit, e.git_commit_date, e.git_committer,
           e.git_remote, e.id, e.start_time, e.state, e.progress,
           (SELECT to_json(u) FROM (SELECT id, username FROM users WHERE id = e.owner_id) u)
			as owner,
           (SELECT coalesce(jsonb_agg(t ORDER BY id ASC), '[]'::jsonb)
            FROM (
                SELECT t.end_time, t.experiment_id, t.hparams, t.id, t.seed, t.start_time, t.state,
                       t.warm_start_checkpoint_id,
                (SELECT coalesce(jsonb_agg(s ORDER BY id ASC), '[]'::jsonb)
                 FROM (
                     SELECT s.end_time, s.id, s.start_time, s.state, s.trial_id, s.total_batches,
                     -- Drop batch_metrics field from metrics column because it
                     -- can be very large and compute average on the fly for legacy
                     -- metrics.
                     --
                     -- We cannot use coalesce here because, when metrics
                     -- are pending, its value is 'null'::jsonb which is not equal
                     -- to (sql) NULL.
                     --
                     -- If we migrate metrics data, we can remove the legacy case.
                     (SELECT CASE
                          WHEN s.metrics->'batch_metrics' IS NULL THEN s.metrics
                          WHEN s.metrics->'avg_metrics' IS NULL THEN
                              (s.metrics - 'batch_metrics') ||
                                  jsonb_build_object('avg_metrics',
                                  (SELECT to_jsonb(r1)
                                   FROM (
                                      SELECT %s
                                      FROM jsonb_array_elements(s.metrics->'batch_metrics')
                                   ) r1))
                          ELSE s.metrics - 'batch_metrics'
                      END) AS metrics,
                     (SELECT row_to_json(c)
                      FROM (
                          SELECT c.end_time, c.id, c.metadata, c.resources, c.start_time, c.state,
                                 c.total_batches, c.trial_id, c.uuid
                          FROM checkpoints c
                          WHERE c.trial_id = t.id AND c.total_batches = s.total_batches
                      ) c) AS checkpoint,
                     (SELECT row_to_json(v)
                      FROM (
                          SELECT v.end_time, v.id, v.metrics, v.start_time, v.state,
                                 v.total_batches, v.trial_id
                          FROM validations v
                          WHERE v.trial_id = t.id AND v.total_batches = s.total_batches
                      ) v) AS validation
                     FROM steps s
                     WHERE s.trial_id = t.id
                 ) s
                ) AS steps
                FROM trials t
                WHERE t.experiment_id = e.id
            ) t
           ) AS trials
    FROM experiments e
    WHERE e.id = $1
) e
`
	return db.rawQuery(fmt.Sprintf(queryTemplate, strings.Join(averageMetrics, ",")), id)
}

// CheckExperimentExists checks if the experiment exists.
func (db *PgDB) CheckExperimentExists(id int) (bool, error) {
	var exists bool
	err := db.sql.QueryRow(`
SELECT
EXISTS(
  select id
  FROM experiments
  WHERE id = $1
)`, id).Scan(&exists)
	return exists, err
}

// CheckTrialExists checks if the trial exists.
func (db *PgDB) CheckTrialExists(id int) (bool, error) {
	var exists bool
	err := db.sql.QueryRow(`
SELECT
EXISTS(
  select id
  FROM trials
  WHERE id = $1
)`, id).Scan(&exists)
	return exists, err
}

// TrialExperimentAndRequestID returns the trial's experiment and request ID.
func (db *PgDB) TrialExperimentAndRequestID(id int) (int, model.RequestID, error) {
	var eID int
	var rID model.RequestID
	err := db.sql.QueryRow(`
SELECT e.id, t.request_id
FROM trials t, experiments e
WHERE t.experiment_id = e.id
  AND t.id = $1`, id).Scan(&eID, &rID)
	switch {
	case err == sql.ErrNoRows:
		return eID, rID, errors.WithStack(ErrNotFound)
	case err != nil:
		return eID, rID, errors.Wrap(err, "failed to get trial exp and req id")
	default:
		return eID, rID, nil
	}
}

// ExperimentCheckpointsRaw returns a JSON string describing checkpoints for a given experiment,
// either all of them or the best subset.
func (db *PgDB) ExperimentCheckpointsRaw(id int, numBest *int) ([]byte, error) {
	return db.rawQuery(`
WITH const AS (
    SELECT config->'searcher'->>'metric' AS metric_name,
           (SELECT
               CASE
                   WHEN coalesce((config->'searcher'->>'smaller_is_better')::boolean, true)
                   THEN 1
                   ELSE -1
               END) as sign
    FROM experiments WHERE id = $1
)
SELECT row_to_json(x)
FROM (
    SELECT const.metric_name,
           (SELECT coalesce(jsonb_agg(c), '[]'::jsonb)
            FROM (
                -- We can't use a computed column in a WHERE clause for the same query, which we
                -- would like to do here with the "steps" column, so do this subquery.
                SELECT * FROM (
                    SELECT c.id, c.trial_id, c.total_batches, c.state, c.start_time, c.end_time,
                           c.uuid, c.resources, c.metadata,
                           (SELECT row_to_json(s)
                            FROM (
                                SELECT s.end_time, s.id, s.start_time, s.state, s.trial_id,
                                    s.total_batches,
                                    (SELECT row_to_json(v)
                                    FROM (
                                        SELECT v.end_time, v.id, v.metrics, v.start_time,
                                            v.state, v.total_batches, v.trial_id
                                        FROM validations v
                                        WHERE v.trial_id = s.trial_id
                                              AND v.total_batches = s.total_batches
                                    ) v
                                    ) AS validation
                                FROM steps s
                                WHERE s.total_batches = c.total_batches AND s.trial_id = c.trial_id
                            ) s
                           ) AS step
                    FROM checkpoints c, trials t, const
                    WHERE c.trial_id = t.id AND t.experiment_id = $1
                ) _
                -- If a non-null "numBest" parameter is specified, find the numBest ones with
                -- the best validation metric values. Otherwise, these clauses do nothing.
                WHERE ($2::int IS NULL OR
                       (step->'validation'->'metrics'->
                              'validation_metrics'->>const.metric_name)::float8 IS NOT NULL)
                ORDER BY (
                    CASE
                        WHEN $2 IS NULL
                        THEN NULL
                        ELSE const.sign * (step->'validation'->'metrics'->
                                                 'validation_metrics'->>const.metric_name)::float8
                    END
                ) ASC
                LIMIT $2
            ) c
           ) AS checkpoints
    FROM const
) x
`, id, numBest)
}

// ExperimentConfigRaw returns the full config object for an experiment as a JSON string.
func (db *PgDB) ExperimentConfigRaw(id int) ([]byte, error) {
	return db.rawQuery(`
SELECT config
FROM experiments
WHERE id = $1`, id)
}

// ExperimentConfigByTrialsRaw returns a JSON string with the id, config fields
// of an experiment from a list of trial ids iff all the trial ids provided
// belong to the same experiment. If the trial doesn't exist or the trial ids
// provided belong to different experiments this method returns an error.
func (db *PgDB) ExperimentConfigByTrialsRaw(trialIDs []int) ([]byte, error) {
	var numExp int
	sqlIDList := strings.Join(strings.Fields(fmt.Sprint(trialIDs)), ", ")

	err := db.sql.Get(&numExp, fmt.Sprintf(
		`SELECT COUNT(DISTINCT experiment_id) FROM trials WHERE id IN (%s)`,
		sqlIDList[1:len(sqlIDList)-1],
	))

	if err != nil {
		return nil, err
	}

	if numExp > 1 {
		return nil, errors.Errorf("trial ids %d belong to different experiments", trialIDs)
	}

	if numExp == 0 {
		return nil, errors.Errorf("no experiment found for trial ids %d", trialIDs)
	}

	return db.rawQuery(`
WITH exp AS (SELECT experiment_id FROM trials WHERE id = $1),
conf AS (SELECT id, config FROM experiments WHERE id IN (SELECT * FROM exp))
SELECT coalesce(row_to_json(u), '{}') FROM (SELECT * FROM conf) AS u;
`, trialIDs[0])
}

// ExperimentRaw creates a JSON string containing information for one experiment. The progress is
// not in the database but is expected to be in the JSON result, so it is passed in as an argument.
func (db *PgDB) ExperimentRaw(id int) ([]byte, error) {
	return db.rawQuery(`
SELECT row_to_json(e)
FROM (
    SELECT e.archived, e.config, e.end_time, e.git_commit, e.git_commit_date, e.git_committer,
           e.git_remote, e.id, e.start_time, e.state, e.progress,
           (SELECT to_json(u) FROM (SELECT id, username FROM users WHERE id = e.owner_id) u)
			as owner,
           (SELECT coalesce(jsonb_agg(t ORDER BY id ASC), '[]'::jsonb)
            FROM (
                SELECT t.end_time, t.experiment_id, t.hparams, t.id, t.seed, t.start_time, t.state,
                       t.warm_start_checkpoint_id,
                (SELECT coalesce(jsonb_agg(s ORDER BY id ASC), '[]'::jsonb)
                 FROM (
                     SELECT s.end_time, s.id, s.start_time, s.state, s.trial_id,  s.total_batches,
                     (SELECT row_to_json(c)
                      FROM (
                          SELECT c.end_time, c.id, c.metadata, c.resources, c.start_time, c.state,
                                 c.total_batches, c.trial_id, c.uuid
                          FROM checkpoints c
                          WHERE c.trial_id = t.id AND c.total_batches = s.total_batches
                      ) c) AS checkpoint,
                     (SELECT row_to_json(v)
                      FROM (
                          SELECT v.end_time, v.id, v.metrics, v.start_time, v.state,
                                 v.total_batches, v.trial_id
                          FROM validations v
                          WHERE v.trial_id = t.id AND v.total_batches = s.total_batches
                      ) v) AS validation
                     FROM steps s
                     WHERE s.trial_id = t.id
                 ) s
                ) AS steps
                FROM trials t
                WHERE t.experiment_id = e.id
            ) t
           ) AS trials
    FROM experiments e
    WHERE e.id = $1
) e
`, id)
}

// ExperimentListRaw creates a JSON string containing information for all experiments.
func (db *PgDB) ExperimentListRaw(
	skipArchived bool, username string, limit, offset int,
) ([]byte, error) {
	// Keep track of how many parameters we have added to the query so far.
	varCounter := 1
	usernameQuery := ""
	if username != "" {
		usernameQuery = fmt.Sprintf("AND u.username = $%d", varCounter+1)
		varCounter++
	}

	limitOffsetQuery := ""
	if limit != 0 {
		limitOffsetQuery = fmt.Sprintf(`
LIMIT $%d
OFFSET $%d
		`, varCounter+1, varCounter+2)
	}

	query := fmt.Sprintf(`
SELECT coalesce(jsonb_agg(e ORDER BY e.id DESC), '[]'::jsonb)
FROM (
    SELECT e.archived, e.config, e.end_time, e.git_commit, e.git_commit_date, e.git_committer,
	   e.git_remote, e.id, e.start_time, e.state, e.progress,
      (SELECT to_json(u) FROM (SELECT id, username FROM users WHERE id = e.owner_id) u)
		as owner
    FROM experiments e
	 LEFT JOIN
	 users u
	 ON u.id = e.owner_id
		WHERE (e.archived = false OR $1 = false)
			%s
			%s
) e
`, usernameQuery, limitOffsetQuery)

	// Build up the list of parameters based on the dynamic queries.
	var parameters []interface{}
	parameters = append(parameters, skipArchived)
	if usernameQuery != "" {
		parameters = append(parameters, username)
	}
	if limitOffsetQuery != "" {
		parameters = append(parameters, limit, offset)
	}
	return db.rawQuery(query, parameters...)
}

// ExperimentDescriptorsRaw creates a JSON string containing short descriptors for all experiments.
func (db *PgDB) ExperimentDescriptorsRaw(skipArchived, skipInactive bool) ([]byte, error) {
	return db.rawQuery(`
SELECT coalesce(jsonb_agg(descs ORDER BY id DESC), '[]'::jsonb)
FROM (
    SELECT id,
        config->'description' AS description,
        coalesce(config->'labels', '{}') AS labels
    FROM experiments
    WHERE (archived = false OR $1 = false)
    AND   (state = 'ACTIVE' OR $2 = false)
) descs`, skipArchived, skipInactive)
}

// ExperimentDescriptorsRawForUser returns a JSON string containing short descriptors for each
// experiment owned by a user.
func (db *PgDB) ExperimentDescriptorsRawForUser(skipArchived, skipInactive bool,
	username string) ([]byte, error) {
	return db.rawQuery(`
SELECT coalesce(jsonb_agg(descs ORDER BY id DESC), '[]'::jsonb)
FROM (
    SELECT id,
        config->'description' AS description,
        coalesce(config->'labels', '{}') AS labels
    FROM experiments
    JOIN users ON (experiments.owner_id = users.id)
    WHERE (archived = false OR $1 = false)
    AND   (state = 'ACTIVE' OR $2 = false)
    AND	  (users.username = $3)
) descs`, skipArchived, skipInactive, username)
}

// AddExperiment adds the experiment to the database and sets its ID.
func (db *PgDB) AddExperiment(experiment *model.Experiment) error {
	if experiment.ID != 0 {
		return errors.Errorf("error adding an experiment with non-zero id %v", experiment.ID)
	}
	err := db.namedGet(&experiment.ID, `
INSERT INTO experiments
(state, config, model_definition, start_time, end_time, archived,
 git_remote, git_commit, git_committer, git_commit_date, owner_id, original_config, notes)
VALUES (:state, :config, :model_definition, :start_time, :end_time, :archived,
        :git_remote, :git_commit, :git_committer, :git_commit_date, :owner_id, :original_config,
        :notes)
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

// LegacyExperimentConfigByID parses very old configs, returning a LegacyConfig which
// exposes a select subset of fields in a type-safe way.
func (db *PgDB) LegacyExperimentConfigByID(
	id int,
) (expconf.LegacyConfig, error) {
	var byts []byte
	if err := db.sql.QueryRow(
		"SELECT config FROM experiments WHERE id = $1", id).Scan(&byts); err != nil {
		return expconf.LegacyConfig{}, err
	}

	config, err := expconf.ParseLegacyConfigJSON(byts)
	if err != nil {
		return expconf.LegacyConfig{}, errors.Wrap(err, "parsing legacy conf from database")
	}

	return config, nil
}

// ExperimentWithoutConfigByID looks up an experiment by ID in a database, returning an error if
// none exists. It loads the experiment without its configuration, for callers that do not need
// it, or can't handle backwards incompatible changes.
func (db *PgDB) ExperimentWithoutConfigByID(id int) (*model.Experiment, error) {
	var experiment model.Experiment

	if err := db.query(`
SELECT id, state, model_definition, start_time, end_time, archived,
       git_remote, git_commit, git_committer, git_commit_date, owner_id
FROM experiments
WHERE id = $1`, &experiment, id); err != nil {
		return nil, err
	}

	return &experiment, nil
}

// ExperimentIDByTrialID looks up an experiment ID by a trial ID.
func (db *PgDB) ExperimentIDByTrialID(trialID int) (int, error) {
	var experimentID int
	if err := db.sql.Get(&experimentID, `
SELECT experiment_id FROM trials where id = $1
`, trialID); err != nil {
		return 0, errors.Wrapf(err, "querying for experiment id for trial %v", trialID)
	}
	return experimentID, nil
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
			items, err := rows.SliceScan()
			if err != nil {
				return nil, errors.Wrap(err, "unable to read experiment from db")
			}

			expID, ok := items[0].(int64)
			if !ok {
				return nil, errors.Errorf(
					"Expected an integer experiment ID, but got: %s", reflect.TypeOf(items[0]))
			}

			err = db.TerminateExperimentInRestart(int(expID), model.ErrorState)
			if err != nil {
				log.WithError(err).Error("failed to mark experiment as errored")
			}
			continue
		}
		exps = append(exps, &exp)
	}
	return exps, nil
}

// FailDeletingExperiment finds all experiments that were deleting when the master crashed and moves
// them to DELETE_FAILED.
func (db *PgDB) FailDeletingExperiment() error {
	if _, err := db.sql.Exec(`
UPDATE experiments
SET state = 'DELETE_FAILED'
WHERE state = 'DELETING'
`); err != nil {
		return errors.Wrap(err, "failing deleting experiments")
	}
	return nil
}

// TerminateExperimentInRestart is used during master restart to properly terminate an experiment
// which was either in the process of stopping or which is not restorable for some reason, such as
// an invalid experiment config after a version upgrade.
func (db *PgDB) TerminateExperimentInRestart(id int, state model.State) error {
	if _, ok := model.TerminalStates[state]; !ok {
		return errors.Errorf("state %v is not a terminal state", state)
	}

	now := time.Now().UTC()

	tx, err := db.sql.Begin()
	if err != nil {
		return errors.Wrap(err, "starting transaction")
	}
	defer func() {
		if tx == nil {
			return
		}

		if rErr := tx.Rollback(); rErr != nil {
			log.Errorf("during rollback: %v", rErr)
		}
	}()

	// Terminate trials.
	if _, err = tx.Exec(
		`UPDATE trials SET state=$1, end_time=$2 WHERE experiment_id=$3 and end_time IS NULL`,
		state,
		now,
		id,
	); err != nil {
		return errors.Wrap(err, "terminating trials of a stopping experiment")
	}

	// Terminate experiment.
	if _, err = tx.Exec(
		`UPDATE experiments SET state=$1, end_time=$2, progress=NULL WHERE id=$3`, state, now, id,
	); err != nil {
		return errors.Wrap(err, "terminating a stopping experiment")
	}

	if err = tx.Commit(); err != nil {
		return errors.Wrapf(err, "committing termination of stopping experiment %v", id)
	}

	tx = nil

	return nil
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

// TrySaveExperimentState saves the current experiment state to the database and
// returns if we successfully changed the state or not.
func (db *PgDB) TrySaveExperimentState(experiment *model.Experiment) error {
	var newState, oldState model.State
	if err := db.sql.QueryRowx(`
UPDATE experiments e
SET state=$2
FROM (SELECT state FROM experiments WHERE id = $1 FOR UPDATE) old
WHERE e.id = $1
RETURNING e.state, old.state
`, experiment.ID, experiment.State).Scan(&newState, &oldState); err != nil {
		return errors.Wrap(err, "updating experiment state")
	}
	if newState == oldState {
		return errors.New("could not transition experiment")
	}
	return nil
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
	return db.withTransaction("delete experiment", func(tx *sqlx.Tx) error {
		// This delete cascades to checkpoints and validations.
		// TODO(DET-5210): If/When validations and checkpoints are no longer linked
		// to steps, this delete will not work properly.
		if _, err := tx.Exec(`
DELETE FROM steps
WHERE trial_id IN (SELECT id FROM trials WHERE experiment_id = $1)
`, id); err != nil {
			return errors.Wrapf(err, "error deleting steps for experiment %v", id)
		}

		if err := db.deleteSnapshotsForExperiment(id)(tx); err != nil {
			return errors.Wrapf(err, "error deleting snapshots for experiment %v", id)
		}

		if _, err := tx.Exec(`
DELETE FROM trials
WHERE experiment_id = $1;
`, id); err != nil {
			return errors.Wrapf(err, "error deleting trials for experiment %v", id)
		}

		result, err := tx.Exec(`
DELETE FROM experiments
WHERE id = $1
`, id)
		if err != nil {
			return errors.Wrapf(err, "error deleting experiment %v", id)
		}
		switch num, err := result.RowsAffected(); {
		case err != nil:
			return errors.Wrapf(err, "error in RowsAffected when deleting experiment %v", id)
		case num != 1:
			return errors.Errorf("error deleting non-existing experiment %v", id)
		}
		return nil
	})
}

// ExperimentHasCheckpointsInRegistry checks if the experiment has any checkpoints in the registry.
func (db *PgDB) ExperimentHasCheckpointsInRegistry(id int) (bool, error) {
	var exists bool
	err := db.sql.QueryRow(`
SELECT
EXISTS(
   SELECT 1
   FROM experiments e
   JOIN trials t ON e.id = t.experiment_id
   JOIN checkpoints c ON c.trial_id = t.id
   JOIN model_versions mv ON mv.checkpoint_uuid = c.uuid
   WHERE e.id = $1
)`, id).Scan(&exists)
	return exists, err
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

// ExperimentConfig returns the full config object for an experiment.
func (db *PgDB) ExperimentConfig(id int) (expconf.ExperimentConfig, error) {
	expConfigBytes, err := db.rawQuery(`
SELECT config
FROM experiments
WHERE id = $1`, id)
	if err != nil {
		return expconf.ExperimentConfig{}, err
	}
	expConfig, err := expconf.ParseAnyExperimentConfigYAML(expConfigBytes)
	if err != nil {
		return expconf.ExperimentConfig{}, errors.WithStack(err)
	}
	return schemas.WithDefaults(expConfig).(expconf.ExperimentConfig), nil
}

// ExperimentTotalStepTime returns the total elapsed time for all steps of the experiment
// with the given ID. Any step with a NULL end_time does not contribute. Elapsed time is
// expressed as a floating point number of seconds.
func (db *PgDB) ExperimentTotalStepTime(id int) (float64, error) {
	var seconds float64
	if err := db.sql.Get(&seconds, `
SELECT coalesce(extract(epoch from sum(s.end_time - s.start_time)), 0)
FROM raw_steps s, trials t
WHERE t.experiment_id = $1 AND s.trial_id = t.id
`, id); err != nil {
		return 0, errors.Wrapf(err, "querying for total step time of experiment %v", id)
	}
	return seconds, nil
}

// ExperimentNumTrials returns the total number of trials for the experiment.
func (db *PgDB) ExperimentNumTrials(id int) (int64, error) {
	var numTrials int64
	if err := db.sql.Get(&numTrials, `
SELECT count(*)
FROM trials
WHERE trials.experiment_id = $1
`, id); err != nil {
		return 0, errors.Wrapf(err, "querying for number of trials of experiment %v", id)
	}
	return numTrials, nil
}

// ExperimentTrialIDs returns the trial IDs for the experiment.
func (db *PgDB) ExperimentTrialIDs(expID int) ([]int, error) {
	var trialIDRows []struct {
		ID int
	}
	if err := db.queryRows(`
SELECT id
FROM trials
WHERE trials.experiment_id = $1
`, &trialIDRows, expID); err != nil {
		return nil, errors.Wrapf(err, "querying for trial IDs of experiment %v", expID)
	}
	var trialIDs []int
	for _, r := range trialIDRows {
		trialIDs = append(trialIDs, r.ID)
	}
	return trialIDs, nil
}

// ExperimentNumSteps returns the total number of steps for all trials of the experiment.
func (db *PgDB) ExperimentNumSteps(id int) (int64, error) {
	var numSteps int64
	if err := db.sql.Get(&numSteps, `
SELECT count(*)
FROM raw_steps s, trials t
WHERE t.experiment_id = $1 AND s.trial_id = t.id
`, id); err != nil {
		return 0, errors.Wrapf(err, "querying for number of steps of experiment %v", id)
	}
	return numSteps, nil
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
	experimentBest, trialBest, trialLatest int,
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
            END) AS sign
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
                   ORDER BY total_batches DESC
               ) AS trial_order_rank
        FROM (
            SELECT c.id, c.trial_id, c.total_batches, c.state, c.start_time, c.end_time, c.uuid,
                   c.resources, c.metadata,
                   (SELECT row_to_json(s)
                    FROM (
                        SELECT s.end_time, s.id, s.start_time, s.state, s.trial_id,
                            s.total_batches,
                            (SELECT row_to_json(v)
                            FROM (
                                SELECT v.end_time, v.id, v.metrics, v.start_time,
                                    v.state, v.total_batches, v.trial_id
                                    FROM validations v
                                    WHERE v.trial_id = t.id AND v.total_batches = s.total_batches
                                ) v
                               ) AS validation
                        FROM steps s
                        WHERE s.total_batches = c.total_batches AND s.trial_id = c.trial_id
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
    WHERE (SELECT COUNT(*) FROM trials t WHERE t.warm_start_checkpoint_id = c.id) = 0
          AND c.trial_order_rank > $4
          AND ((c.experiment_rank > $2
                AND c.trial_rank > $3)
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
	err := db.namedGet(&trial.ID, `
INSERT INTO trials
  (request_id, experiment_id, state, start_time, end_time,
   hparams, warm_start_checkpoint_id, seed)
VALUES (:request_id, :experiment_id, :state, :start_time,
        :end_time, :hparams, :warm_start_checkpoint_id, :seed)
RETURNING id`, trial)
	// Assume the foreign key constraint is handled by the database.
	return errors.Wrapf(err, "error inserting trial %v", *trial)
}

// TrialByID looks up a trial by ID, returning an error if none exists.
func (db *PgDB) TrialByID(id int) (*model.Trial, error) {
	var trial model.Trial
	err := db.query(`
SELECT id, request_id, experiment_id, state, start_time, end_time,
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
SELECT id, request_id, experiment_id, state, start_time,
  end_time, hparams, warm_start_checkpoint_id, seed
FROM trials
WHERE experiment_id = $1 AND request_id = $2`, &trial, experimentID, requestID)
	return &trial, errors.Wrapf(err, "error querying for trial %v", requestID)
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
// that happened after the batch provided.
func (db *PgDB) RollBackTrial(id, totalBatches int) error {
	// This delete cascades to checkpoints and validations.
	_, err := db.sql.Exec(`
DELETE FROM raw_steps
WHERE trial_id = $1 AND total_batches > $2
`, id, totalBatches)
	if err != nil {
		return errors.Wrapf(err, "error rolling back trial %v to batch %v", id, totalBatches)
	}

	// This explicitly deletes any unfinished validations for the current step. These can occur
	// any time we checkpoint before we validate.
	_, err = db.sql.Exec(`
DELETE FROM raw_validations
WHERE trial_id = $1 AND total_batches = $2 AND state != 'COMPLETED'
`, id, totalBatches)
	if err != nil {
		return errors.Wrapf(err, "error rolling back vals for trial %v on batch %v", id, totalBatches)
	}

	err = db.SetTrialBestValidation(id)
	if err != nil {
		return errors.Wrapf(err, "error rolling back best val for trial %v on batch %v", id, totalBatches)
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
                SELECT s.end_time, s.id, s.state, s.start_time, s.total_batches,
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
                            SELECT v.end_time, v.id, v.metrics, v.state, v.start_time
                            FROM validations v
                            WHERE v.trial_id = t.id AND v.total_batches = s.total_batches
                                  AND v.metrics IS NOT NULL
                        ) r4
                       ) AS validation,
                       (SELECT row_to_json(r5)
                        FROM (
                            SELECT c.id, c.trial_id, c.total_batches, c.state, c.start_time,
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
INSERT INTO raw_steps
	(trial_id, id, total_batches, state, start_time, end_time)
VALUES (
	:trial_id, :id, :total_batches, :state, :start_time, :end_time
)`, step)
	if err != nil {
		return errors.Wrapf(err, "error inserting step %v", *step)
	}
	return nil
}

// AddNoOpStep adds a no-op completed step to the database. This is used for trials with initial
// validations (used for testing models pre-fine-tuning).
func (db *PgDB) AddNoOpStep(step *model.Step) error {
	if step.State != model.CompletedState {
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
INSERT INTO raw_steps
	(trial_id, id, total_batches, state, start_time, end_time)
VALUES (
	:trial_id, :id, :total_batches, :state, :start_time, :end_time
)`,
		step)
	if err != nil {
		return errors.Wrapf(err, "error inserting step %v", *step)
	}
	return nil
}

// AddTrialRun saves a new run for the trial.
func (db *PgDB) AddTrialRun(trialID, runID int) error {
	_, err := db.sql.Exec(`
INSERT INTO runs (run_type, run_type_fk, id)
VALUES ('TRIAL', $1, $2)
ON CONFLICT (run_type, run_type_fk, id)
DO UPDATE SET start_time = now()`, trialID, runID)
	return err
}

// CompleteTrialRun the given run.
func (db *PgDB) CompleteTrialRun(trialID, runID int) error {
	_, err := db.sql.Exec(`
UPDATE runs
SET end_time = now()
WHERE run_type = 'TRIAL'
  AND run_type_fk = $1 AND id = $2`, trialID, runID)
	return err
}

// EndTrialRuns sets the end time on all open runs to now.
func (db *PgDB) EndTrialRuns(trialID int) error {
	_, err := db.sql.Exec(`
UPDATE runs
SET end_time = now()
WHERE run_type = 'TRIAL'
  AND run_type_fk = $1
  AND end_time IS NULL`, trialID)
	return err
}

// StepByTotalBatches looks up a step by (TrialID, TotalBatches) pair,
// returning an error if none exists.
func (db *PgDB) StepByTotalBatches(trialID, totalBatches int) (*model.Step, error) {
	var step model.Step
	if err := db.query(`
SELECT
	trial_id, id, total_batches, state, start_time, end_time, metrics
FROM steps
WHERE trial_id = $1 AND total_batches = $2`, &step, trialID, totalBatches); err != nil {
		return nil, errors.Wrapf(err, "error querying for step %v, %v", trialID, totalBatches)
	}
	return &step, nil
}

// UpdateStep updates an existing step. Fields that are nil or zero are not
// updated.  end_time is set if the step moves to a terminal state.
func (db *PgDB) UpdateStep(
	trialID, totalBatches int, newState model.State, metrics model.JSONObj) error {
	if len(newState) == 0 && len(metrics) == 0 {
		return nil
	}
	step, err := db.StepByTotalBatches(trialID, totalBatches)
	if err != nil {
		return errors.Wrapf(err, "error finding step (%v, %v) to update", trialID, totalBatches)
	}
	return db.updateStep(step, newState, metrics)
}

// AddTrainingMetrics adds a completed step to the database with the given training metrics.
// If these training metrics occur before any others, a rollback is assumed and later
// training and validation metrics are cleaned up.
func (db *PgDB) AddTrainingMetrics(ctx context.Context, m *trialv1.TrainingMetrics) error {
	return db.withTransaction("add training metrics", func(tx *sqlx.Tx) error {
		if err := checkTrialRunID(ctx, tx, m.TrialId, m.TrialRunId); err != nil {
			return err
		}

		if _, err := tx.ExecContext(ctx, `
UPDATE raw_steps SET archived = true
WHERE trial_id = $1
  AND trial_run_id < $2
  AND total_batches >= $3;
`, m.TrialId, m.TrialRunId, m.TotalBatches); err != nil {
			return errors.Wrap(err, "archiving training metrics")
		}

		if _, err := tx.ExecContext(ctx, `
UPDATE raw_validations SET archived = true
WHERE trial_id = $1
  AND trial_run_id < $2
  AND total_batches > $3;
`, m.TrialId, m.TrialRunId, m.TotalBatches); err != nil {
			return errors.Wrap(err, "archiving validations")
		}

		if _, err := tx.ExecContext(ctx, `
UPDATE raw_checkpoints SET archived = true
WHERE trial_id = $1
  AND trial_run_id < $2
  AND total_batches > $3;
`, m.TrialId, m.TrialRunId, m.TotalBatches); err != nil {
			return errors.Wrap(err, "archiving checkpoints")
		}

		// TODO(DET-5210): This can go away when step ID does.
		var id int
		if err := tx.QueryRowxContext(ctx, `
SELECT coalesce(max(id), 0) + 1
FROM raw_steps
WHERE trial_id = $1`, m.TrialId).Scan(&id); err != nil {
			return errors.Wrap(err, "querying next step id")
		}

		startTime, err := derivePriorWorkloadEndTime(ctx, tx, m.TrialId)
		if err != nil {
			return err
		}

		if _, err := tx.NamedExecContext(ctx, `
INSERT INTO raw_steps
	(trial_id, id, trial_run_id, state, start_time,
	 end_time, metrics, total_batches, total_records, total_epochs)
VALUES
	(:trial_id, :id, :trial_run_id, :state, :start_time,
	 now(), :metrics, :total_batches, :total_records, :total_epochs)
`, model.Step{
			TrialID:    int(m.TrialId),
			ID:         id,
			TrialRunID: int(m.TrialRunId),
			State:      model.CompletedState,
			StartTime:  startTime,
			Metrics: map[string]interface{}{
				"avg_metrics":   m.Metrics,
				"batch_metrics": m.BatchMetrics,
			},
			TotalBatches: int(m.TotalBatches),
			TotalRecords: int(m.TotalRecords),
			TotalEpochs:  m.TotalEpochs,
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
	ctx context.Context, m *trialv1.ValidationMetrics,
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
`, m.TrialId, m.TotalBatches); err != nil {
			return errors.Wrap(err, "archiving validations")
		}

		startTime, err := derivePriorWorkloadEndTime(ctx, tx, m.TrialId)
		if err != nil {
			return err
		}

		if _, err := tx.NamedExecContext(ctx, `
INSERT INTO raw_validations
	(trial_id, trial_run_id, state, start_time, end_time,
	 metrics, total_batches, total_records, total_epochs)
VALUES
	(:trial_id, :trial_run_id, :state, :start_time, now(),
	 :metrics, :total_batches, :total_records, :total_epochs)
`, model.Validation{
			TrialID:    int(m.TrialId),
			TrialRunID: int(m.TrialRunId),
			State:      model.CompletedState,
			StartTime:  startTime,
			Metrics: map[string]interface{}{
				"validation_metrics": m.Metrics,
			},
			TotalBatches: int(m.TotalBatches),
			TotalRecords: int(m.TotalRecords),
			TotalEpochs:  m.TotalEpochs,
		}); err != nil {
			return errors.Wrap(err, "inserting validation metrics")
		}
		return nil
	})
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
`, m.TrialId, m.TrialRunId, m.TotalBatches); err != nil {
			return errors.Wrap(err, "archiving checkpoints")
		}

		startTime, err := derivePriorWorkloadEndTime(ctx, tx, m.TrialId)
		if err != nil {
			return err
		}

		if _, err := tx.NamedExecContext(ctx, `
INSERT INTO raw_checkpoints
	(trial_id, trial_run_id, state, start_time, end_time, total_batches,
	 total_records, total_epochs, uuid, resources, framework, format, determined_version)
VALUES
	(:trial_id, :trial_run_id, :state, :start_time, now(), :total_batches,
	 :total_records, :total_epochs, :uuid, :resources, :framework, :format, :determined_version)
`, model.Checkpoint{
			TrialID:           int(m.TrialId),
			TrialRunID:        int(m.TrialRunId),
			State:             model.CompletedState,
			StartTime:         startTime,
			TotalBatches:      int(m.TotalBatches),
			TotalRecords:      int(m.TotalRecords),
			TotalEpochs:       m.TotalEpochs,
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

// derivePriorWorkloadEndTime approximates the start time of currently reported metrics since
// resource allocation uses these times.
func derivePriorWorkloadEndTime(
	ctx context.Context, tx *sqlx.Tx, trialID int32,
) (time.Time, error) {
	var endTime time.Time
	if err := tx.QueryRowxContext(ctx, `
SELECT coalesce(greatest(
	(SELECT max(end_time) FROM raw_steps WHERE trial_id = $1),
	(SELECT max(end_time) FROM raw_validations WHERE trial_id = $1),
	(SELECT max(end_time) FROM raw_checkpoints WHERE trial_id = $1),
	(
	    SELECT coalesce(r.start_time, t.start_time)
		FROM trials t
		LEFT JOIN runs r ON t.id = r.run_type_fk
		WHERE t.id = $1
	      AND r.run_type = 'TRIAL'
	    ORDER BY r.id DESC
	    LIMIT 1
	)), now())
`, trialID).Scan(&endTime); err != nil {
		return time.Time{}, errors.Wrap(err, "deriving start time")
	}
	return endTime, nil
}

func checkTrialRunID(ctx context.Context, tx *sqlx.Tx, trialID, runID int32) error {
	var cRunID int
	switch err := tx.QueryRowxContext(ctx, `
SELECT coalesce(max(id), 0)
FROM runs
WHERE run_type = 'TRIAL'
  AND run_type_fk = $1
`, trialID).Scan(&cRunID); {
	case err != nil:
		return errors.Wrap(err, "querying current run")
	case int(runID) != cRunID:
		return api.AsErrBadRequest("invalid run id, %d != %d", runID, cRunID)
	default:
		return nil
	}
}

func (db *PgDB) updateStep(step *model.Step, newState model.State, metrics model.JSONObj) error {
	toUpdate := []string{}
	if len(newState) != 0 {
		if !model.StepTransitions[step.State][newState] {
			return errors.Errorf("illegal transition %v -> %v for step (%v, %v)",
				step.State, newState, step.TrialID, step.TotalBatches)
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
			return errors.Errorf("step (%v, %v) already has metrics", step.TrialID, step.ID)
		}
		step.Metrics = metrics
		toUpdate = append(toUpdate, "metrics")
	}
	err := db.namedExecOne(fmt.Sprintf(`
UPDATE raw_steps
%v
WHERE trial_id = :trial_id
AND id = :id`, setClause(toUpdate)), step)
	if err != nil {
		return errors.Wrapf(err, "error updating (%v) in step (%v, %v)",
			strings.Join(toUpdate, ", "), step.TrialID, step.TotalBatches)
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
	var count int
	err = db.namedGet(&count, `
SELECT COUNT(*)
FROM validations
WHERE trial_id = :trial_id
AND total_batches = :total_batches`, validation)
	if err != nil {
		return errors.Wrapf(err, "error checking at-most-one validation %v", *validation)
	}
	if count > 0 {
		return errors.Errorf("duplicate validation for trial %v total batch %v",
			validation.TrialID, validation.TotalBatches)
	}
	err = db.namedGet(&validation.ID, `
INSERT INTO validations
(trial_id, total_batches, state, start_time, end_time)
VALUES (:trial_id, :total_batches, :state, :start_time, :end_time)
RETURNING id`, validation)
	if err != nil {
		return errors.Wrapf(err, "error inserting validation %v", *validation)
	}
	return nil
}

// ValidationByTotalBatches looks up a validation by trial and step ID,
// returning nil if none exists.
func (db *PgDB) ValidationByTotalBatches(trialID, totalBatches int) (*model.Validation, error) {
	var validation model.Validation
	if err := db.query(`
SELECT id, trial_id, total_batches, state, start_time, end_time, metrics
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

// UpdateValidation updates an existing validation. Fields that are nil or zero
// are not updated. end_time is set if the validation moves to a terminal
// state.
func (db *PgDB) UpdateValidation(
	trialID, totalBatches int, newState model.State, metrics model.JSONObj,
) error {
	if len(newState) == 0 && len(metrics) == 0 {
		return nil
	}
	validation, err := db.ValidationByTotalBatches(trialID, totalBatches)
	if err != nil {
		return errors.Wrapf(err, "error querying for validation (%v, %v) to update",
			trialID, totalBatches)
	}
	if validation == nil {
		return errors.Wrapf(err, "can't update missing validation (%v, %v)",
			trialID, totalBatches)
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
				trialID, totalBatches)
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
			strings.Join(toUpdate, ", "), trialID, totalBatches)
	}

	if err := db.SetTrialBestValidation(trialID); err != nil {
		return errors.Wrapf(err, "error setting best validation for (%v, %v)", trialID, totalBatches)
	}

	return nil
}

// AddCheckpoint adds the checkpoint to the database and sets its ID.
func (db *PgDB) AddCheckpoint(checkpoint *model.Checkpoint) error {
	if !checkpoint.IsNew() {
		return errors.Errorf("unexpected state for new checkpoint: %v", checkpoint)
	}
	if err := db.namedGet(&checkpoint.ID, `
INSERT INTO checkpoints
(trial_id, total_batches, state, start_time, metadata, determined_version)
VALUES (:trial_id, :total_batches, :state, :start_time, :metadata, :determined_version)
RETURNING id`, checkpoint); err != nil {
		return errors.Wrapf(err, "error inserting checkpoint %v", *checkpoint)
	}
	return nil
}

// CheckpointByTotalBatches looks up a checkpoint by trial and total batch,
// returning nil if none exists.
func (db *PgDB) CheckpointByTotalBatches(trialID, totalBatches int) (*model.Checkpoint, error) {
	var checkpoint model.Checkpoint
	if err := db.query(`
SELECT id, trial_id, total_batches, state, start_time, end_time, uuid, resources, metadata
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
SELECT id, trial_id, total_batches, state, start_time, end_time, uuid, resources, metadata
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
SELECT id, trial_id, total_batches, state, start_time, end_time, uuid, resources, metadata
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

// UpdateCheckpoint updates an existing checkpoint. Fields that are nil or zero
// are not updated. end_time is set if the checkpoint moves to a terminal
// state.
func (db *PgDB) UpdateCheckpoint(
	trialID, totalBatches int,
	newCheckpoint model.Checkpoint,
) error {
	if len(newCheckpoint.State) == 0 && len(*newCheckpoint.UUID) == 0 &&
		len(newCheckpoint.Resources) == 0 && len(newCheckpoint.Metadata) == 0 {
		return nil
	}

	checkpoint, err := db.CheckpointByTotalBatches(trialID, totalBatches)
	if err != nil {
		return errors.Wrapf(err, "error querying for checkpoint (%v, %v) to update",
			trialID, totalBatches)
	}
	if checkpoint == nil {
		return errors.Wrapf(err, "can't update missing checkpoint (%v, %v)",
			trialID, totalBatches)
	}

	toUpdate := []string{}
	if len(newCheckpoint.State) != 0 {
		if !model.CheckpointTransitions[checkpoint.State][newCheckpoint.State] {
			return errors.Errorf("illegal transition %v -> %v for checkpoint %v",
				checkpoint.State, newCheckpoint.State, checkpoint.ID)
		}
		checkpoint.State = newCheckpoint.State
		toUpdate = append(toUpdate, "state")
		if model.TerminalStates[newCheckpoint.State] {
			now := time.Now().UTC()
			checkpoint.EndTime = &now
			toUpdate = append(toUpdate, "end_time")
		}
	}
	if newCheckpoint.UUID != nil && len(*newCheckpoint.UUID) != 0 {
		if checkpoint.UUID != nil && len(*checkpoint.UUID) != 0 {
			return errors.Errorf("checkpoint (%v, %v) already has UUID",
				trialID, totalBatches)
		}
		checkpoint.UUID = newCheckpoint.UUID
		toUpdate = append(toUpdate, "uuid")
	}
	if len(newCheckpoint.Resources) != 0 {
		if len(checkpoint.Resources) != 0 {
			return errors.Errorf("checkpoint (%v, %v) already has resources",
				trialID, totalBatches)
		}
		checkpoint.Resources = newCheckpoint.Resources
		toUpdate = append(toUpdate, "resources")
	}
	if len(newCheckpoint.Metadata) != 0 {
		if len(checkpoint.Metadata) == 0 {
			checkpoint.Metadata = model.JSONObj{}
		}

		for k, v := range newCheckpoint.Metadata {
			checkpoint.Metadata[k] = v
		}

		toUpdate = append(toUpdate, "metadata")
	}

	if len(newCheckpoint.Framework) != 0 {
		if len(checkpoint.Framework) != 0 {
			return errors.Errorf("checkpoint (%v, %v) already has a framework", trialID, totalBatches)
		}

		checkpoint.Framework = newCheckpoint.Framework
		toUpdate = append(toUpdate, "framework")
	}

	if len(newCheckpoint.Format) != 0 {
		if len(checkpoint.Format) != 0 {
			return errors.Errorf("checkpoint (%v, %v) already has a format", trialID, totalBatches)
		}

		checkpoint.Format = newCheckpoint.Format
		toUpdate = append(toUpdate, "format")
	}

	err = db.namedExecOne(fmt.Sprintf(`
UPDATE checkpoints
%v
WHERE id = :id`, setClause(toUpdate)), checkpoint)
	if err != nil {
		return errors.Wrapf(err, "error updating (%v) in checkpoint (%v, %v)",
			strings.Join(toUpdate, ", "), trialID, totalBatches)
	}
	return nil
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

func (db *PgDB) queryRowsWithParser(
	query string, p func(*sqlx.Rows, interface{}) error, v interface{}, args ...interface{},
) error {
	rows, err := db.sql.Queryx(query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	vType := reflect.TypeOf(v).Elem()
	switch kind := vType.Kind(); kind {
	case reflect.Slice:
		vValue := reflect.ValueOf(v).Elem()
		vValue.Set(reflect.MakeSlice(vValue.Type(), 0, 0))
		for rows.Next() {
			switch k := vValue.Type().Elem().Kind(); k {
			case reflect.Ptr:
				sValue := reflect.New(vValue.Type().Elem().Elem())
				if err = p(rows, sValue.Interface()); err != nil {
					return err
				}
				vValue = reflect.Append(vValue, sValue)
			case reflect.Struct:
				sValue := reflect.New(vValue.Type().Elem())
				if err = p(rows, sValue.Interface()); err != nil {
					return err
				}
				vValue = reflect.Append(vValue, sValue.Elem())
			default:
				return errors.Errorf("unexpected type: %s", k)
			}
		}
		reflect.ValueOf(v).Elem().Set(vValue)
		return nil
	case reflect.Struct:
		if rows.Next() {
			return p(rows, v)
		}
		return ErrNotFound
	default:
		panic(fmt.Sprintf("unsupported query type: %s", kind))
	}
}

// Query returns the result of the query. Any placeholder parameters are replaced
// with supplied params.
func (db *PgDB) Query(queryName string, v interface{}, params ...interface{}) error {
	parser := func(rows *sqlx.Rows, val interface{}) error { return rows.StructScan(val) }
	return db.queryRowsWithParser(db.queries.getOrLoad(queryName), parser, v, params...)
}

// QueryF returns the result of the formatted query. Any placeholder parameters are replaced
// with supplied params.
func (db *PgDB) QueryF(
	queryName string, args []interface{}, v interface{}, params ...interface{}) error {
	parser := func(rows *sqlx.Rows, val interface{}) error { return rows.StructScan(val) }
	query := db.queries.getOrLoad(queryName)
	if len(args) > 0 {
		query = fmt.Sprintf(query, args...)
	}
	return db.queryRowsWithParser(query, parser, v, params...)
}

// RawQuery returns the result of the query as a raw byte string. Any placeholder parameters are
// replaced with supplied params.
func (db *PgDB) RawQuery(queryName string, params ...interface{}) ([]byte, error) {
	return db.rawQuery(db.queries.getOrLoad(queryName), params...)
}

// withTransaction executes a function with a transaction.
func (db *PgDB) withTransaction(name string, exec func(tx *sqlx.Tx) error) error {
	tx, err := db.sql.Beginx()
	if err != nil {
		return errors.Wrapf(err, "failed to start transaction (%s)", name)
	}
	defer func() {
		if tx == nil {
			return
		}

		if rErr := tx.Rollback(); rErr != nil {
			log.Errorf("failed to rollback transaction (%s): %v", name, rErr)
		}
	}()

	if err = exec(tx); err != nil {
		return errors.Wrapf(err, "failed to exec transaction (%s)", name)
	}

	if err = tx.Commit(); err != nil {
		return errors.Wrapf(err, "failed to commit transaction: (%s)", name)
	}

	tx = nil
	return nil
}

// SetTrialBestValidation sets `public.trials.best_validation_id` to the `id` of the row in
// `public.validations` corresponding to the trial's best validation.
func (db *PgDB) SetTrialBestValidation(id int) error {
	_, err := db.sql.Exec(`
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
			`SELECT date_trunc('day', min(start_time)) FROM raw_steps`,
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
