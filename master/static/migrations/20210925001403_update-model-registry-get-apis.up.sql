ALTER TABLE public.models ADD COLUMN id integer;
CREATE SEQUENCE public.models_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

UPDATE public.models SET id = nextval('public.models_id_seq');
ALTER TABLE public.models ALTER COLUMN id SET NOT NULL;
ALTER TABLE public.models ALTER COLUMN id SET DEFAULT nextval('public.models_id_seq');
ALTER TABLE public.models DROP CONSTRAINT models_pkey CASCADE;
ALTER TABLE public.models ADD PRIMARY KEY (id);
ALTER TABLE public.models ADD COLUMN labels text[];
ALTER TABLE public.models ADD COLUMN readme text;
ALTER TABLE public.models ADD COLUMN user_id integer;
WITH det_id as (SELECT id from public.users where username LIKE 'determined') UPDATE public.models SET user_id = det_id.id FROM det_id WHERE user_id is NULL;
/* ALTER TABLE public.models ALTER COLUMN user_id SET NOT NULL; */
ALTER TABLE public.models ADD CONSTRAINT users_fk FOREIGN KEY (user_id) REFERENCES public.users(id);
ALTER TABLE public.models ADD COLUMN archived boolean DEFAULT false NOT NULL;

ALTER TABLE public.model_versions ADD COLUMN model_id INTEGER REFERENCES public.models(id);
WITH id_name_map as (
    SELECT m.id, m.name
    FROM public.models as m
) UPDATE public.model_versions mv SET model_id = id_name_map.id FROM id_name_map WHERE mv.model_name = id_name_map.name;
ALTER TABLE public.model_versions ALTER COLUMN model_id SET NOT NULL;
ALTER TABLE public.model_versions DROP CONSTRAINT model_versions_pkey CASCADE;
ALTER TABLE public.model_versions ADD PRIMARY KEY (model_id, version);
ALTER TABLE public.model_versions DROP COLUMN model_name;
