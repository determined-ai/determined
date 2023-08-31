INSERT INTO jobs (job_id, job_type, owner_id) VALUES ('test_job', 'EXPERIMENT', 1);

INSERT INTO experiments (state, config, model_definition, start_time, owner_id, notes, job_id)
    VALUES ('ERROR', '{}', '', '2023-07-25 16:44:21.610081+00', 1, '', 'test_job');

INSERT INTO tasks (task_id, task_type, start_time, job_id) VALUES ('1.1', 'TRIAL', '2023-07-25 16:44:21.610081+00', 'test_job');
INSERT INTO tasks (task_id, task_type, start_time, job_id) VALUES ('1.2', 'TRIAL', '2023-07-25 16:44:21.610081+00', 'test_job');
INSERT INTO tasks (task_id, task_type, start_time, job_id) VALUES ('1.3', 'TRIAL', '2023-07-25 16:44:21.610081+00', 'test_job');

INSERT INTO trials (id, experiment_id, state, start_time, hparams, task_id, seq) VALUES (1, 1, 'ERROR', '2023-07-25 16:44:21.610081+00', '{}', '1.1', 1);
INSERT INTO trials (id, experiment_id, state, start_time, hparams, task_id, seq) VALUES (2, 1, 'ERROR', '2023-07-25 16:44:21.610081+00', '{}', '1.2', 2);
INSERT INTO trials (id, experiment_id, state, start_time, hparams, task_id, seq) VALUES (3, 1, 'ERROR', '2023-07-25 16:44:21.610081+00', '{}', '1.3', 3);
