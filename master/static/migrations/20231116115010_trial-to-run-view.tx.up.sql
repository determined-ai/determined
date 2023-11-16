ALTER TABLE trials RENAME TO runs;

-- TODO unhack this. Or maybe don't.
CREATE TABLE dummy (
    xyz INT
);
INSERT INTO dummy (xyz) VALUES (1);

CREATE OR REPLACE VIEW trials AS -- TODO just replace view
SELECT
  id AS id,
  experiment_id AS experiment_id,
  state AS state,
  start_time AS start_time,
  end_time AS end_time,
  hparams AS hparams,
  warm_start_checkpoint_id AS warm_start_checkpoint_id,
  seed AS seed,
  request_id AS request_id,
  best_validation_id AS best_validation_id,
  runner_state AS runner_state,
  run_id AS run_id,
  restarts AS restarts,
  tags AS tags,
  checkpoint_size AS checkpoint_size,
  checkpoint_count AS checkpoint_count,
  searcher_metric_value AS searcher_metric_value,
  total_batches AS total_batches,
  searcher_metric_value_signed AS searcher_metric_value_signed,
  latest_validation_id AS latest_validation_id,
  summary_metrics AS summary_metrics,
  summary_metrics_timestamp AS summary_metrics_timestamp,
  last_activity AS last_activity,
  external_trial_id AS external_trial_id
FROM runs, dummy; -- FROM dummy is a hack to make this view not insertable.
