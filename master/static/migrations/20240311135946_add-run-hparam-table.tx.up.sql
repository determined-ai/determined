CREATE TABLE run_hparams (
    run_id int REFERENCES runs(id) ON DELETE CASCADE,
    hparam text NOT NULL,
    number_val float NULL,
    text_val text NULL,
    date_val timestamp with time zone NULL,
    bool_val boolean NULL
);

CREATE INDEX ix_run_hparams_num ON run_hparams(hparam, number_val);
CREATE INDEX ix_run_hparams_text ON run_hparams(hparam, text_val);
CREATE INDEX ix_run_hparams_date ON run_hparams(hparam, date_val);
CREATE INDEX ix_run_hparams_bool ON run_hparams(hparam, bool_val);
