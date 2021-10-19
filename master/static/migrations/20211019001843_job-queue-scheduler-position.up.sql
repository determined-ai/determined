/*
add queue position to jobs table
*/

ALTER TABLE public.jobs
    ADD COLUMN queue_position text NOT NULL DEFAULT '-1';

