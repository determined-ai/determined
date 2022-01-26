WITH m AS (
  SELECT id
  FROM models
  WHERE name = $1
  AND (user_id = $2 OR $3 IS TRUE)
),
mv AS (
  DELETE FROM model_versions
  WHERE model_id IN (SELECT id FROM m)
)
DELETE FROM models
WHERE models.id IN (SELECT id FROM m)
RETURNING models.id;
