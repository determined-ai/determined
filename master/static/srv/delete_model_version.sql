WITH myuser AS (
  SELECT * FROM users WHERE id = $2 LIMIT 1
),
WITH mv AS (
  SELECT id
  FROM model_versions, myuser
  JOIN models ON models.id = model_versions.model_id
  WHERE model_versions.id = $1 AND (
    models.user_id = $2
    OR myuser.admin
  )
)
DELETE FROM model_versions
WHERE id IN (SELECT id FROM mv)
RETURNING id;
