WITH mm AS (
  UPDATE maintenance_messages
  SET end_time = NOW()
  WHERE (id = $1 OR $1 = 0)
  AND start_time <= NOW()
  AND (end_time < '1900-01-01' OR end_time >= NOW())
  RETURNING id
)
SELECT id FROM mm;
