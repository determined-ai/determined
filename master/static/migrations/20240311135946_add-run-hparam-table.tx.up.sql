CREATE TABLE run_hparams (
    run_id int REFERENCES runs(id),
    hparam text NOT NULL,
    number_val float NULL,
    text_val text NULL,
    date_val timestamp with time zone NULL
);
