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
CREATE OR REPLACE FUNCTION stream_trial_body(name text, trial jsonb, project_id integer) RETURNS jsonb AS $$
DECLARE
    temp jsonb;
BEGIN
    temp := trial || jsonb_object_agg('project_id', project_id);
    return jsonb_object_agg(name, temp);
END;
$$ LANGUAGE plpgsql;

-- Trigger function to NOTIFY the master of changes, for INSERT and UPDATE.
-- This should fire AFTER so that it is guaranteed to emit the final row value.
CREATE OR REPLACE FUNCTION stream_trial_iu() RETURNS TRIGGER AS $$
DECLARE
    proj integer;
BEGIN
    proj := project_id from experiments where experiments.id = NEW.experiment_id;
    PERFORM pg_notify(
        'stream_trial_chan',
        (
            stream_trial_body('old', to_jsonb(OLD), proj)
            || stream_trial_body('new', to_jsonb(NEW), proj)
        )::text
    );
    -- return value for AFTER triggers is ignored
    return NULL;
END;
$$ LANGUAGE plpgsql;
--
DROP TRIGGER IF EXISTS stream_trial_trigger_iu ON trials;
CREATE TRIGGER stream_trial_trigger_iu
AFTER INSERT OR UPDATE OF
    state, start_time, end_time, runner_state, restarts, tags
ON trials
FOR EACH ROW EXECUTE PROCEDURE stream_trial_iu();

-- Trigger function to NOTIFY the master of changes, for DELETE.
-- This should fire BEFORE so that it is guaranteed we still have access to the
-- relevant experiment to look up the project_id.
CREATE OR REPLACE FUNCTION stream_trial_d() RETURNS TRIGGER AS $$
DECLARE
    project_id integer;
BEGIN
    project_id = project_id from experiments where experiment_id = OLD.id;
    PERFORM pg_notify(
        'stream_trial_chan',
        stream_trial_body('old', to_jsonb(OLD), project_id)::text
    );
    -- return any non-NULL value, because NULL would stop the DELETE from happening.
    return OLD;
END;
$$ LANGUAGE plpgsql;
--
DROP TRIGGER IF EXISTS stream_trial_trigger_d ON trials;
CREATE TRIGGER stream_trial_trigger_d
BEFORE DELETE ON trials
FOR EACH ROW EXECUTE PROCEDURE stream_trial_d();

-- Trigger for detecting trial ownership changes based derived from experiments.project_id.
CREATE OR REPLACE FUNCTION stream_trial_ownership() RETURNS TRIGGER AS $$
DECLARE
    trial RECORD;
    jtrial jsonb;
BEGIN
    FOR trial IN
        SELECT * from trials where experiment_id = NEW.id
    LOOP
        trial.seq = nextval('stream_trial_seq');
        UPDATE trials SET seq = trial.seq where id = trial.id;
        jtrial = to_jsonb(trial);
        PERFORM pg_notify(
            'stream_trial_chan',
            (
                stream_trial_body('old', jtrial, OLD.project_id)
                || stream_trial_body('new', jtrial, NEW.project_id)
            )::text
        );
    END LOOP;
    -- return value for AFTER triggers is ignored
    return NULL;
END;
$$ LANGUAGE plpgsql;
--
DROP TRIGGER IF EXISTS stream_trial_trigger_ownership ON experiments;
CREATE TRIGGER stream_trial_trigger_ownership
AFTER UPDATE OF project_id ON EXPERIMENTS
FOR EACH ROW EXECUTE PROCEDURE stream_trial_ownership();
