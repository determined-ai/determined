INSERT INTO model_versions
	(
		model_id,
		version,
		checkpoint_uuid,
		creation_time,
		last_updated_time
	)
VALUES (
		$1,
		(SELECT COALESCE(max(version), 0) + 1 FROM model_versions WHERE model_id = $1),
		$2,
		current_timestamp,
		current_timestamp
	)
RETURNING version, creation_time;
