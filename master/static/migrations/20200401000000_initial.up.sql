--
-- PostgreSQL database dump
--

-- Dumped from database version 10.12 (Debian 10.12-2.pgdg90+1)
-- Dumped by pg_dump version 10.8 (Debian 10.8-1.pgdg90+1)

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- Name: checkpoint_state; Type: TYPE; Schema: public; Owner: -
--

CREATE TYPE public.checkpoint_state AS ENUM (
    'ACTIVE',
    'COMPLETED',
    'ERROR',
    'DELETED'
);


--
-- Name: experiment_state; Type: TYPE; Schema: public; Owner: -
--

CREATE TYPE public.experiment_state AS ENUM (
    'ACTIVE',
    'CANCELED',
    'COMPLETED',
    'ERROR',
    'PAUSED',
    'STOPPING_CANCELED',
    'STOPPING_COMPLETED',
    'STOPPING_ERROR'
);


--
-- Name: step_state; Type: TYPE; Schema: public; Owner: -
--

CREATE TYPE public.step_state AS ENUM (
    'ACTIVE',
    'COMPLETED',
    'ERROR'
);


--
-- Name: trial_state; Type: TYPE; Schema: public; Owner: -
--

CREATE TYPE public.trial_state AS ENUM (
    'ACTIVE',
    'CANCELED',
    'COMPLETED',
    'ERROR'
);


--
-- Name: validation_state; Type: TYPE; Schema: public; Owner: -
--

CREATE TYPE public.validation_state AS ENUM (
    'ACTIVE',
    'COMPLETED',
    'ERROR'
);


SET default_tablespace = '';

SET default_with_oids = false;

--
-- Name: checkpoints; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.checkpoints (
    id integer NOT NULL,
    trial_id integer NOT NULL,
    step_id integer NOT NULL,
    state public.checkpoint_state NOT NULL,
    start_time timestamp with time zone NOT NULL,
    end_time timestamp with time zone,
    uuid uuid,
    resources jsonb,
    labels jsonb
);


--
-- Name: best_checkpoint_by_metric(integer, text, boolean); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.best_checkpoint_by_metric(tid integer, metric text, smaller_is_better boolean) RETURNS SETOF public.checkpoints
    LANGUAGE sql STABLE
    AS $$
    SELECT c.*
    FROM checkpoints c JOIN validations v ON (c.trial_id, c.step_id) = (v.trial_id, v.step_id)
    WHERE c.trial_id = tid AND c.state = 'COMPLETED' AND v.state = 'COMPLETED'
    ORDER BY (SELECT CASE WHEN smaller_is_better THEN 1 ELSE -1 END) * (v.metrics->'validation_metrics'->>metric)::float8 ASC
    LIMIT 1
$$;


--
-- Name: experiments; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.experiments (
    id integer NOT NULL,
    state public.experiment_state NOT NULL,
    config jsonb NOT NULL,
    model_definition bytea NOT NULL,
    start_time timestamp with time zone NOT NULL,
    end_time timestamp with time zone,
    model_packages bytea,
    archived boolean DEFAULT false NOT NULL,
    git_remote character varying,
    git_commit character varying,
    git_committer character varying,
    git_commit_date timestamp without time zone,
    parent_id integer,
    owner_id integer NOT NULL,
    progress double precision
);


--
-- Name: validations; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.validations (
    id integer NOT NULL,
    trial_id integer NOT NULL,
    step_id integer NOT NULL,
    state public.validation_state NOT NULL,
    start_time timestamp with time zone NOT NULL,
    end_time timestamp with time zone,
    metrics jsonb
);


