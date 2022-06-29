ALTER TABLE resourcemanagers_dispatcher_dispatches
    -- Used to cancel the job, since it must be the original user that cancels it.
    ADD COLUMN impersonated_user text NOT NULL;
