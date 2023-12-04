ALTER TABLE trials RENAME TO runs;
ALTER TABLE runs RENAME COLUMN run_id TO restart_id;
ALTER TABLE runs RENAME COLUMN external_trial_id TO external_run_id;

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
  external_run_id AS external_trial_id,
  restart_id AS run_id,
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
FROM runs;
