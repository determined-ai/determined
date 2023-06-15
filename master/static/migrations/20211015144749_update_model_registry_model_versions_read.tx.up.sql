ALTER TABLE public.model_versions ADD COLUMN id INTEGER;
CREATE SEQUENCE public.model_versions_id_seq OWNED BY public.model_versions.id
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER TABLE public.model_versions ALTER COLUMN id SET DEFAULT nextval('public.model_versions_id_seq');
UPDATE public.model_versions SET id = nextval('public.model_versions_id_seq');
ALTER TABLE public.model_versions ALTER COLUMN id SET NOT NULL;
ALTER TABLE public.model_versions ADD COLUMN name text;
ALTER TABLE public.model_versions ADD COLUMN comment text;
ALTER TABLE public.model_versions ADD COLUMN readme text;
ALTER TABLE public.model_versions ADD COLUMN user_id integer;
/* What should the default value of user id be here? Should we do the same thing we did with models and make it correspond to the determined user? */
ALTER TABLE public.model_versions ADD CONSTRAINT users_fk FOREIGN KEY (user_id) REFERENCES public.users(id);
