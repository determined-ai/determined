DROP VIEW trials;
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
  r.log_retention_days AS log_retention_days,
  
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

  r.metadata as metadata,

  -- eventually delete runner state.
  r.runner_state AS runner_state
    
FROM trials_v2 t
JOIN runs r ON t.run_id = r.id;
