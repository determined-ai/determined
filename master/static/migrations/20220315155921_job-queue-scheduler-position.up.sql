/*
add queue position to jobs table
*/

ALTER TABLE public.jobs
    ADD COLUMN q_position text NOT NULL DEFAULT '-1';

