WITH m AS (
  SELECT id
  FROM models
  WHERE name = $1
),
mv AS (
  DELETE FROM model_versions
  WHERE model_id IN (SELECT id FROM m)
)
DELETE FROM models
WHERE models.id IN (SELECT id FROM m)
RETURNING models.id;
