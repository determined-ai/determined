ALTER TABLE projects ADD COLUMN max_local_id int NOT NULL DEFAULT 0;
ALTER TABLE runs ADD COLUMN local_id int;

UPDATE runs as rn SET local_id = lids.local_id FROM (SELECT r.id as id, ROW_NUMBER() OVER(PARTITION BY p.id) as local_id FROM projects p JOIN runs r ON r.project_id=p.id) lids WHERE rn.id=lids.id;
UPDATE projects as pr SET max_local_id =  cnt.max_cnt FROM (SELECT p.id as id, COUNT(*) as max_cnt FROM projects p JOIN runs r ON r.project_id=p.id GROUP BY p.id) as cnt WHERE pr.id=cnt.id;

CREATE TABLE local_id_redirect (
    run_id int REFERENCES runs(id) ON DELETE CASCADE,
	project_id int REFERENCES projects(id) ON DELETE CASCADE,
    project_key text NOT NULL,
    local_id int NOT NULL
);
INSERT INTO local_id_redirect(run_id, project_id, project_key, local_id)
SELECT r.id, p.id, p.key, r.local_id FROM runs r JOIN projects p ON r.project_id=p.id
