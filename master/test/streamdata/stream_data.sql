-- objects to insert for test
INSERT INTO workspaces (name) VALUES ('test_workspace1');

INSERT INTO projects (name, workspace_id) VALUES ('test_project1', 2);

INSERT INTO jobs (job_id, job_type, owner_id) VALUES ('test_job1', 'EXPERIMENT', 1);
INSERT INTO jobs (job_id, job_type, owner_id) VALUES ('test_job2', 'EXPERIMENT', 1);

INSERT INTO experiments (state, config, model_definition, start_time, owner_id, notes, job_id, project_id)
VALUES ('ERROR', '{}', '', '2023-07-25 16:44:21.610081+00', 1, '', 'test_job1', 2);
INSERT INTO experiments (state, config, model_definition, start_time, owner_id, notes, job_id, project_id)
VALUES ('ERROR', '{}', '', '2023-07-25 16:44:21.610081+00', 1, '', 'test_job2', 2);

INSERT INTO tasks (task_id, task_type, start_time, job_id) VALUES ('1.1', 'TRIAL', '2023-07-25 16:44:21.610081+00', 'test_job1');
INSERT INTO tasks (task_id, task_type, start_time, job_id) VALUES ('1.2', 'TRIAL', '2023-07-25 16:44:21.610081+00', 'test_job1');
INSERT INTO tasks (task_id, task_type, start_time, job_id) VALUES ('1.3', 'TRIAL', '2023-07-25 16:44:21.610081+00', 'test_job1');

INSERT INTO trials (id, experiment_id, state, start_time, hparams, seq) VALUES (1, 1, 'ERROR', '2023-07-25 16:44:21.610081+00', '{}', 1);
INSERT INTO trials (id, experiment_id, state, start_time, hparams, seq) VALUES (2, 1, 'ERROR', '2023-07-25 16:44:22.610081+00', '{}', 2);
INSERT INTO trials (id, experiment_id, state, start_time, hparams, seq) VALUES (3, 1, 'ERROR', '2023-07-25 16:44:23.610081+00', '{}', 3);

INSERT INTO trial_id_task_id (trial_id, task_id) VALUES (1, '1.1');
INSERT INTO trial_id_task_id (trial_id, task_id) VALUES (2, '1.2');
INSERT INTO trial_id_task_id (trial_id, task_id) VALUES (3, '1.3');

INSERT INTO checkpoints_v2 (uuid, task_id, report_time, state) VALUES ('ae4fb7ae-887f-41fa-a70b-97c55c9b18d2', '1.1', '2023-07-25 16:44:23.610081+00', 'COMPLETED');
INSERT INTO checkpoints_v2 (uuid, task_id, report_time, state) VALUES ('ae4fb7ae-887f-41fa-a70b-97c55c9b18d3', '1.2', '2023-07-25 16:44:23.610081+00', 'COMPLETED');
