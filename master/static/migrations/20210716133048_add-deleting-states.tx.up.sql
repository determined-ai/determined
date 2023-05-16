ALTER TYPE public.experiment_state RENAME TO _experiment_state;
CREATE TYPE public.experiment_state AS ENUM (
    'ACTIVE',
    'CANCELED',
    'COMPLETED',
    'ERROR',
    'PAUSED',
    'STOPPING_CANCELED',
    'STOPPING_COMPLETED',
    'STOPPING_ERROR',
    'DELETING',
    'DELETE_FAILED'
);
ALTER TABLE public.experiments ALTER COLUMN state TYPE experiment_state USING state::text::experiment_state;
DROP TYPE _experiment_state;
