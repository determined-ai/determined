WITH t AS (
  INSERT INTO webhook_triggers (trigger_type, condition, webhook_id)
  VALUES ($1, $2, $3)
  RETURNING id
)
SELECT t.id
FROM t