--
-- Name: experiments_best_validation_history(public.experiments); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.experiments_best_validation_history(e public.experiments) RETURNS SETOF public.validations
    LANGUAGE sql STABLE
    AS $$
    WITH is_best AS (
        SELECT v.id,
               coalesce(
                   get_signed_metric(v, e) < min(get_signed_metric(v, e))
                   OVER (
                       PARTITION BY e.id
                       ORDER BY v.end_time ASC
                       ROWS BETWEEN UNBOUNDED PRECEDING AND 1 PRECEDING
                   ),
                   true
               ) AS is_best
        FROM trials t, validations v
        WHERE e.id = t.experiment_id AND t.id = v.trial_id AND v.state = 'COMPLETED'
    )
    SELECT v.* FROM validations v, is_best WHERE v.id = is_best.id AND is_best.is_best
$$;


--
-- Name: frequencies(anyarray); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.frequencies(vals anyarray) RETURNS jsonb
    LANGUAGE sql IMMUTABLE
    AS $$
    SELECT coalesce(jsonb_agg(row_to_json(counts)), '[]'::jsonb)
    FROM (
        SELECT to_jsonb(unnest) as value, count(*)
        FROM unnest(vals) GROUP BY unnest
    ) counts
$$;


--
-- Name: get_raw_metric(public.validations, public.experiments); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.get_raw_metric(v public.validations, e public.experiments) RETURNS double precision
    LANGUAGE sql STABLE
    AS $$
    SELECT (v.metrics->'validation_metrics'->>(e.config->'searcher'->>'metric'))::float8
$$;


--
-- Name: get_signed_metric(public.validations, public.experiments); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.get_signed_metric(v public.validations, e public.experiments) RETURNS double precision
    LANGUAGE sql STABLE
    AS $$
    SELECT get_raw_metric(v, e) * (
        SELECT
        CASE
            WHEN coalesce((e.config->'searcher'->>'smaller_is_better')::boolean, true)
            THEN 1
            ELSE -1
        END)
$$;


--
-- Name: try_float8_cast(text); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.try_float8_cast(text) RETURNS double precision
    LANGUAGE sql IMMUTABLE STRICT
    AS $_$
            SELECT
                CASE
                    WHEN $1 ~ e'^-?(?:0|[1-9]\\d*)'
                               '(?:\\.\\d+)?(?:[eE][+-]?\\d+)?$' THEN
                        $1::float8
                END;
        $_$;


--
-- Name: agent_user_groups; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.agent_user_groups (
    id integer NOT NULL,
    user_id integer NOT NULL,
    user_ text NOT NULL,
    uid integer NOT NULL,
    group_ text NOT NULL,
    gid integer NOT NULL
);


--
-- Name: agent_user_groups_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.agent_user_groups_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: agent_user_groups_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.agent_user_groups_id_seq OWNED BY public.agent_user_groups.id;


--
-- Name: auth_token_keypair; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.auth_token_keypair (
    public_key bytea NOT NULL,
    private_key bytea NOT NULL
);


--
-- Name: checkpoints_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

