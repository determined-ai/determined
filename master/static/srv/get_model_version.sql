SELECT
    to_json(m) as model,
	  to_json(c) as checkpoint,
	  mv.version as version
	  FROM model_versions mv
	  JOIN models m ON mv.model_name = m.name
	  JOIN checkpoints c ON mv.checkpoint_uuid = c.uuid
	  WHERE mv.model_name = $1 AND mv.version = $2
