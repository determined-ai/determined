CREATE VIEW trials AS
 SELECT t.run_id AS id,
    r.summary_metrics,
    r.summary_metrics_timestamp,
    r.latest_validation_id,
    r.total_batches,
    r.state,
    r.tags,
    r.external_run_id AS external_trial_id,
    r.restart_id AS run_id,
    r.last_activity,
    r.start_time,
    r.end_time,
    r.restarts,
    r.log_retention_days,
    r.hparams,
    r.searcher_metric_value,
    r.searcher_metric_value_signed,
    r.best_validation_id,
    r.checkpoint_size,
    r.checkpoint_count,
    t.request_id,
    t.seed,
    r.experiment_id,
    r.warm_start_checkpoint_id,
    r.runner_state,
    r.log_policy_matched,
    rm.metadata AS metadata
   FROM trials_v2 t
     JOIN runs r ON t.run_id = r.id
     LEFT JOIN runs_metadata rm ON r.id = rm.run_id;

CREATE VIEW steps AS
 SELECT raw_steps.trial_id,
    raw_steps.end_time,
    raw_steps.metrics,
    raw_steps.total_batches,
    raw_steps.trial_run_id,
    raw_steps.archived,
    raw_steps.id
   FROM raw_steps
  WHERE NOT raw_steps.archived;

CREATE VIEW validations AS
  SELECT raw_validations.id,
    raw_validations.trial_id,
    raw_validations.end_time,
    raw_validations.metrics,
    raw_validations.total_batches,
    raw_validations.trial_run_id,
    raw_validations.archived
   FROM raw_validations
  WHERE NOT raw_validations.archived;


CREATE FUNCTION get_raw_metric(v raw_validations, e experiments) RETURNS double precision
    LANGUAGE sql STABLE
    AS $$
    SELECT (v.metrics->'validation_metrics'->>(e.config->'searcher'->>'metric'))::float8
$$;

CREATE FUNCTION get_signed_metric(v raw_validations, e experiments) RETURNS double precision
    LANGUAGE sql STABLE
    AS $$
    SELECT get_raw_metric(v, e) * (
        SELECT
        CASE
            WHEN coalesce((e.config->'searcher'->>'smaller_is_better')::boolean, true)
            THEN 1
            ELSE -1
        END)
$$;

CREATE VIEW validation_metrics AS
 SELECT v.id,
    get_raw_metric(v.*, e.*) AS raw,
    get_signed_metric(v.*, e.*) AS signed
   FROM experiments e,
    runs t,
    raw_validations v
  WHERE e.id = t.experiment_id AND t.id = v.trial_id;

CREATE FUNCTION autoupdate_exp_best_trial_metrics() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    WITH bt AS (
        SELECT id, best_validation_id
        FROM trials
        WHERE experiment_id = NEW.experiment_id
        ORDER BY searcher_metric_value_signed LIMIT 1)
    UPDATE experiments SET best_trial_id = bt.id FROM bt
    WHERE experiments.id = NEW.experiment_id;
    RETURN NEW;
END;
$$;
CREATE TRIGGER autoupdate_exp_best_trial_metrics AFTER UPDATE OF best_validation_id ON runs FOR EACH ROW EXECUTE PROCEDURE autoupdate_exp_best_trial_metrics();


CREATE FUNCTION autoupdate_exp_best_trial_metrics_on_delete() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    WITH bt AS (
        SELECT id, best_validation_id
        FROM trials
        WHERE experiment_id = OLD.experiment_id
        ORDER BY searcher_metric_value_signed LIMIT 1)
    UPDATE experiments SET best_trial_id = bt.id FROM bt
    WHERE experiments.id = OLD.experiment_id;
    RETURN NEW;
END;
$$;
CREATE TRIGGER autoupdate_exp_best_trial_metrics_on_run_delete AFTER DELETE ON runs FOR EACH ROW EXECUTE PROCEDURE autoupdate_exp_best_trial_metrics_on_delete();
