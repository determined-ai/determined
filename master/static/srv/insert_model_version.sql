INSERT INTO model_versions (model_name, version, checkpoint_uuid, creation_time, last_updated_time)
VALUES (
	(SELECT CAST($1 AS character varying)),
	(SELECT COALESCE(max(version), 0) + 1 FROM model_versions WHERE model_name = $1),
	$2, $3, $4)
RETURNING version;
