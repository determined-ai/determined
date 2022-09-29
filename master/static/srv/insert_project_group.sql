WITH g AS (
  INSERT INTO experiment_groups (project_id, name)
  VALUES ($1, $2)
  RETURNING id, name, project_id
)
SELECT g.id, g.name, g.project_id
FROM g
