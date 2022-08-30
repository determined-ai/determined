WITH pe AS (
  SELECT project_id, state, start_time
  FROM experiments
  UNION SELECT null, null, null
),
w AS (
  SELECT archived
  FROM workspaces
  WHERE $1 = 0 OR id = $1
)
SELECT p.id, p.name, p.workspace_id, p.description, p.immutable, p.notes,
  'WORKSPACE_STATE_' || p.state AS state, p.error_message,
  (w.archived OR p.archived) AS archived,
  SUM(case when pe.project_id = p.id then 1 else 0 end) AS num_experiments,
  SUM(case when pe.project_id = p.id AND pe.state = 'ACTIVE' then 1 else 0 end) AS num_active_experiments,
  MAX(case when pe.project_id = p.id then pe.start_time else NULL end) AS last_experiment_started_at,
  u.username, p.user_id
FROM pe, w, projects as p
  LEFT JOIN users as u ON u.id = p.user_id
  WHERE $1 = 0 OR p.workspace_id = $1
  AND ($2 = '' OR (u.username IN (SELECT unnest(string_to_array($2, ',')))))
  AND ($3 = '' OR p.name ILIKE $3)
  AND ($4 = '' OR p.archived = $4::BOOL)
GROUP BY p.id, u.username, p.user_id, w.archived
ORDER BY %s;
