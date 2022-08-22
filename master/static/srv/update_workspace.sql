WITH w AS (
  UPDATE workspaces SET name = $2
  WHERE workspaces.id = $1
  RETURNING workspaces.*
),
u AS (
  SELECT username FROM users, w
  WHERE users.id = w.user_id
),
p AS (
  SELECT COUNT(*) AS num_projects
  FROM projects
  WHERE workspace_id = $1
)
SELECT w.id, w.name, 'WORKSPACE_STATE_' || w.state AS state, w.error_message, w.archived, w.immutable,
  u.username, w.user_id, p.num_projects,
  (SELECT COUNT(*) FROM experiments WHERE project_id IN (SELECT id FROM p))
    AS num_experiments,
  (SELECT COUNT(*) > 0 FROM workspace_pins
    WHERE workspace_id = $1 AND user_id = $3
  ) AS pinned
FROM w, u, p;
