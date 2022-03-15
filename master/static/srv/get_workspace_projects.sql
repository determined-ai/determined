WITH pe AS (
  SELECT project_id, state, start_time
  FROM experiments
)
SELECT p.id, p.name, p.workspace_id, p.description, p.archived, p.notes,
  SUM(case when pe.project_id = p.id then 1 else 0 end) AS num_experiments,
  SUM(case when pe.project_id = p.id AND pe.state = 'ACTIVE' then 1 else 0 end) AS num_active_experiments,
  MAX(case when pe.project_id = p.id then pe.start_time else TO_TIMESTAMP(0) end) AS last_experiment_started_at,
  u.username
FROM pe, projects as p
  LEFT JOIN users as u ON u.id = p.user_id
  WHERE p.workspace_id = $1
  AND ($2 = '' OR (u.username IN (SELECT unnest(string_to_array($2, ',')))))
  AND ($3 = '' OR p.name ILIKE $3)
  AND ($4 = '' OR p.archived = $4::BOOL)
GROUP BY p.id, u.username
ORDER BY %s;
