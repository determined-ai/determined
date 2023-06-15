CREATE SEQUENCE public.task_sessions_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

CREATE TABLE public.task_sessions (
      id integer NOT NULL PRIMARY KEY DEFAULT nextval('public.task_sessions_id_seq'::regclass),
      task_id uuid NOT NULL UNIQUE
);
