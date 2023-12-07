ALTER TABLE checkpoints_v2 ADD COLUMN IF NOT EXISTS seq bigint;

-- the sequence
CREATE SEQUENCE IF NOT EXISTS stream_checkpoint_seq START 1;

-- trigger function to update sequence number on row modification
-- this should fire BEFORE so that it can modify NEW directly.
CREATE OR REPLACE FUNCTION stream_checkpoint_seq_modify() RETURNS TRIGGER AS $$
BEGIN
    NEW.seq = nextval('stream_checkpoint_seq');
RETURN NEW;
END;
$$ LANGUAGE plpgsql;
--
DROP TRIGGER IF EXISTS stream_checkpoint_trigger_seq ON checkpoints_v2;
CREATE TRIGGER stream_checkpoint_trigger_seq
    BEFORE INSERT OR UPDATE OF
    task_id, allocation_id, report_time, state, resources, metadata, size
                     ON checkpoints_v2
                         FOR EACH ROW EXECUTE PROCEDURE stream_checkpoint_seq_modify();

-- helper function to create a checkpoint jsonb object for streaming
CREATE OR REPLACE FUNCTION stream_checkpoint_notify(
    before jsonb, beforework integer, after jsonb, afterwork integer, trialid integer, expid integer
) RETURNS integer AS $$
DECLARE
output jsonb = NULL;
    temp jsonb = NULL;
BEGIN
    IF before IS NOT NULL THEN
        temp = before || jsonb_object_agg('workspace_id', beforework);
        temp = temp || jsonb_build_object('trial_id', trialid);
        temp = temp || jsonb_build_object('experiment_id', expid);
        output = jsonb_object_agg('before', temp);
    END IF;
    IF after IS NOT NULL THEN
        temp = after || jsonb_object_agg('workspace_id', afterwork);
        temp = temp || jsonb_build_object('trial_id', trialid);
        temp = temp || jsonb_build_object('experiment_id', expid);
        IF output IS NULL THEN
            output = jsonb_object_agg('after', temp);
        ELSE
            output = output || jsonb_object_agg('after', temp);
    END IF;
END IF;
    PERFORM pg_notify('stream_checkpoint_chan', output::text);
    -- seems necessary I guess
return 0;
END;
$$ LANGUAGE plpgsql;

-- Trigger function to NOTIFY the master of changes, for INSERT/UPDATE/DELETE.
CREATE OR REPLACE FUNCTION stream_checkpoint_change() RETURNS TRIGGER AS $$
DECLARE
trialid integer;
exp integer;
proj integer;
work integer;
BEGIN
    IF (TG_OP = 'INSERT') THEN
        trialid = trial_id from trial_id_task_id where trial_id_task_id.task_id = NEW.task_id;
        exp = experiment_id from trials where trials.id = trialid;
        proj = project_id from experiments where experiments.id = exp;
        work = workspace_id from projects where projects.id = proj;
        PERFORM stream_checkpoint_notify(NULL, NULL, to_jsonb(NEW), work, trialid, exp);
    ELSEIF (TG_OP = 'UPDATE') THEN
        trialid = trial_id from trial_id_task_id where trial_id_task_id.task_id = NEW.task_id;
        exp = experiment_id from trials where trials.id = trialid;
        proj = project_id from experiments where experiments.id = exp;
        work = workspace_id from projects where projects.id = proj;
        PERFORM stream_checkpoint_notify(to_jsonb(OLD), work, to_jsonb(NEW), work, trialid, exp);
    ELSEIF (TG_OP = 'DELETE') THEN
        trialid = trial_id from trial_id_task_id where trial_id_task_id.task_id = OLD.task_id;
        exp = experiment_id from trials where trials.id = trialid;
        proj = project_id from experiments where experiments.id = exp;
        work = workspace_id from projects where projects.id = proj;
        PERFORM stream_checkpoint_notify(to_jsonb(OLD), work,  NULL, NULL, trialid, exp);
        -- DELETEs trigger BEFORE, and must return a non-NULL value.
