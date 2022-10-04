WITH exp AS (
  UPDATE experiments e
  SET group_id = NULL
  WHERE e.group_id = $1
)
DELETE FROM experiment_groups
  WHERE id = $1
RETURNING experiment_groups.id;
