-- update runs_metadata_index table when project_id is updated in runs table
CREATE OR REPLACE FUNCTION determined_code.update_run_metadata_index_project_id()
RETURNS TRIGGER AS $$
BEGIN
UPDATE runs_metadata_index
    SET project_id = NEW.project_id
    WHERE run_id = NEW.id;
RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- create trigger to update runs_metadata_index table when project_id is updated in runs table
DROP TRIGGER IF EXISTS trigger_update_run_metadata_index_project_id ON runs;
CREATE TRIGGER trigger_update_run_metadata_index_project_id
AFTER UPDATE OF project_id ON runs
FOR EACH ROW EXECUTE PROCEDURE update_run_metadata_index_project_id();
