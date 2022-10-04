WITH g AS (
  UPDATE experiment_groups SET name = $2
  WHERE experiment_groups.id = $1
  RETURNING id, name, project_id
),
ge AS (
  SELECT COUNT(*) AS num_experiments
  FROM experiments
  WHERE group_id = $1
)
SELECT g.id, g.name, g.project_id, ge.num_experiments AS num_experiments
FROM ge, g
