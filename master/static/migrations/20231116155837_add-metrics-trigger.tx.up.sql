-- add sequence numbers to metrics
ALTER TABLE metrics ADD COLUMN IF NOT EXISTS seq bigint;
CREATE SEQUENCE IF NOT EXISTS stream_metric_seq START 1;

-- trigger function to update the sequence number on metric row modfication
-- this should fire before so that it can modify new directly
CREATE OR REPLACE FUNCTION stream_metric_seq_modify() RETURNS TRIGGER AS $$
BEGIN
    NEW.seq = nextval('stream_metric_seq');
RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- TODO(corban): replaces these triggers with one on the metrics table if we move to postgres >=11
DROP TRIGGER IF EXISTS stream_metric_raw_steps_trigger_seq ON raw_steps;
CREATE TRIGGER stream_metric_raw_steps_trigger_seq
    BEFORE INSERT
                ON raw_steps
                FOR EACH ROW EXECUTE PROCEDURE stream_metric_seq_modify();

DROP TRIGGER IF EXISTS stream_metric_raw_validations_trigger_seq ON raw_validations;
CREATE TRIGGER stream_metric_raw_validations_trigger_seq
    BEFORE INSERT
                ON raw_validations
                FOR EACH ROW EXECUTE PROCEDURE stream_metric_seq_modify();

DROP TRIGGER IF EXISTS stream_metric_generic_metrics_trigger_seq ON generic_metrics;
CREATE TRIGGER stream_metric_generic_metrics_trigger_seq
    BEFORE INSERT
                ON generic_metrics
                FOR EACH ROW EXECUTE PROCEDURE stream_metric_seq_modify();


-- helper function to create metric jsonb object for streaming
CREATE OR REPLACE FUNCTION stream_metric_notify(
    before jsonb, beforework integer, after jsonb, afterwork integer
) RETURNS integer AS $$
DECLARE
    output jsonb = NULL;
    temp jsonb = NULL;
BEGIN
    IF after IS NOT NULL THEN
        temp = after || jsonb_object_agg('workspace_id', afterwork);
        IF output IS NULL THEN
            output = jsonb_object_agg('after', temp);
        ELSE
            output = output || jsonb_object_agg('after', temp);
        END IF;
    END IF;
    PERFORM pg_notify('stream_metric_chan', output::text);
    return 0;
END;
$$ LANGUAGE plpgsql;

-- Trigger function to NOTIFY the master of new metric inserts
CREATE OR REPLACE FUNCTION stream_metric_change() RETURNS TRIGGER AS $$
DECLARE
    eid  integer;
    proj integer;
    work integer;
BEGIN
    IF (TG_OP = 'INSERT') THEN
        eid = experiment_id FROM trials WHERE trials.id = NEW.trial_id;
        proj = project_id from experiments where experiments.id = eid;
        work = workspace_id from projects where projects.id = proj;
        PERFORM stream_metric_notify(NULL, NULL, to_jsonb(NEW), work);
    END IF;
    return NULL;
    END;
$$ LANGUAGE plpgsql;

-- TODO(corban): replaces these triggers with one on the metrics table if we move to postgres >=11
-- INSERT should fire AFTER to gurantee to emit the final row value.
DROP TRIGGER IF EXISTS stream_metric_raw_steps_trigger_i ON raw_steps;
CREATE TRIGGER stream_metric_raw_steps_trigger_i
    AFTER INSERT 
                ON raw_steps
                FOR EACH ROW EXECUTE PROCEDURE stream_metric_change();

DROP TRIGGER IF EXISTS stream_metric_raw_validations_trigger_i ON raw_validations;
CREATE TRIGGER stream_metric_raw_validations_trigger_i
    AFTER INSERT
                ON raw_validations
                FOR EACH ROW EXECUTE PROCEDURE stream_metric_change();

DROP TRIGGER IF EXISTS stream_metric_generic_metrics_trigger_i ON generic_metrics;
CREATE TRIGGER stream_metric_generic_metrics_trigger_i
    AFTER INSERT
                ON generic_metrics
                FOR EACH ROW EXECUTE PROCEDURE stream_metric_change();
