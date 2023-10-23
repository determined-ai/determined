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
BEFORE INSERT OR UPDATE
ON raw_steps
FOR EACH ROW EXECUTE PROCEDURE stream_metric_seq_modify();

DROP TRIGGER IF EXISTS stream_metric_raw_validations_trigger_seq ON raw_validations;
CREATE TRIGGER stream_metric_raw_validations_trigger_seq 
BEFORE INSERT OR UPDATE
ON raw_validations
FOR EACH ROW EXECUTE PROCEDURE stream_metric_seq_modify();

DROP TRIGGER IF EXISTS stream_metric_generic_metrics_trigger_seq ON generic_metrics;
CREATE TRIGGER stream_metric_generic_metrics_trigger_seq 
BEFORE INSERT OR UPDATE
ON generic_metrics
FOR EACH ROW EXECUTE PROCEDURE stream_metric_seq_modify();


-- helper function to create metric jsonb object for streaming 
CREATE OR REPLACE FUNCTION stream_metric_notify(
    before jsonb, beforework integer, beforeeid integer,after jsonb, afterwork integer, aftereid integer
) RETURNS integer AS $$
DECLARE
    output jsonb = NULL;
    temp jsonb = NULL;
BEGIN
    IF before IS NOT NULL THEN
        temp = before || jsonb_object_agg('workspace_id', beforework);
        temp = before || jsonb_object_agg('experiment_id', beforeeid);
        output = jsonb_object_agg('before', temp);
    END IF;
    IF after IS NOT NULL THEN 
        temp = after || jsonb_object_agg('workspace_id', afterwork);
        temp = after || jsonb_object_agg('experiment_id', aftereid);
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

-- Trigger function to NOTIFY the master of changes, for INSERT
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
        PERFORM stream_metric_notify(NULL, NULL, NULL, to_jsonb(NEW), work, eid);
    ELSEIF (TG_OP = 'UPDATE') THEN
        eid = experiment_id FROM trials WHERE trials.id = NEW.trial_id;
        proj = project_id from experiments where experiments.id = eid;
        work = workspace_id from projects where projects.id = proj;
        PERFORM stream_metric_notify(to_jsonb(OLD), work, eid, to_jsonb(NEW), work, eid);
    ELSEIF (TG_OP = 'DELETE') THEN
        eid = experiment_id FROM trials WHERE trials.id = OLD.trial_id;
        proj = project_id from experiments where experiments.id = eid;
        work = workspace_id from projects where projects.id = proj;
        PERFORM stream_metric_notify(to_jsonb(OLD), work, eid, NULL, NULL, NULL);
        -- DELETE trigger BEFORE, and must return a non-NULL value.
        return OLD;
    END IF;
    return NULL;
END;
$$ LANGUAGE plpgsql;

-- TODO(corban): replaces these triggers with one on the metrics table if we move to postgres >=11 
-- INSERT AND UPDATE should fire AFTER to gurantee to emit the final row value.
DROP TRIGGER IF EXISTS stream_metric_raw_steps_trigger_iu ON raw_steps;
CREATE TRIGGER stream_metric_raw_steps_trigger_iu
AFTER INSERT OR UPDATE
ON raw_steps
FOR EACH ROW EXECUTE PROCEDURE stream_metric_change();

DROP TRIGGER IF EXISTS stream_metric_raw_validations_trigger_iu ON raw_validations;
CREATE TRIGGER stream_metric_raw_validations_trigger_iu
AFTER INSERT OR UPDATE
ON raw_validations
FOR EACH ROW EXECUTE PROCEDURE stream_metric_change();

DROP TRIGGER IF EXISTS stream_metric_generic_metrics_trigger_iu ON generic_metrics;
CREATE TRIGGER stream_metric_generic_metrics_trigger_iu
AFTER INSERT OR UPDATE
ON generic_metrics
FOR EACH ROW EXECUTE PROCEDURE stream_metric_change();

