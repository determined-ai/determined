ALTER TABLE runs REMOVE COLUMN metadata;
DROP TABLE IF EXISTS runs_metadata_index;
DROP FUNCTION IF EXISTS func_update_run_metadata_index_project_id();
DROP TRIGGER IF EXISTS trigger_update_run_metadata_index_project_id;

