WITH pe AS (
  SELECT
    MAX(start_time) AS last_experiment_started_at,
    COUNT(*) AS num_experiments,
    SUM(case when state = 'ACTIVE' then 1 else 0 end) AS num_active_experiments
  FROM experiments
  WHERE project_id = $1
),
p AS (
  UPDATE projects SET name = $2, description = $3
  WHERE projects.id = $1
  RETURNING projects.*
),
u AS (
  SELECT username FROM users, p
  WHERE users.id = p.user_id
)
SELECT p.id, p.name, 'WORKSPACE_STATE_' || p.state AS state, p.error_message, p.workspace_id, p.description, p.archived, p.immutable, p.notes,
pe.last_experiment_started_at, pe.num_experiments, pe.num_active_experiments,
u.username, p.user_id
FROM p, pe, u;
