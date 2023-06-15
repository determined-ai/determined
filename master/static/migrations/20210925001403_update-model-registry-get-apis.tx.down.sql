ALTER TABLE public.models DROP CONSTRAINT models_pkey CASCADE;
ALTER TABLE public.models ADD PRIMARY KEY (name);
ALTER TABLE public.model_versions ADD COLUMN model_name character varying REFERENCES public.models(name);
WITH id_name_map as (
    SELECT m.name, m.id
    FROM public.models as m
) UPDATE public.model_versions mv SET model_name = id_name_map.name FROM id_name_map WHERE mv.model_id = id_name_map.id;
ALTER TABLE public.model_versions ALTER COLUMN model_name SET NOT NULL;
ALTER TABLE public.model_versions DROP CONSTRAINT model_versions_pkey CASCADE;
ALTER TABLE public.model_versions ADD PRIMARY KEY (model_name, version);
ALTER TABLE public.model_versions DROP COLUMN model_id;

ALTER TABLE public.models DROP COLUMN id CASCADE;
ALTER TABLE public.models DROP COLUMN labels,
DROP COLUMN readme,
DROP COLUMN user_id,
DROP COLUMN archived;
