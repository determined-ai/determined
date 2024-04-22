CREATE TABLE run_hparams (
    run_id int REFERENCES runs(id) ON DELETE CASCADE,
    hparam text NOT NULL,
    number_val float NULL,
    text_val text NULL,
    bool_val boolean NULL
);

CREATE INDEX ix_run_hparams_num ON run_hparams(hparam, number_val);
CREATE INDEX ix_run_hparams_text ON run_hparams(hparam, text_val);
CREATE INDEX ix_run_hparams_bool ON run_hparams(hparam, bool_val);

INSERT INTO run_hparams(run_id, hparam, number_val, text_val, bool_val)
SELECT s.id as run_id, s.key as hparam,
CASE WHEN s.type='number' THEN s.value::text::float ELSE NULL END as number_val,
CASE WHEN s.type='string' THEN s.value::text ELSE NULL END as text_val,
CASE WHEN s.type='boolean' THEN s.value::text::boolean ELSE NULL END as bool_val
FROM (SELECT r.id, h.key, h.value, jsonb_typeof(h.value) as type FROM runs as r, jsonb_each(r.hparams) as h WHERE r.hparams is not NULL AND r.hparams !='null') as s WHERE s.type!='object'
