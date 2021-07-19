ALTER TABLE public.trials
    DROP COLUMN restarts,
    DROP COLUMN trial_run_id;

DROP TABLE public.tasks;

DROP TYPE public.job_type;

CREATE TYPE public.run_type AS ENUM (
    'TRIAL'
);

CREATE TABLE public.runs (
    id integer NOT NULL,
    start_time timestamp without time zone NOT NULL DEFAULT now(),
    end_time timestamp without time zone NULL,
    run_type run_type NOT NULL,
    run_type_fk integer NOT NULL,
    CONSTRAINT trial_runs_id_trial_id_unique UNIQUE (run_type, run_type_fk, id)
);


CREATE TABLE public.trial_snapshots (
    id SERIAL,
    trial_id integer NOT NULL UNIQUE,
    request_id bytea NOT NULL,
    experiment_id integer NOT NULL,
    content jsonb NOT NULL,
    version integer NOT NULL,
    created_at timestamp with time zone NOT NULL DEFAULT NOW(),
    updated_at timestamp with time zone NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_trial_snapshots_trials_trial_id FOREIGN KEY(trial_id) REFERENCES public.trials(id),
    CONSTRAINT fk_trial_snapshots_experiments_experiment_id FOREIGN KEY(experiment_id) REFERENCES public.experiments(id),
    CONSTRAINT uq_trial_snapshots_experiment_id_request_id UNIQUE(experiment_id, request_id)
);