-- TODO(corban): replaces these triggers with one on the metrics table if we move to postgres >=11 
-- DELETE should fire BEFORE to guarantee the experiment still exists to grab the workspace_id.
DROP TRIGGER IF EXISTS stream_metric_raw_steps_trigger_d ON raw_steps;
CREATE TRIGGER stream_metric_raw_steps_trigger_d
BEFORE DELETE ON raw_steps
FOR EACH ROW EXECUTE PROCEDURE stream_metric_change();

DROP TRIGGER IF EXISTS stream_metric_raw_validations_trigger_d ON raw_validations;
CREATE TRIGGER stream_metric_raw_validations_trigger_d
BEFORE DELETE ON raw_validations
FOR EACH ROW EXECUTE PROCEDURE stream_metric_change();

DROP TRIGGER IF EXISTS stream_metric_generic_metrics_trigger_d ON generic_metrics;
CREATE TRIGGER stream_metric_generic_metrics_trigger_d
BEFORE DELETE ON generic_metrics
FOR EACH ROW EXECUTE PROCEDURE stream_metric_change();

-- Trigger for detecting metric permission scope changes derived from projects.workspace_id.
CREATE OR REPLACE FUNCTION stream_metric_workspace_change_notify() RETURNS TRIGGER AS $$
DECLARE
    metric RECORD;
    jmetric jsonb;
BEGIN 
    FOR metric IN 
    SELECT
        m.*,
        p.workspace_id,
        t.experiment_id
        FROM
            projects p
            INNER JOIN
                experiment e ON e.project_id = p.id
            INNER JOIN
                trials t ON t.experiment_id = e.id
            INNER JOIN
                metrics m ON m.trial_id = t.id
        WHERE
            p.id = NEW.id
    LOOP
        metric.seq = nextval('stream_metric_seq');
        UPDATE metrics SET seq = metric.seq where id = metric.id;
        jmetric = to_jsonb(metric);
        PERFORM stream_metric_notify(jmetric, OLD.workspace_id, OLD.experiment_id, jmetric, NEW.workspace_id, NEW.experiment_id);
    END LOOP;
    -- return value for AFTER triggers is ignored
    return NULL;
END;
$$ LANGUAGE plpgsql;
--
DROP TRIGGER IF EXISTS stream_metric_workspace_change_trigger ON projects;
CREATE TRIGGER stream_metric_workspace_change_trigger
AFTER UPDATE OF workspace_id ON projects
FOR EACH ROW EXECUTE PROCEDURE stream_metric_workspace_change_notify();

-- Trigger for decting metric permission scope changes derived from experiments.project_id.
CREATE OR REPLACE FUNCTION stream_metric_project_change_notify() RETURNS TRIGGER AS $$
DECLARE 
    metric RECORD;
    jmetric jsonb;
    eid integer;
    proj integer;
    oldwork integer;
    newwork integer;
BEGIN
    FOR metric IN
        SELECT
            m.*
        FROM
            experiments e 
            INNER JOIN
                trials t ON t.experiment_id = e.id
            INNER JOIN
                metrics m ON m.trial_id = t.id 
            WHERE
                e.id = NEW.id
    LOOP
        proj = project_id from experiments where experiments.id = OLD.id;
        oldwork = workspace_id from projects where projects.id = proj;
        proj = project_id from experiments where experiments.id = NEW.id;
        newwork = workspace_id from projects where projects.id = proj;
        metric.seq = nextval('stream_metric_seq');
        UPDATE metrics SET seq = metric.seq where id = metric.id;
        jmetric = to_jsonb(metric);
        PERFORM stream_metric_notify(jmetric, oldwork, eid, jmetric, newwork, eid);
    END LOOP;
    -- return value for AFTER triggers is ignored
    return NULL;
END;
$$ LANGUAGE plpgsql;
--
DROP TRIGGER IF EXISTS stream_metric_project_change_trigger ON experiments;
CREATE TRIGGER stream_metric_project_change_trigger
AFTER UPDATE OF project_id ON experiments
FOR EACH ROW EXECUTE PROCEDURE stream_metric_project_change_notify();
