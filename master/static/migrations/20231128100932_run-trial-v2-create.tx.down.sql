DROP VIEW trials;

ALTER TABLE public.runs
  ADD COLUMN request_id text,
  ADD COLUMN runner_state text,
  ADD COLUMN seed integer;

UPDATE runs SET
  request_id = t.request_id,
  runner_state = t.runner_state,
  seed = t.seed
FROM trials_v2 t
WHERE runs.id = t.run_id;

ALTER TABLE runs
  ALTER COLUMN runner_state SET NOT NULL,
  ALTER COLUMN runner_state SET DEFAULT 'UNSPECIFIED',
  ALTER COLUMN seed SET NOT NULL,
  ALTER COLUMN seed SET DEFAULT 0;

DROP TABLE trials_v2;

CREATE TABLE dummy (
  xyz INT
);
INSERT INTO dummy (xyz) VALUES (1);

-- TODO lift this from previous migration
CREATE VIEW trials AS
SELECT
  id AS id,

  -- metrics
  summary_metrics AS summary_metrics,
  summary_metrics_timestamp AS summary_metrics_timestamp,
  latest_validation_id AS latest_validation_id,
  total_batches AS total_batches,

  -- metadata fields
  state AS state,
  tags AS tags,
  external_run_id AS external_trial_id, -- TODO rename
  restart_id AS run_id, -- TODO rename
  last_activity AS last_activity,
  start_time AS start_time,
  end_time AS end_time,
  restarts AS restarts,
  -- project_id / owner_id will be propagated from experiments.

  -- run_hp_search_stuff
  hparams AS hparams,
  searcher_metric_value AS searcher_metric_value,
  searcher_metric_value_signed AS searcher_metric_value_signed,
  best_validation_id AS best_validation_id,

  -- run_checkpoint_stats
  checkpoint_size AS checkpoint_size,
  checkpoint_count AS checkpoint_count,

  -- trial_v2 table.
  request_id AS request_id,
  runner_state AS runner_state,
  seed AS seed,

  -- experiment_id will eventually be in the runs to run collection MTM.
  experiment_id AS experiment_id,

  -- warm_start_checkpoint_id will eventually be in the runs to checkpoint MTM.
  warm_start_checkpoint_id AS warm_start_checkpoint_id
FROM runs, dummy; -- FROM dummy is a hack to make this view not insertable.
