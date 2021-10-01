ALTER TABLE public.tasks
    ADD COLUMN log_version smallint DEFAULT 0;

CREATE TABLE public.task_logs (
    id BIGSERIAL,
    task_id text NOT NULL,
    log bytea NOT NULL,
    allocation_id text NULL,
    agent_id text NULL,
     -- In the case of k8s, this is a pod name.
    container_id text NULL,
    rank_id smallint NULL,
    timestamp timestamp NULL,
    level text NULL,
    source text NULL,
    stdtype text NULL
);

CREATE INDEX ix_task_logs_task_id ON public.task_logs USING btree (task_id);
