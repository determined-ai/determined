WITH g AS (
  UPDATE project_experiment_groups SET name = $2
  WHERE project_experiment_groups.id = $1
  RETURNING id, name, project_id
)
SELECT g.id, g.name, g.project_id
FROM g
