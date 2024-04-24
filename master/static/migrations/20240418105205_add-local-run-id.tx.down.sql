DROP TRIGGER autoupdate_local_id_on_project_id_update ON runs;
DROP FUNCTION autoupdate_local_id_on_project_id_update;

ALTER TABLE projects DROP COLUMN max_local_id;
ALTER TABLE runs DROP COLUMN local_id;
