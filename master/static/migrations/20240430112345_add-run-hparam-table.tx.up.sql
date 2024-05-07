CREATE TABLE run_hparams (
    run_id int REFERENCES runs(id) ON DELETE CASCADE,
    hparam text NOT NULL,
    number_val float NULL,
    text_val text NULL,
    bool_val boolean NULL
);

INSERT INTO run_hparams(run_id, hparam, number_val, text_val, bool_val)
SELECT s.id as run_id, s.key as hparam,
CASE WHEN s.type='number' THEN s.value::text::float ELSE NULL END as number_val,
CASE WHEN s.type='string' THEN s.value::text ELSE NULL END as text_val,
CASE WHEN s.type='boolean' THEN s.value::text::boolean ELSE NULL END as bool_val
FROM (SELECT r.id, h.key, h.value, jsonb_typeof(h.value) as type FROM runs as r, jsonb_each(r.hparams) as h WHERE r.hparams is not NULL AND r.hparams !='null') as s WHERE s.type!='object';

INSERT INTO run_hparams(run_id, hparam, number_val, text_val, bool_val)
SELECT n.id as run_id, n.key as hparam,
CASE WHEN n.type='number' THEN n.value::text::float ELSE NULL END as number_val,
CASE WHEN n.type='string' THEN n.value::text ELSE NULL END as text_val,
CASE WHEN n.type='boolean' THEN n.value::text::boolean ELSE NULL END as bool_val
FROM (SELECT s.id, CONCAT(s.key, '.', nh.key) as key, nh.value,  jsonb_typeof(nh.value) as type FROM (SELECT r.id, h.key, h.value, jsonb_typeof(h.value) as type FROM runs as r, jsonb_each(r.hparams) as h WHERE r.hparams is not NULL AND r.hparams !='null') as s, jsonb_each(s.value) nh WHERE s.type='object') as n;


CREATE TABLE project_hparams (
    project_id int REFERENCES projects(id) ON DELETE CASCADE,
    hparam text NOT NULL,
    type text NOT NULL,
    UNIQUE (project_id, hparam, type)
);

INSERT INTO project_hparams(project_id, hparam, type)
SELECT * FROM
(SELECT r.project_id, h.key, jsonb_typeof(h.value) as type
FROM runs as r, jsonb_each(r.hparams) as h WHERE r.hparams is not NULL AND r.hparams !='null'
GROUP BY project_id, key, type) as s WHERE s.type!='object';

INSERT INTO project_hparams(project_id, hparam, type)
SELECT * FROM
(SELECT s.project_id, CONCAT(s.key, '.', nh.key) as key,  jsonb_typeof(nh.value) as type
FROM (SELECT r.project_id, r.id, h.key, h.value, jsonb_typeof(h.value) as type FROM
runs as r, jsonb_each(r.hparams) as h WHERE r.hparams is not NULL AND r.hparams !='null') as s,
jsonb_each(s.value) nh WHERE s.type='object') as n GROUP BY project_id, key, type;
