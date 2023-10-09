ALTER TABLE experiments ADD COLUMN IF NOT EXISTS seq bigint;
CREATE SEQUENCE IF NOT EXISTS stream_experiment_seq START 1;

-- trigger function to update sequence number on row modification
-- this should fire BEFORE so that it can modify NEW directly.
CREATE OR REPLACE FUNCTION stream_experiment_seq_modify() RETURNS TRIGGER AS $$
BEGIN
    NEW.seq = nextval('stream_experiment_seq');
RETURN NEW;
END;
$$ LANGUAGE plpgsql;
--
DROP TRIGGER IF EXISTS stream_experiment_trigger_seq ON experiments;
CREATE TRIGGER stream_experiment_trigger_seq
    BEFORE INSERT OR UPDATE OF
    state, archived, progress, start_time, end_time, notes
                     ON experiments
                         FOR EACH ROW EXECUTE PROCEDURE stream_experiment_seq_modify();

-- helper function to create exp jsonb object for streaming
CREATE OR REPLACE FUNCTION stream_experiment_notify(
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
    PERFORM pg_notify('stream_experiment_chan', output::text);
    -- seems necessary I guess
    return 0;
END;
$$ LANGUAGE plpgsql;

-- Trigger function to NOTIFY the master of changes, for INSERT/UPDATE/DELETE.
CREATE OR REPLACE FUNCTION stream_experiment_change() RETURNS TRIGGER AS $$
DECLARE
    work integer;
    data jsonb;
    old_data jsonb;
BEGIN
    IF (TG_OP = 'INSERT') THEN
        -- add seq
        -- problematic: config, model def, original config, git stuff
        data = to_jsonb(NEW) - 'config' - 'model_definition' - 'model_packages' - 'git_remote' - 'git_commit' - 'git_committer' - 'git_commit_date' - 'original_config';
        work = workspace_id from projects where projects.id = NEW.project_id;
        PERFORM stream_experiment_notify(NULL, NULL, to_jsonb(data), work);
    ELSEIF (TG_OP = 'UPDATE') THEN
        data = to_jsonb(NEW) - 'config' - 'model_definition' - 'model_packages' - 'git_remote' - 'git_commit' - 'git_committer' - 'git_commit_date' - 'original_config';
        old_data = to_jsonb(OLD) - 'config' - 'model_definition' - 'model_packages' - 'git_remote' - 'git_commit' - 'git_committer' - 'git_commit_date' - 'original_config';
        work = workspace_id from projects where projects.id = NEW.project_id;
        PERFORM stream_experiment_notify(to_jsonb(old_data), work, to_jsonb(data), work);
    ELSEIF (TG_OP = 'DELETE') THEN
        old_data = to_jsonb(OLD) - 'config' - 'model_definition' - 'model_packages' - 'git_remote' - 'git_commit' - 'git_committer' - 'git_commit_date' - 'original_config';
        work = workspace_id from projects where projects.id = OLD.project_id;
        PERFORM stream_experiment_notify(to_jsonb(old_data), work, NULL, NULL);
        -- DELETEs trigger BEFORE, and must return a non-NULL value.
        return OLD;
    END IF;
    return NULL;
END;
$$ LANGUAGE plpgsql;
-- INSERT and UPDATE should fire AFTER to guarantee to emit the final row value.
DROP TRIGGER IF EXISTS stream_experiment_trigger_iu ON experiments;
CREATE TRIGGER stream_experiment_trigger_iu
    AFTER INSERT OR UPDATE OF
    state, archived, progress, start_time, end_time, notes
                    ON experiments
                        FOR EACH ROW EXECUTE PROCEDURE stream_experiment_change();
-- DELETE should fire BEFORE to guarantee the experiment still exists to grab the workspace_id.
DROP TRIGGER IF EXISTS stream_experiment_trigger_d ON experiments;
CREATE TRIGGER stream_experiment_trigger_d
    BEFORE DELETE ON experiments
    FOR EACH ROW EXECUTE PROCEDURE stream_experiment_change();

-- Trigger for detecting experiment permission scope changes derived from projects.workspace_id.
CREATE OR REPLACE FUNCTION stream_experiment_workspace_change_notify() RETURNS TRIGGER AS $$
DECLARE
    experiment RECORD;
    jexp jsonb;
BEGIN
FOR experiment IN
SELECT
    e.*
FROM
    experiments e
        INNER JOIN
    projects p ON e.project_id = p.id
WHERE
        p.id = NEW.id
    LOOP
    experiment.seq = nextval('stream_experiment_seq');
    UPDATE experiments SET seq = experiment.seq where id = experiment.id;
    jexp = to_jsonb(experiment);
    PERFORM stream_experiment_notify(jexp, OLD.workspace_id, jexp, NEW.workspace_id);
END LOOP;
    return NULL;
END;
$$ LANGUAGE plpgsql;
