CREATE TABLE public.experiment_snapshots (
    id SERIAL,
    experiment_id INTEGER NOT NULL UNIQUE,
    content JSONB NOT NULL,
    version INTEGER NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_experiment_snapshots_experiments_experiment_id FOREIGN KEY (
        experiment_id
    ) REFERENCES public.experiments (id)
);

CREATE TABLE public.trial_snapshots (
    id SERIAL,
    trial_id INTEGER NOT NULL UNIQUE,
    request_id BYTEA NOT NULL,
    experiment_id INTEGER NOT NULL,
    content JSONB NOT NULL,
    version INTEGER NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_trial_snapshots_trials_trial_id FOREIGN KEY (
        trial_id
    ) REFERENCES public.trials (id),
    CONSTRAINT fk_trial_snapshots_experiments_experiment_id FOREIGN KEY (
        experiment_id
    ) REFERENCES public.experiments (id),
    CONSTRAINT uq_trial_snapshots_experiment_id_request_id UNIQUE (
        experiment_id, request_id
    )
);

ALTER TABLE public.trials ADD COLUMN request_id TEXT NULL;

DROP TABLE public.searcher_events;
