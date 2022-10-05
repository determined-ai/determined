WITH g AS (
  INSERT INTO experiment_groups (name, project_id)
  VALUES ($1, $2)
  RETURNING id, name, project_id
), ge AS (
  SELECT COUNT(*) AS num_experiments
  FROM experiments, g
  WHERE group_id = g.id
)
SELECT g.id, g.name, g.project_id, ge.num_experiments AS num_experiments
FROM g, ge