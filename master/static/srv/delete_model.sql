WITH myuser AS (
  SELECT * FROM users WHERE id = $2 LIMIT 1
),
WITH m AS (
  SELECT id
  FROM models, myuser
  WHERE models.id = $1 AND (
    models.user_id = $2
    OR myuser.admin
  )
),
WITH mv AS (
  DELETE FROM model_versions
  WHERE model_id IN (SELECT id FROM m)
  RETURNING id
)
DELETE FROM models
WHERE id IN (SELECT id FROM m)
RETURNING id;
