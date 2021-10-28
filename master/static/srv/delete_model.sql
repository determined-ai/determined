WITH m AS (
  SELECT * FROM models
  WHERE id = $1 AND user_id = $2
),
v AS (
  DELETE FROM model_versions
  WHERE (
    model_versions.model_id IN (SELECT id FROM m)
  )
)
DELETE FROM models
WHERE id IN (SELECT id FROM m)
RETURNING id;
