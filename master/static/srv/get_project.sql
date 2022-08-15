WITH pe AS (
  SELECT
    COUNT(*) AS num_experiments,
    SUM(case when state = 'ACTIVE' then 1 else 0 end) AS num_active_experiments,
    MAX(start_time) AS last_experiment_started_at
  FROM experiments
  WHERE project_id = $1
)
SELECT p.id, p.name, p.workspace_id, p.description, p.immutable, p.notes, w.name as workspace_name,
  'WORKSPACE_STATE_' || p.state AS state, p.error_message,
  (p.archived OR w.archived) AS archived,
  MAX(pe.num_experiments) AS num_experiments,
  MAX(pe.num_active_experiments) AS num_active_experiments,
  COALESCE(MAX(pe.last_experiment_started_at), NULL) AS last_experiment_started_at,
  u.username, p.user_id
FROM pe, projects as p
  LEFT JOIN users as u ON u.id = p.user_id
  LEFT JOIN workspaces AS w on w.id = p.workspace_id
WHERE p.id = $1
GROUP BY p.id, u.username, w.archived, w.name;
