WITH mv AS (
  SELECT model_versions.id
  FROM model_versions
  JOIN models ON models.id = model_versions.model_id
  WHERE model_versions.id = $1
    AND (
      model_versions.user_id = $2
      OR
      models.user_id = $2
      OR
      $3 IS TRUE
    )
)
DELETE FROM model_versions
WHERE model_versions.id IN (SELECT id FROM mv)
RETURNING model_versions.id;
