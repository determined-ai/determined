WITH myuser AS (
  SELECT * FROM users WHERE id = $2 LIMIT 1
),
mv AS (
  SELECT model_versions.id
  FROM model_versions
  JOIN models ON models.id = model_versions.model_id
  WHERE model_versions.id = $1
    AND (
      model_versions.user_id = $2
      OR
      models.user_id = $2
    )
)
DELETE FROM model_versions
USING myuser
WHERE
  model_versions.id = $1 AND (
    model_versions.id IN (SELECT id FROM mv)
    OR myuser.admin
  )
RETURNING model_versions.id;