ALTER TABLE public.checkpoints ALTER COLUMN id ADD GENERATED BY DEFAULT AS IDENTITY (
    SEQUENCE NAME public.checkpoints_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: cluster_id; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.cluster_id (
    cluster_id text NOT NULL
);


--
-- Name: config_files; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.config_files (
    id integer NOT NULL,
    content bytea
);


--
-- Name: config_files_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.config_files_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: config_files_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.config_files_id_seq OWNED BY public.config_files.id;


--
-- Name: experiments_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

ALTER TABLE public.experiments ALTER COLUMN id ADD GENERATED BY DEFAULT AS IDENTITY (
    SEQUENCE NAME public.experiments_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: searcher_events; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.searcher_events (
    id integer NOT NULL,
    experiment_id integer NOT NULL,
    event_type character varying NOT NULL,
    content jsonb NOT NULL
);


--
-- Name: searcher_events_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

ALTER TABLE public.searcher_events ALTER COLUMN id ADD GENERATED BY DEFAULT AS IDENTITY (
    SEQUENCE NAME public.searcher_events_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: steps; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.steps (
    trial_id integer NOT NULL,
    id integer NOT NULL,
    state public.step_state NOT NULL,
    start_time timestamp with time zone NOT NULL,
    end_time timestamp with time zone,
    metrics jsonb
);


--
-- Name: templates; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.templates (
    name character varying NOT NULL,
    config jsonb NOT NULL
);


--
-- Name: trial_logs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.trial_logs (
    id int8 NOT NULL,
    trial_id integer NOT NULL,
    message bytea NOT NULL
);


--
-- Name: trial_logs_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

ALTER TABLE public.trial_logs ALTER COLUMN id ADD GENERATED BY DEFAULT AS IDENTITY (
    SEQUENCE NAME public.trial_logs_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: trials; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.trials (
    id integer NOT NULL,
    experiment_id integer NOT NULL,
    state public.trial_state NOT NULL,
    start_time timestamp with time zone NOT NULL,
    end_time timestamp with time zone,
    hparams jsonb NOT NULL,
    warm_start_checkpoint_id integer,
    seed integer DEFAULT 0 NOT NULL
);


--
-- Name: trials_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

ALTER TABLE public.trials ALTER COLUMN id ADD GENERATED BY DEFAULT AS IDENTITY (
    SEQUENCE NAME public.trials_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: user_sessions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_sessions (
    id integer NOT NULL,
    user_id integer NOT NULL,
    expiry timestamp without time zone NOT NULL
);


--
-- Name: user_sessions_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.user_sessions_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: user_sessions_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.user_sessions_id_seq OWNED BY public.user_sessions.id;


--
-- Name: users; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.users (
    id integer NOT NULL,
    username text NOT NULL,
    password_hash text,
    admin boolean DEFAULT false,
    active boolean DEFAULT false
);


--
-- Name: users_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.users_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: users_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.users_id_seq OWNED BY public.users.id;


--
-- Name: validation_metrics; Type: VIEW; Schema: public; Owner: -
--

CREATE VIEW public.validation_metrics AS
 SELECT v.id,
    public.get_raw_metric(v.*, e.*) AS raw,
    public.get_signed_metric(v.*, e.*) AS signed
   FROM public.experiments e,
    public.trials t,
    public.validations v
  WHERE ((e.id = t.experiment_id) AND (t.id = v.trial_id));


--
-- Name: validations_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

ALTER TABLE public.validations ALTER COLUMN id ADD GENERATED BY DEFAULT AS IDENTITY (
    SEQUENCE NAME public.validations_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: agent_user_groups id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.agent_user_groups ALTER COLUMN id SET DEFAULT nextval('public.agent_user_groups_id_seq'::regclass);


--
-- Name: config_files id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.config_files ALTER COLUMN id SET DEFAULT nextval('public.config_files_id_seq'::regclass);


--
-- Name: user_sessions id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_sessions ALTER COLUMN id SET DEFAULT nextval('public.user_sessions_id_seq'::regclass);


--
-- Name: users id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users ALTER COLUMN id SET DEFAULT nextval('public.users_id_seq'::regclass);


--
-- Name: agent_user_groups agent_user_groups_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.agent_user_groups
    ADD CONSTRAINT agent_user_groups_pkey PRIMARY KEY (id);


--
-- Name: agent_user_groups agent_user_groups_user_id_unique; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.agent_user_groups
    ADD CONSTRAINT agent_user_groups_user_id_unique UNIQUE (user_id);


--
-- Name: checkpoints checkpoints_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.checkpoints
    ADD CONSTRAINT checkpoints_pkey PRIMARY KEY (id);


--
-- Name: checkpoints checkpoints_trial_step_unique; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.checkpoints
    ADD CONSTRAINT checkpoints_trial_step_unique UNIQUE (trial_id, step_id);


--
-- Name: config_files config_files_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.config_files
    ADD CONSTRAINT config_files_pkey PRIMARY KEY (id);


--
-- Name: experiments experiments_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.experiments
    ADD CONSTRAINT experiments_pkey PRIMARY KEY (id);


--
-- Name: searcher_events searcher_events_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.searcher_events
    ADD CONSTRAINT searcher_events_pkey PRIMARY KEY (id);


--
-- Name: steps steps_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.steps
    ADD CONSTRAINT steps_pkey PRIMARY KEY (trial_id, id);


--
-- Name: templates templates_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.templates
    ADD CONSTRAINT templates_pkey PRIMARY KEY (name);


--
-- Name: trial_logs trial_logs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.trial_logs
    ADD CONSTRAINT trial_logs_pkey PRIMARY KEY (id);


--
-- Name: trials trials_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.trials
    ADD CONSTRAINT trials_pkey PRIMARY KEY (id);


--
-- Name: user_sessions user_sessions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_sessions
    ADD CONSTRAINT user_sessions_pkey PRIMARY KEY (id);


--
-- Name: users users_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);


--
-- Name: users users_username_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_username_key UNIQUE (username);


--
-- Name: validations validations_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.validations
    ADD CONSTRAINT validations_pkey PRIMARY KEY (id);


--
-- Name: validations validations_trial_step_unique; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.validations
    ADD CONSTRAINT validations_trial_step_unique UNIQUE (trial_id, step_id);


--
-- Name: ix_checkpoints_trial_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX ix_checkpoints_trial_id ON public.checkpoints USING btree (trial_id);


--
-- Name: ix_searcher_events_experiment_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX ix_searcher_events_experiment_id ON public.searcher_events USING btree (experiment_id);


--
-- Name: ix_trial_logs_trial_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX ix_trial_logs_trial_id ON public.trial_logs USING btree (trial_id);


--
-- Name: ix_trials_experiment_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX ix_trials_experiment_id ON public.trials USING btree (experiment_id);


--
-- Name: ix_trials_warm_start_checkpoint_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX ix_trials_warm_start_checkpoint_id ON public.trials USING btree (warm_start_checkpoint_id);


--
-- Name: ix_validations_trial_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX ix_validations_trial_id ON public.validations USING btree (trial_id);


--
-- Name: agent_user_groups agent_user_groups_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.agent_user_groups
    ADD CONSTRAINT agent_user_groups_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id);


--
-- Name: checkpoints checkpoints_trial_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.checkpoints
    ADD CONSTRAINT checkpoints_trial_id_fkey FOREIGN KEY (trial_id, step_id) REFERENCES public.steps(trial_id, id) ON DELETE CASCADE;


--
-- Name: experiments experiments_owner_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.experiments
    ADD CONSTRAINT experiments_owner_fkey FOREIGN KEY (owner_id) REFERENCES public.users(id);


--
-- Name: searcher_events searcher_events_experiment_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.searcher_events
    ADD CONSTRAINT searcher_events_experiment_id_fkey FOREIGN KEY (experiment_id) REFERENCES public.experiments(id);


--
-- Name: steps steps_trial_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.steps
    ADD CONSTRAINT steps_trial_id_fkey FOREIGN KEY (trial_id) REFERENCES public.trials(id);


--
-- Name: trial_logs trial_logs_trial_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.trial_logs
    ADD CONSTRAINT trial_logs_trial_id_fkey FOREIGN KEY (trial_id) REFERENCES public.trials(id);


--
-- Name: trials trials_experiment_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.trials
    ADD CONSTRAINT trials_experiment_id_fkey FOREIGN KEY (experiment_id) REFERENCES public.experiments(id);


--
-- Name: trials trials_warm_start_checkpoint_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.trials
    ADD CONSTRAINT trials_warm_start_checkpoint_id_fkey FOREIGN KEY (warm_start_checkpoint_id) REFERENCES public.checkpoints(id);


--
-- Name: user_sessions user_sessions_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_sessions
    ADD CONSTRAINT user_sessions_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id);


--
-- Name: validations validations_trial_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.validations
    ADD CONSTRAINT validations_trial_id_fkey FOREIGN KEY (trial_id, step_id) REFERENCES public.steps(trial_id, id) ON DELETE CASCADE;


--
-- PostgreSQL database dump complete
--

RESET search_path;
INSERT INTO users (username, admin, active) VALUES ('admin', 't', 't'), ('determined', 'f', 't');
