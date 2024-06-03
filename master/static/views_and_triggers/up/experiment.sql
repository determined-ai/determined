CREATE FUNCTION stream_experiment_notify(before jsonb, after jsonb) RETURNS integer
    LANGUAGE plpgsql
    AS $$
DECLARE
    output jsonb = NULL;
BEGIN
    IF before IS NOT NULL THEN
        output = jsonb_object_agg('before', before);
    END IF;
    IF after IS NOT NULL THEN
        IF output IS NULL THEN
            output = jsonb_object_agg('after', after);
        ELSE
            output = output || jsonb_object_agg('after', after);
    END IF;
END IF;
    PERFORM pg_notify('stream_experiment_chan', output::text);
return 0;
END;
$$;
CREATE FUNCTION stream_experiment_change() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
DECLARE
    n jsonb = NULL;
    o jsonb = NULL;
BEGIN
    IF (TG_OP = 'INSERT') THEN
        n = jsonb_build_object('id', NEW.id, 'workspace_id', to_jsonb((SELECT workspace_id FROM projects WHERE id = NEW.project_id)::text));
        PERFORM stream_experiment_notify(NULL, n);
    ELSEIF (TG_OP = 'UPDATE') THEN
        n = jsonb_build_object('id', NEW.id, 'workspace_id', to_jsonb((SELECT workspace_id FROM projects WHERE id = NEW.project_id)::text));
        o = jsonb_build_object('id', OLD.id, 'workspace_id', to_jsonb((SELECT workspace_id FROM projects WHERE id = OLD.project_id)::text));
        PERFORM stream_experiment_notify(o, n);
    ELSEIF (TG_OP = 'DELETE') THEN
        PERFORM stream_experiment_notify(jsonb_build_object('id', OLD.id), NULL);
        -- DELETEs trigger BEFORE, and must return a non-NULL value.
        return OLD;
    END IF;
    return NULL;
END;
$$;
CREATE TRIGGER stream_experiment_trigger_d 
    BEFORE DELETE 
        ON experiments 
            FOR EACH ROW EXECUTE PROCEDURE stream_experiment_change();
CREATE TRIGGER stream_experiment_trigger_iu 
    AFTER INSERT OR UPDATE 
        OF state, config, model_definition, start_time, end_time, archived, parent_id, owner_id, progress,
        original_config, notes, job_id, project_id, checkpoint_size, checkpoint_count, best_trial_id, unmanaged,
        external_experiment_id
            ON experiments 
                FOR EACH ROW EXECUTE PROCEDURE stream_experiment_change();


CREATE FUNCTION stream_experiment_change_by_project() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
DECLARE
    f record;
BEGIN
    FOR f in (SELECT * FROM experiments WHERE projectdel_id = NEW.id)
    LOOP
        PERFORM stream_experiment_notify(
            jsonb_build_object('id', f.id, 'workspace_id', to_jsonb(OLD.workspace_id::text)),
            jsonb_build_object('id', f.id, 'workspace_id', to_jsonb(NEW.workspace_id::text))
        );
    END LOOP;
    RETURN NEW;
END;
$$;
CREATE TRIGGER stream_experiment_trigger_by_project_iu 
    BEFORE UPDATE OF workspace_id 
        ON projects 
            FOR EACH ROW EXECUTE PROCEDURE stream_experiment_change_by_project();



CREATE FUNCTION stream_experiment_seq_modify() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    NEW.seq = nextval('stream_experiment_seq');
RETURN NEW;
END;
$$;
CREATE TRIGGER stream_experiment_trigger_seq
    BEFORE INSERT OR UPDATE
        OF state, config, model_definition, start_time, end_time, archived, parent_id, owner_id, progress,
        original_config, notes, job_id, project_id, checkpoint_size, checkpoint_count, best_trial_id, unmanaged,
        external_experiment_id
            ON experiments
                FOR EACH ROW EXECUTE PROCEDURE stream_experiment_seq_modify();

CREATE FUNCTION stream_experiment_seq_modify_by_project() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    UPDATE experiments SET seq = nextval('stream_experiment_seq') WHERE project_id = NEW.id;
RETURN NEW;
END;
$$;
CREATE TRIGGER stream_experiment_trigger_by_project 
    BEFORE UPDATE OF workspace_id 
        ON projects 
            FOR EACH ROW EXECUTE PROCEDURE stream_experiment_seq_modify_by_project();
