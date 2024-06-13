CREATE INDEX run_hparam_id ON run_hparams(run_id);
CREATE INDEX run_hparam_number_val ON run_hparams(hparam, number_val);
CREATE INDEX run_hparam_text_val ON run_hparams(hparam, text_val);
CREATE INDEX run_hparam_bool_val ON run_hparams(hparam, bool_val);
