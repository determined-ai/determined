SELECT
    c.uuid AS uuid,
		e.config AS experiment_config,
		e.id AS  experiment_id,
		t.hparams as hparams,
    s.id * (e.config->>'batches_per_step')::int AS batch_number,
    s.start_time AS start_time,
    s.end_time AS end_time,
    c.metadata AS metadata,
    c.resources AS resources,
    c.framework as framework,
    c.format as format,
    c.determined_version as determined_version,
    v.metrics AS metrics,
    v.state AS validation_state
FROM checkpoints c
JOIN steps s ON c.step_id = s.id AND c.trial_id = s.trial_id
LEFT JOIN validations v ON v.step_id = s.id AND v.trial_id = s.trial_id
JOIN trials t ON s.trial_id = t.id
JOIN experiments e ON t.experiment_id = e.id
WHERE c.uuid = $1
