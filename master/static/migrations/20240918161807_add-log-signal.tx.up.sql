ALTER table public.runs
    ADD COLUMN log_signal text;

DROP VIEW IF EXISTS trials;

CREATE VIEW trials AS
WITH task_log_retention AS (
  SELECT  
    r.run_id, 
    -- This is written with the assumption that every task related 
    -- to a trial will have the same log_retention_days. MIN() is 
    -- just used to aggregate the number of days for each trial.
    MIN(t.log_retention_days) as log_retention_days
  FROM tasks t
  JOIN run_id_task_id as r ON t.task_id = r.task_id
  GROUP BY r.run_id
)
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
  l.log_retention_days AS log_retention_days,

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
  t.seed AS seed,

  r.experiment_id AS experiment_id,
  r.warm_start_checkpoint_id AS warm_start_checkpoint_id,
  r.log_signal AS log_signal,

  -- eventually delete runner state.
  r.runner_state AS runner_state
FROM trials_v2 t
JOIN runs r ON t.run_id = r.id
JOIN task_log_retention l ON l.run_id = t.run_id;
