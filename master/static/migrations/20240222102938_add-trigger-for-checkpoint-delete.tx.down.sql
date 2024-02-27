DROP TRIGGER IF EXISTS on_checkpoint_deletion ON checkpoints_v2;

DROP FUNCTION IF EXISTS abort_checkpoint_delete();
