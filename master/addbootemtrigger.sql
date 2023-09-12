-- helper function to create bootstream jsonb object for handling permission scope change
CREATE OR REPLACE FUNCTION permission_change_notify() RETURNS integer AS $$
DECLARE
BEGIN
    PERFORM pg_notify('permission_change_chan');
    return 0;
END;
$$ LANGUAGE plpgsql;