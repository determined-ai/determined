CREATE VIEW determined_code.checkpoints_view AS
SELECT c.id,
    c.uuid,
    c.task_id,
    c.allocation_id,
    c.report_time,
    c.state,
    c.resources,
    c.metadata,
    r.id AS trial_id,
    e.id AS experiment_id,
    e.config AS experiment_config,
    r.hparams,
    s.metrics AS training_metrics,
    v.metrics -> 'validation_metrics'::text AS validation_metrics,
    ((v.metrics -> 'validation_metrics'::text) ->> ((e.config -> 'searcher'::text) ->> 'metric'::text))::double precision AS searcher_metric,
    (c.metadata ->> 'steps_completed'::text)::integer AS steps_completed,
    c.size,
    c.storage_id
   FROM checkpoints_v2 c
     LEFT JOIN run_checkpoints rc ON rc.checkpoint_id = c.uuid
     LEFT JOIN runs r ON r.id = rc.run_id
     LEFT JOIN experiments e ON r.experiment_id = e.id
     LEFT JOIN raw_validations v ON ((c.metadata ->> 'steps_completed'::text)::integer) = v.total_batches AND r.id = v.trial_id AND NOT v.archived
     LEFT JOIN raw_steps s ON ((c.metadata ->> 'steps_completed'::text)::integer) = s.total_batches AND r.id = s.trial_id AND NOT s.archived;

CREATE VIEW determined_code.proto_checkpoints_view AS
SELECT c.uuid,
    c.task_id,
    c.allocation_id,
    c.report_time,
    'STATE_'::text || c.state AS state,
    c.resources,
    c.metadata,
    c.storage_id,
    jsonb_build_object('trial_id', c.trial_id, 'experiment_id', c.experiment_id, 'experiment_config', c.experiment_config, 'hparams', c.hparams, 'training_metrics', jsonb_build_object('avg_metrics', c.training_metrics -> 'avg_metrics'::text, 'batch_metrics', c.training_metrics -> 'batch_metrics'::text), 'validation_metrics', json_build_object('avg_metrics', c.validation_metrics), 'searcher_metric', c.searcher_metric) AS training
   FROM checkpoints_view c;

CREATE FUNCTION determined_code.abort_checkpoint_delete() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    IF OLD.state <> 'DELETED' THEN
        RETURN NULL;
    END IF;
   RETURN OLD;
END
$$;
CREATE TRIGGER on_checkpoint_deletion BEFORE DELETE ON checkpoints_v2 FOR EACH ROW EXECUTE PROCEDURE abort_checkpoint_delete();
