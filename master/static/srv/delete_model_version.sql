DELETE FROM model_versions
WHERE id = $1
RETURNING id;
