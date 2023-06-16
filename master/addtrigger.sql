ALTER TABLE trials ADD COLUMN IF NOT EXISTS seq bigint;

-- sequence
CREATE SEQUENCE IF NOT EXISTS stream_trial_seq START 1;

-- insert or update: function
CREATE OR REPLACE FUNCTION stream_trial_iu() RETURNS TRIGGER AS $$
BEGIN
    UPDATE trials SET seq = nextval('stream_trial_seq') WHERE id = NEW.id;
    NOTIFY stream_trial_chan;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

-- insert or update: trigger
DROP TRIGGER IF EXISTS stream_trial_trigger_iu ON trials;
CREATE TRIGGER stream_trial_trigger_iu
AFTER INSERT OR UPDATE OF
    state, start_time, end_time, runner_state, restarts, tags
ON trials
FOR EACH ROW
EXECUTE PROCEDURE stream_trial_iu();
