WITH p AS (
  SELECT id FROM projects
  WHERE workspace_id = $1
),
exp_count AS (
  SELECT COUNT(*) AS count FROM experiments
  WHERE project_id IN (SELECT id FROM p)
)
SELECT w.id, w.name, w.archived, w.immutable, u.username, w.user_id,
  'WORKSPACE_STATE_' || w.state AS state, w.error_message,
  (SELECT COUNT(*) FROM p) AS num_projects,
  (SELECT count FROM exp_count) AS num_experiments,
  (SELECT COUNT(*) > 0 FROM workspace_pins
    WHERE workspace_id = $1 AND user_id = $2
  ) AS pinned
FROM workspaces as w
  LEFT JOIN users as u ON u.id = w.user_id
WHERE w.id = $1;