return OLD;
END IF;
return NULL;
END;
$$ LANGUAGE plpgsql;
-- INSERT and UPDATE should fire AFTER to guarantee to emit the final row value.
DROP TRIGGER IF EXISTS stream_checkpoint_trigger_iu ON checkpoints_v2;
CREATE TRIGGER stream_checkpoint_trigger_iu
    AFTER INSERT OR UPDATE OF
    task_id, allocation_id, report_time, state, resources, metadata, size
                    ON checkpoints_v2
                        FOR EACH ROW EXECUTE PROCEDURE stream_checkpoint_change();
-- DELETE should fire BEFORE to guarantee the experiment still exists to grab the workspace_id.
DROP TRIGGER IF EXISTS stream_checkpoint_trigger_d ON checkpoints_v2;
CREATE TRIGGER stream_checkpoint_trigger_d
    BEFORE DELETE ON checkpoints_v2
    FOR EACH ROW EXECUTE PROCEDURE stream_checkpoint_change();

-- Trigger for detecting checkpoint permission scope changes derived from projects.workspace_id.
CREATE OR REPLACE FUNCTION stream_checkpoint_workspace_change_notify() RETURNS TRIGGER AS $$
DECLARE
checkpoint RECORD;
jcheckpoint jsonb;
BEGIN
FOR checkpoint IN
SELECT
    c.*,
    workspace_id
FROM
    projects p
        INNER JOIN
    experiments e ON e.project_id = p.id
        INNER JOIN
    trials t ON t.experiment_id = e.id
        INNER JOIN
    trial_id_task_id tt ON t.trial_id = tt.trial_id
        INNER JOIN
    checkpoints_v2 c ON tt.task_id = c.task_id
WHERE
        p.id = NEW.id
    LOOP
        checkpoint.seq = nextval('stream_checkpoint_seq');
UPDATE checkpoints_v2 SET seq = checkpoint.seq where id = checkpoint.id;
jcheckpoint = to_jsonb(checkpoint);
        PERFORM stream_checkpoint_notify(jcheckpoint, OLD.workspace_id, jcheckpoint, NEW.workspace_id);
END LOOP;
    -- return value for AFTER triggers is ignored
return NULL;
END;
$$ LANGUAGE plpgsql;
--
DROP TRIGGER IF EXISTS stream_checkpoint_workspace_change_trigger ON projects;
CREATE TRIGGER stream_checkpoint_workspace_change_trigger
    AFTER UPDATE OF workspace_id ON projects
    FOR EACH ROW EXECUTE PROCEDURE stream_checkpoint_workspace_change_notify();

-- Trigger for detecting checkpoint permission scope changes derived from experiment.project_id.
CREATE OR REPLACE FUNCTION stream_checkpoint_project_change_notify() RETURNS TRIGGER AS $$
DECLARE
checkpoint RECORD;
    jcheckpoint jsonb;
    proj integer;
    oldwork integer;
    newwork integer;
BEGIN
FOR checkpoint IN
SELECT
    c.*
FROM
    experiments e
        INNER JOIN
    trials t ON t.experiment_id = e.id
        INNER JOIN
    trial_id_task_id tt ON t.trial_id = tt.trial_id
        INNER JOIN
    checkpoints_v2 c ON tt.task_id = c.task_id
WHERE
        e.id = NEW.id
    LOOP
        proj = project_id from experiments where experiments.id = OLD.id;
oldwork = workspace_id from projects where projects.id = proj;
        proj = project_id from experiments where experiments.id = NEW.id;
        newwork = workspace_id from projects where projects.id = proj;
        checkpoint.seq = nextval('stream_checkpoint_seq');
UPDATE checkpoints_v2 SET seq = checkpoint.seq where id = checkpoint.id;
jcheckpoint = to_jsonb(checkpoint);
        PERFORM stream_checkpoint_notify(jcheckpoint, oldwork, jcheckpoint, newwork);
END LOOP;
    -- return value for AFTER triggers is ignored
return NULL;
END;
$$ LANGUAGE plpgsql;
--
DROP TRIGGER IF EXISTS stream_checkpoint_project_change_trigger ON experiments;
CREATE TRIGGER stream_checkpoint_project_change_trigger
    AFTER UPDATE OF project_id ON experiments
    FOR EACH ROW EXECUTE PROCEDURE stream_checkpoint_project_change_notify();
