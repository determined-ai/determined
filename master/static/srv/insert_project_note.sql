WITH pe AS (
  SELECT project_id, state, start_time
  FROM experiments
  WHERE project_id = $1
),
p AS (
  UPDATE projects
  SET notes = $2
  WHERE id = $1
  RETURNING id, name, description, notes, user_id, workspace_id, archived
)
SELECT p.id, p.name, p.description, p.notes, p.workspace_id, p.archived,
  COUNT(pe) AS num_experiments,
  SUM(case when pe.state = 'ACTIVE' then 1 else 0 end) AS num_active_experiments,
  MAX(pe.start_time) AS last_experiment_started_at,
  u.username
FROM pe, p
  LEFT JOIN users as u ON u.id = p.user_id;
