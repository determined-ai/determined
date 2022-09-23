WITH g AS (
  INSERT INTO project_experiment_groups (name, project_id)
  VALUES ($1, $2)
  RETURNING id, name, project_id
)
SELECT g.id, g.name, g.project_id
FROM g
