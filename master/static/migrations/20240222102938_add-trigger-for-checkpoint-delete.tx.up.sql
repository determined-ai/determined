CREATE OR REPLACE FUNCTION abort_checkpoint_delete() RETURNS TRIGGER AS $$
BEGIN   
    IF OLD.state <> 'DELETED' THEN 
        RETURN NULL;
    END IF;
   RETURN OLD;
END
$$ LANGUAGE plpgsql;

CREATE TRIGGER on_checkpoint_deletion
   BEFORE DELETE ON checkpoints_v2
   FOR EACH ROW EXECUTE PROCEDURE abort_checkpoint_delete();
