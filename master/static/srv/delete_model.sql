WITH v AS (
  DELETE FROM model_versions
  USING models
  WHERE (
    model_versions.id = $1
    AND models.user_id = $2
    AND models.id = model_versions.model_id
  )
)
DELETE FROM models
WHERE id = $1 AND user_id = $2
RETURNING id;
