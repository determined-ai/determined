DO $$
    DECLARE exec_text text;
    DECLARE trials_augmented_view text;
BEGIN
    trials_augmented_view  := pg_get_viewdef('trials_augmented_view');
    DROP VIEW trials_augmented_view;

    ALTER TYPE trial_state RENAME TO _trial_state;

    CREATE TYPE trial_state AS ENUM (
        'ACTIVE',
        'PAUSED',
        'STOPPING_CANCELED',
        'STOPPING_KILLED',
        'STOPPING_COMPLETED',
        'STOPPING_ERROR',
        'CANCELED',
        'COMPLETED',
        'ERROR',
        'RUNNING'
    );

    ALTER TABLE trials ALTER COLUMN state TYPE trial_state USING state::text::trial_state;
    DROP TYPE _trial_state;

    exec_text := format('CREATE VIEW trials_augmented_view AS %s', trials_augmented_view);
    EXECUTE exec_text;

    ALTER TYPE experiment_state RENAME TO _experiment_state;
    CREATE TYPE public.experiment_state AS ENUM (
        'ACTIVE',
        'PAUSED',
        'STOPPING_CANCELED',
        'STOPPING_KILLED',
        'STOPPING_COMPLETED',
        'STOPPING_ERROR',
        'DELETING',
        'CANCELED',
        'COMPLETED',
        'ERROR',
        'DELETE_FAILED',
        'RUNNING'
    );

    ALTER TABLE experiments ALTER COLUMN state TYPE experiment_state USING state::text::experiment_state;
    DROP TYPE _experiment_state;
END $$;
