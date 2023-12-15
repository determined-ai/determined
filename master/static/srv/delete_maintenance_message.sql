WITH mm AS (
  DELETE FROM maintenance_messages
  WHERE id = $1 OR $1 = 0
  RETURNING id
)
SELECT id FROM mm;
