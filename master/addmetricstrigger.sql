-- add sequence numbers to metrics
ALTER TABLE metrics ADD COLUMN IF NOT EXISTS seq bigint;
CREATE SEQUENCE IF NOT EXISTS stream_metric_seq START 1;

-- trigger fucntion to update teh sequence number on metric row modfication
-- thsi should fire before so that it can modify new directly
CREATE OR REPLACE FUNCTION stream_metric_seq_modify() RETURNS TRIGGER AS $$ 
BEGIN  
    NEW.seq = nextval('stream_metrics_seq');
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS stream_metric_trigger_seq ON metrics;
CREATE TRIGGER stream_metric_trigger_seq 
BEFORE INSERT OR UPDATE
ON metrics
FOR EACH ROW EXECUTE PROCEDURE stream_metric_seq_modify();

-- helper function to create metric jsonb object for streaming 
CREATE OR REPLACE FUNCTION stream_metric_notify(
    before jsonb, beforework integer, after jsonb, afterwork integer
) RETURNS integer AS $$
DECLARE
    output jsonb = NULL;
    temp jsonb = NULL;
BEGIN
    IF before IS NOT NULL THEN
        temp = before || jsonb_object_agg('workspace_id', beforework);
        output = jsonb_object_agg('before', temp);
    END IF;
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

-- Trigger function to NOTIFY the master of changes, for INSERT
CREATE OR RPELACE FUNCTION stream_metric_change() RETURNS TRIGGER AS $$ 
DECLARE
    eid  integer;
    oldproj integer;
    oldwork integer;
    newproj integer;
    newwork integer;
BEGIN
    IF (TG_OP = 'INSERT') THEN 
        eid = experiment_id FROM trials WHERE trials.id = NEW.trial_id;
        proj = project_id from experiments where experiments.id = eid;
        work = workspace_id from projects where projects.id = proj;
        PERFORM stream_metric_notify(NULL, NULL, to_jsonb(NEW), work);
    ELSEIF (TG_OP = 'UPDATE') THEN
        eid = experiment_id FROM trials WHERE trials.id = NEW.trial_id;
        proj = project_id from experiments where experiments.id = eid;
        work = workspace_id from projects where projects.id = proj;
        PERFORM stream_metric_notify(to_jsonb(OLD), work, to_jsonb(NEW), work);
    END IF;
    return NULL;
END;
$$ LANGUAGE plpgsql;

-- INSERT AND UPDATE should fire AFTER to gurantee to emit the final row value.
DROP TRIGGER IF EXISTS stream_metric_trigger_iu ON metrics;
CREATE TRIGGER stream_metric_trigger_iu
AFTER INSERT OR UPDATE
ON metrics
FOR EACH ROW EXECUTE PROCEDURE stream_metric_change();

-- Trigger for detecting metric permission scope changes derived from projects.workspace_id.
CREATE OR REPLACE FUNCTION stream_metric_workspace_change_notify() RETURN TRIGGER $$
DECLARE
    metric RECORD:
    jmetric jsonb;
BEGIN 
    FOR metric IN 
    SELECT
        m.*,
        workspace_id
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
        PERFORM stream_metric_notify(jmetric, OLD.workspace_id, jmetric, NEW.workspace_id);
    END LOOP;
    -- return value for AFTER triggers is ignored
    return NULL;
END;
$$ LANGUAGE plpgsql;
--
DROP TRIGGER IF EXISTS stream_metric_workspace_Change_trigger ON projects;
CREATE TRIGGER stream_metric_workspace_Change_trigger
AFTER UPDATE OF workspace_id ON projects
FOR EACH ROW EXECUTE PROCEDURE stream_trial_workspace_change_notify();

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
    FOR metrics IN
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
        PERFORM stream_metric_notify(jmetric, oldwork, jmetric, newwork);
    END LOOP;
    -- return value for AFTER triggers is ignored
    return NULL;
END;
$$ LANGUAGE plpgsql;
--
DROP TRIGGER IF EXISTS stream_metric_project_change_trigger ON experiments;
CREATE TRIGGER stream_metric_project_change_trigger
AFTER UPDATE OF project_id ON experiments
FOR EACH ROW EXECUTE PROCEDURE stream_trial_project_change_notify();

