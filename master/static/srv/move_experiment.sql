WITH e AS (
  SELECT id, project_id FROM experiments
  WHERE id = $1
),
p AS (
  SELECT projects.id FROM projects, e
  WHERE projects.id = e.project_id
  AND NOT projects.archived
)
UPDATE experiments SET project_id = $2
WHERE experiments.id = (SELECT id FROM e)
AND experiments.project_id = (SELECT id FROM p)
RETURNING experiments.id;
