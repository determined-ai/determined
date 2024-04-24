ALTER TABLE projects ADD COLUMN max_local_id int NOT NULL DEFAULT 0;
ALTER TABLE runs ADD COLUMN local_id int;

UPDATE runs as rn SET local_id = lids.local_id FROM (SELECT r.id as id, ROW_NUMBER() OVER(PARTITION BY p.id) as local_id FROM projects p JOIN runs r ON r.project_id=p.id) lids WHERE rn.id=lids.id;
UPDATE projects as pr SET max_local_id =  cnt.max_cnt FROM (SELECT p.id as id, COUNT(*) as max_cnt FROM projects p JOIN runs r ON r.project_id=p.id GROUP BY p.id) as cnt WHERE pr.id=cnt.id;

CREATE OR REPLACE FUNCTION autoupdate_local_id_on_project_id_update() RETURNS trigger AS $$
BEGIN

UPDATE projects SET max_local_id = max_local_id + 1 WHERE id = NEW.project_id;
UPDATE runs SET local_id = (SELECT max_local_id FROM projects WHERE id = NEW.project_id) WHERE id=NEW.id;
RETURN NEW;

END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER autoupdate_local_id_on_project_id_update
AFTER UPDATE OF project_id ON runs
FOR EACH ROW EXECUTE PROCEDURE autoupdate_local_id_on_project_id_update();
