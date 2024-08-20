ALTER TABLE public.workspaces ADD COLUMN auto_created_namespace_name text;

CREATE OR REPLACE FUNCTION abort_update() RETURNS TRIGGER AS $$
        BEGIN
                RETURN NULL;
        END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER on_update_auto_created_namespace_name
   BEFORE UPDATE ON workspaces
   FOR EACH ROW 
   WHEN (OLD.auto_created_namespace_name <> NEW.auto_created_namespace_name)
   EXECUTE PROCEDURE abort_update();
