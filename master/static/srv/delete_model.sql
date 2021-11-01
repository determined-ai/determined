WITH myuser AS (
  SELECT *
  FROM users
  WHERE id = $2
  LIMIT 1
),
m AS (
  SELECT id
  FROM models
  WHERE id = $1 AND user_id = $2
),
mv AS (
  DELETE FROM model_versions
  WHERE model_id IN (SELECT id FROM m)
)
DELETE FROM models
USING myuser
WHERE
  models.id = $1 AND (
    models.id IN (SELECT id FROM m)
    OR myuser.admin
  )
RETURNING models.id;
