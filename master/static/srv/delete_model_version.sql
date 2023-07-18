WITH mv AS (
  SELECT model_versions.id
  FROM model_versions
  JOIN models ON models.id = model_versions.model_id
  WHERE model_versions.id = $1
)
DELETE FROM model_versions
WHERE model_versions.id IN (SELECT id FROM mv)
RETURNING model_versions.id;
