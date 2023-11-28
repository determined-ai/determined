-- Is this run_id a bad idea as a primary key?
-- Would it make sense for "possible performance"
-- reasons to use a an id that won't have gaps in the sequence.
-- My only issue is then a field called id could be mistaked for trial id.
CREATE TABLE trials_v2 (
    run_id integer PRIMARY KEY REFERENCES runs(id) ON DELETE CASCADE,
    request_id text,
    runner_state text NOT NULL DEFAULT 'UNSPECIFIED',
    seed integer NOT NULL DEFAULT 0
);

-- TODO how expensive will this be? (Likely not that expesive???)
INSERT INTO trials_v2 (run_id, request_id, runner_state, seed)
SELECT id, request_id, runner_state, seed
FROM runs;

DROP VIEW trials;
DROP TABLE dummy;

ALTER TABLE public.runs
  DROP COLUMN request_id,
  DROP COLUMN runner_state,
  DROP COLUMN seed;

CREATE VIEW trials AS
SELECT
  t.run_id AS id,

  -- metrics
  r.summary_metrics AS summary_metrics,
  r.summary_metrics_timestamp AS summary_metrics_timestamp,
  r.latest_validation_id AS latest_validation_id,
  r.total_batches AS total_batches,

  -- metadata fields
  r.state AS state,
  r.tags AS tags,
  r.external_run_id AS external_trial_id,
  r.restart_id AS run_id,
  r.last_activity AS last_activity,
  r.start_time AS start_time,
  r.end_time AS end_time,
  r.restarts AS restarts,
  -- project_id / owner_id will be propagated from experiments.

  -- run_hp_search_stuff
  r.hparams AS hparams,
  r.searcher_metric_value AS searcher_metric_value,
  r.searcher_metric_value_signed AS searcher_metric_value_signed,
  r.best_validation_id AS best_validation_id,

  -- run_checkpoint_stats
  r.checkpoint_size AS checkpoint_size,
  r.checkpoint_count AS checkpoint_count,

  -- trial_v2 table.
  t.request_id AS request_id,
  t.runner_state AS runner_state,
  t.seed AS seed,

  -- experiment_id will eventually be in the runs to run collection MTM.
  r.experiment_id AS experiment_id,

  -- warm_start_checkpoint_id will eventually be in the runs to checkpoint MTM.
  r.warm_start_checkpoint_id AS warm_start_checkpoint_id
FROM trials_v2 t
JOIN runs r ON t.run_id = r.id;
