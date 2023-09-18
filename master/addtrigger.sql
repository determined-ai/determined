ALTER TABLE trials ADD COLUMN IF NOT EXISTS seq bigint;

-- the sequence
CREATE SEQUENCE IF NOT EXISTS stream_trial_seq START 1;

-- trigger function to update sequence number on row modification
-- this should fire BEFORE so that it can modify NEW directly.
CREATE OR REPLACE FUNCTION stream_trial_seq_modify() RETURNS TRIGGER AS $$
BEGIN
    NEW.seq = nextval('stream_trial_seq');
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
--
DROP TRIGGER IF EXISTS stream_trial_trigger_seq ON trials;
CREATE TRIGGER stream_trial_trigger_seq
BEFORE INSERT OR UPDATE OF
    state, start_time, end_time, runner_state, restarts, tags
ON trials
FOR EACH ROW EXECUTE PROCEDURE stream_trial_seq_modify();

-- helper function to create trial jsonb object for streaming
CREATE OR REPLACE FUNCTION stream_trial_notify(
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
    PERFORM pg_notify('stream_trial_chan', output::text);
    -- seems necessary I guess
    return 0;
END;
$$ LANGUAGE plpgsql;

-- Trigger function to NOTIFY the master of changes, for INSERT/UPDATE/DELETE.
CREATE OR REPLACE FUNCTION stream_trial_change() RETURNS TRIGGER AS $$
DECLARE
    proj integer;
    work integer;
BEGIN
    IF (TG_OP = 'INSERT') THEN
        proj = project_id from experiments where experiments.id = NEW.experiment_id;
        work = workspace_id from projects where projects.id = proj;
        PERFORM stream_trial_notify(NULL, NULL, to_jsonb(NEW), work);
    ELSEIF (TG_OP = 'UPDATE') THEN
        proj = project_id from experiments where experiments.id = NEW.experiment_id;
        work = workspace_id from projects where projects.id = proj;
        PERFORM stream_trial_notify(to_jsonb(OLD), work, to_jsonb(NEW), work);
    ELSEIF (TG_OP = 'DELETE') THEN
        proj = project_id from experiments where experiments.id = OLD.experiment_id;
        work = workspace_id from projects where projects.id = proj;
        PERFORM stream_trial_notify(to_jsonb(OLD), work, NULL, NULL);
        -- DELETEs trigger BEFORE, and must return a non-NULL value.
        return OLD;
    END IF;
    return NULL;
END;
$$ LANGUAGE plpgsql;
-- INSERT and UPDATE should fire AFTER to guarantee to emit the final row value.
DROP TRIGGER IF EXISTS stream_trial_trigger_iu ON trials;
CREATE TRIGGER stream_trial_trigger_iu
AFTER INSERT OR UPDATE OF
    state, start_time, end_time, runner_state, restarts, tags
ON trials
FOR EACH ROW EXECUTE PROCEDURE stream_trial_change();
-- DELETE should fire BEFORE to guarantee the experiment still exists to grab the workspace_id.
DROP TRIGGER IF EXISTS stream_trial_trigger_d ON trials;
CREATE TRIGGER stream_trial_trigger_d
BEFORE DELETE ON trials
FOR EACH ROW EXECUTE PROCEDURE stream_trial_change();

-- Trigger for detecting trial permission scope changes derived from projects.workspace_id.
CREATE OR REPLACE FUNCTION stream_trial_workspace_change_notify() RETURNS TRIGGER AS $$
DECLARE
    trial RECORD;
    jtrial jsonb;
BEGIN
    FOR trial IN
    SELECT
            t.*,
            workspace_id
        FROM
            projects p
			INNER JOIN
				experiments e ON e.project_id = p.id
            INNER JOIN 
                trials t ON t.experiment_id = e.id 
            WHERE
                p.id = NEW.id
    LOOP
        trial.seq = nextval('stream_trial_seq');
        UPDATE trials SET seq = trial.seq where id = trial.id;
        jtrial = to_jsonb(trial);
        PERFORM stream_trial_notify(jtrial, OLD.workspace_id, jtrial, NEW.workspace_id);
    END LOOP;
    -- return value for AFTER triggers is ignored
    return NULL;
END;
$$ LANGUAGE plpgsql;
--
DROP TRIGGER IF EXISTS stream_trial_workspace_change_trigger ON projects;
CREATE TRIGGER stream_trial_workspace_change_trigger
AFTER UPDATE OF workspace_id ON projects
FOR EACH ROW EXECUTE PROCEDURE stream_trial_workspace_change_notify();

-- Trigger for detecting trial permission scope changes derived from experiment.project_id.
CREATE OR REPLACE FUNCTION stream_trial_project_change_notify() RETURNS TRIGGER AS $$
DECLARE
    trial RECORD;
    jtrial jsonb;
    proj integer;
    oldwork integer;
    newwork integer;
BEGIN
    FOR trial IN
        SELECT
            t.*
        FROM
            experiments e
            INNER JOIN 
                trials t ON t.experiment_id = e.id 
            WHERE
                e.id = NEW.id
    LOOP
        proj = project_id from experiments where experiments.id = OLD.id;
        oldwork = workspace_id from projects where projects.id = proj;
        proj = project_id from experiments where experiments.id = NEW.id;
        newwork = workspace_id from projects where projects.id = proj;
        trial.seq = nextval('stream_trial_seq');
        UPDATE trials SET seq = trial.seq where id = trial.id;
        jtrial = to_jsonb(trial);
        PERFORM stream_trial_notify(jtrial, oldwork, jtrial, newwork);
    END LOOP;
    -- return value for AFTER triggers is ignored
    return NULL;
END;
$$ LANGUAGE plpgsql;
--
DROP TRIGGER IF EXISTS stream_trial_project_change_trigger ON experiments;
CREATE TRIGGER stream_trial_project_change_trigger
AFTER UPDATE OF project_id ON experiments
FOR EACH ROW EXECUTE PROCEDURE stream_trial_project_change_notify();
