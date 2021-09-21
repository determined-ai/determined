BEGIN
  INSERT INTO jobs (job_id, job_type)
  VALUES ('123', 'EXPERIMENT');
  INSERT INTO experiments (state, job_id, config, model_definition, start_time)
  VALUES ('PAUSED', '123', '{}', '', '');
COMMIT
