BEGIN;
  INSERT INTO jobs (job_id, job_type)
  VALUES ('golabi', 'EXPERIMENT');
  INSERT INTO experiments (state, job_id, config, model_definition, start_time, owner_id)
  VALUES ('PAUSED', 'golabi', '{}', '', '2021-09-21 19:42:13.632166+00', 1);
COMMIT;
