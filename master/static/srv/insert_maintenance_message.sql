WITH mm AS (
  INSERT INTO maintenance_messages
  (user_id, message, start_time, end_time)
  VALUES ($1, $2, $3, $4)
  RETURNING id
)
SELECT id FROM mm;
