CREATE OR REPLACE FUNCTION prevent_auto_created_namespace_change() RETURNS TRIGGER AS $$
        BEGIN
                RAISE integrity_constraint_violation 
                        USING MESSAGE = 'Cannot change auto_created_namespace_name once set';
        END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS on_update_auto_created_namespace_name ON workspaces CASCADE;

-- We want to enforce immutability of a workspace's auto_created_namespace_name.
CREATE TRIGGER on_update_auto_created_namespace_name
   BEFORE UPDATE ON workspaces
   FOR EACH ROW
   WHEN (OLD.auto_created_namespace_name IS NOT NULL AND OLD.auto_created_namespace_name
     IS DISTINCT FROM  NEW.auto_created_namespace_name)
   EXECUTE PROCEDURE prevent_auto_created_namespace_change();
