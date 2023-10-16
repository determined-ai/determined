-- add sequence numbers to allocations
ALTER TABLE allocations ADD COLUMN IF NOT EXISTS seq bigint;
CREATE SEQUENCE IF NOT EXISTS stream_allocation_seq START 1;

-- trigger function to update the sequence number on allocation row modfication
-- this should fire before so that it can modify new directly
CREATE OR REPLACE FUNCTION stream_allocation_seq_modify() RETURNS TRIGGER AS $$ 
BEGIN
    NEW.seq = nextval('stream_allocation_seq');
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS stream_allocation_trigger_seq ON allocations;
CREATE TRIGGER stream_allocation_trigger_seq 
BEFORE INSERT OR UPDATE
ON allocations
FOR EACH ROW EXECUTE PROCEDURE stream_allocation_seq_modify();

-- helper function to create allocation jsonb object for streaming 
CREATE OR REPLACE FUNCTION stream_allocation_notify(
    before jsonb, beforework integer, after jsonb, afterwork integer, eid integer
) RETURNS integer AS $$
DECLARE
    output jsonb = NULL;
    temp jsonb = NULL;
BEGIN
    IF before IS NOT NULL THEN
        temp = before || jsonb_object_agg('workspace_id', beforework);
        temp = before || jsonb_object_agg('experiment_id', eid);
        output = jsonb_object_agg('before', temp);
    END IF;
    IF after IS NOT NULL THEN 
        temp = after || jsonb_object_agg('workspace_id', afterwork);
        temp = after || jsonb_object_agg('experiment_id', eid);
        IF output IS NULL THEN
            output = jsonb_object_agg('after', temp);
        ELSE 
            output = output || jsonb_object_agg('after', temp);
        END IF;
    END IF;
    PERFORM pg_notify('stream_allocation_chan', output::text);
    return 0;
END;
$$ LANGUAGE plpgsql;

-- Trigger function to NOTIFY the master of changes, for INSERT
CREATE OR REPLACE FUNCTION stream_allocation_change() RETURNS TRIGGER AS $$ 
DECLARE
    tid  integer;
    eid  integer;
    proj integer;
    work integer;
BEGIN
    IF (TG_OP = 'INSERT') THEN 
        tid = trial_id from trial_id_task_id WHERE trial_id_task_id.task_id = NEW.task_id;
        eid = experiment_id FROM trials WHERE trials.id = tid;
        proj = project_id from experiments where experiments.id = eid;
        work = workspace_id from projects where projects.id = proj;
        PERFORM stream_allocation_notify(NULL, NULL, to_jsonb(NEW), work, eid);
    ELSEIF (TG_OP = 'UPDATE') THEN
        tid = trial_id from trial_id_task_id WHERE trial_id_task_id.task_id = NEW.task_id;
        eid = experiment_id FROM trials WHERE trials.id = tid;
        proj = project_id from experiments where experiments.id = eid;
        work = workspace_id from projects where projects.id = proj;
        PERFORM stream_allocation_notify(to_jsonb(OLD), work, to_jsonb(NEW), work, eid);
    ELSEIF (TG_OP = 'DELETE') THEN
        tid = trial_id from trial_id_task_id WHERE trial_id_task_id.task_id = OLD.task_id;
        eid = experiment_id FROM trials WHERE trials.id = tid;
        proj = project_id from experiments where experiments.id = eid;
        work = workspace_id from projects where projects.id = proj;
        PERFORM stream_allocation_notify(to_jsonb(OLD), work, NULL, NULL, eid);
        -- DELETE trigger BEFORE, and must return a non-NULL value.
        return OLD;
    END IF;
    return NULL;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS stream_allocation_trigger_iu ON allocations;
CREATE TRIGGER stream_allocation_trigger_iu
AFTER INSERT OR UPDATE
ON allocations
FOR EACH ROW EXECUTE PROCEDURE stream_allocation_change();

DROP TRIGGER IF EXISTS stream_allocation_trigger_d ON allocations;
CREATE TRIGGER stream_allocation_trigger_d
BEFORE DELETE ON allocations
FOR EACH ROW EXECUTE PROCEDURE stream_allocation_change();

-- Trigger for detecting allocation permission scope changes derived from projects.workspace_id.
CREATE OR REPLACE FUNCTION stream_allocation_workspace_change_notify() RETURNS TRIGGER AS $$
DECLARE
    allocation RECORD;
    jallocation jsonb;
BEGIN 
    FOR allocation IN 
    SELECT
        a.*,
        p.workspace_id,
        t.experiment_id
        FROM
            projects p
            INNER JOIN
                experiment e ON e.project_id = p.id
            INNER JOIN
                trials t ON t.experiment_id = e.id
            INNER JOIN
                trial_id_task_id tt ON tt.trial_id = t.id
            INNER JOIN
                allocations a ON a.task_id = tt.task_id
        WHERE
            p.id = NEW.id
    LOOP
        allocation.seq = nextval('stream_allocation_seq');
        UPDATE allocations SET seq = allocation.seq where id = allocation.id;
        jallocation = to_jsonb(allocation);
        PERFORM stream_allocation_notify(jallocation, OLD.workspace_id, jallocation, NEW.workspace_id, allocation.experiment_id);
    END LOOP;
    -- return value for AFTER triggers is ignored
    return NULL;
END;
$$ LANGUAGE plpgsql;
--
DROP TRIGGER IF EXISTS stream_allocation_workspace_change_trigger ON projects;
CREATE TRIGGER stream_allocation_workspace_change_trigger
AFTER UPDATE OF workspace_id ON projects
FOR EACH ROW EXECUTE PROCEDURE stream_allocation_workspace_change_notify();

-- Trigger for decting allocation permission scope changes derived from experiments.project_id.
CREATE OR REPLACE FUNCTION stream_allocation_project_change_notify() RETURNS TRIGGER AS $$
DECLARE 
    allocation RECORD;
    jallocation jsonb;
    eid integer;
    proj integer;
    oldwork integer;
    newwork integer;
BEGIN
    FOR allocation IN
        SELECT
            m.*
        FROM
            experiments e 
            INNER JOIN
                trials t ON t.experiment_id = e.id
            INNER JOIN
                trial_id_task_id tt ON tt.trial_id = t.id
            INNER JOIN
                allocations a ON a.task_id = tt.task_id
            WHERE
                e.id = NEW.id
    LOOP
        proj = project_id from experiments where experiments.id = OLD.id;
        oldwork = workspace_id from projects where projects.id = proj;
        proj = project_id from experiments where experiments.id = NEW.id;
        newwork = workspace_id from projects where projects.id = proj;
        allocation.seq = nextval('stream_allocation_seq');
        UPDATE allocations SET seq = allocation.seq where id = allocation.id;
        jallocation = to_jsonb(allocation);
        PERFORM stream_allocation_notify(jallocation, oldwork, jallocation, newwork, eid);
    END LOOP;
    -- return value for AFTER triggers is ignored
    return NULL;
END;
$$ LANGUAGE plpgsql;
--
DROP TRIGGER IF EXISTS stream_allocation_project_change_trigger ON experiments;
CREATE TRIGGER stream_allocation_project_change_trigger
AFTER UPDATE OF project_id ON experiments
FOR EACH ROW EXECUTE PROCEDURE stream_allocation_project_change_notify();
