/*
add queue position to jobs table
*/

ALTER TABLE public.jobs
    ADD COLUMN q_position float NOT NULL DEFAULT '-1';

