CREATE TABLE run_hparams (
    run_id int REFERENCES runs(id) ON DELETE CASCADE,
    hparam text NOT NULL,
    number_val float NULL,
    text_val text NULL,
    bool_val boolean NULL
);

WITH RECURSIVE flat (run_id, key, value, type) AS (
    SELECT r.id, h.key, h.value, jsonb_typeof(h.value) as type
    FROM runs as r, jsonb_each(r.hparams) as h WHERE r.hparams is not NULL AND r.hparams !='null' 
	UNION
	SELECT f.run_id,concat(f.key, '.', j.key) as key, j.value, jsonb_typeof(j.value) as type
    FROM flat f, jsonb_each(f.value) j WHERE f.type = 'object'
	)

INSERT INTO run_hparams(run_id, hparam, number_val, text_val, bool_val)
SELECT run_id, key, 
CASE WHEN type='number' THEN value::text::float ELSE NULL END as number_val,
CASE WHEN type='string' THEN value::text ELSE NULL END as text_val,
CASE WHEN type='boolean' THEN value::text::boolean ELSE NULL END as bool_val
FROM flat WHERE type !='object';

CREATE TABLE project_hparams (
    project_id int REFERENCES projects(id) ON DELETE CASCADE,
    hparam text NOT NULL,
    type text NOT NULL,
    UNIQUE (project_id, hparam, type)
);

WITH RECURSIVE flat (run_id, key, value, type) AS (
    SELECT r.id, h.key, h.value, jsonb_typeof(h.value) as type
    FROM runs as r, jsonb_each(r.hparams) as h WHERE r.hparams is not NULL AND r.hparams !='null' 
	UNION
	SELECT f.run_id,concat(f.key, '.', j.key) as key, j.value, jsonb_typeof(j.value) as type
    FROM flat f, jsonb_each(f.value) j WHERE f.type = 'object'
	)
INSERT INTO project_hparams(project_id, hparam, type)
SELECT r.project_id, f.key, type
FROM flat f JOIN runs r ON f.run_id=r.id
WHERE type != 'object' GROUP BY project_id, key, type;
