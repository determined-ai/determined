DROP TABLE public.models CASCADE;
DROP TABLE public.model_versions CASCADE;

ALTER TABLE ONLY public.checkpoints
	ADD CONSTRAINT checkpoint_uuid_uniq UNIQUE (uuid);

CREATE TABLE public.models (
	name character varying UNIQUE NOT NULL,
	description character varying,
	creation_time timestamp with time zone NOT NULL,
	last_updated_time timestamp with time zone,
	metadata jsonb,

	CONSTRAINT models_pkey PRIMARY KEY (name)
);

CREATE TABLE public.model_versions (
	version integer NOT NULL,
	model_name character varying NOT NULL,
	checkpoint_uuid uuid NOT NULL,
	creation_time timestamp with time zone NOT NULL,
	last_updated_time timestamp with time zone,
	metadata jsonb,

	CONSTRAINT model_and_version_unique UNIQUE (model_name, version),
	CONSTRAINT model_versions_pkey PRIMARY KEY (model_name, version),
	FOREIGN KEY(model_name) REFERENCES public.models(name),
	FOREIGN KEY(checkpoint_uuid) REFERENCES public.checkpoints(uuid)
);
