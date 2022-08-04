WITH proj AS (
  SELECT id FROM projects
  WHERE id = $1
  AND NOT immutable
),
exper AS (
  SELECT COUNT(*) AS count
  FROM experiments
  WHERE project_id IN (SELECT id FROM proj)
)
DELETE FROM projects
WHERE id IN (SELECT id FROM proj)
AND (SELECT count FROM exper) = 0
RETURNING projects.id;
