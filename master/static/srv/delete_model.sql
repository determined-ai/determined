WITH v AS (
  DELETE FROM model_versions
  WHERE model_id = $1
),
DELETE FROM models
WHERE id = $1
RETURNING id;
