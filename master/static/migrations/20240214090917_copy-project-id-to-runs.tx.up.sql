ALTER TABLE runs ADD COLUMN project_id int REFERENCES projects(id);

UPDATE runs
SET project_id = experiments.project_id
FROM experiments
WHERE runs.experiment_id = experiments.id;

ALTER TABLE runs ALTER COLUMN project_id SET NOT NULL;
