DO $$
  DECLARE checkpoints text;
  DECLARE checkpoints_view text;
  DECLARE checkpoints_old_view text;
  DECLARE checkpoints_new_view text;
  DECLARE proto_checkpoints_view text;
  DECLARE exec_text text;
begin
  ALTER TYPE public.checkpoint_state RENAME TO _checkpoint_state;

  CREATE TYPE public.checkpoint_state AS ENUM (
    'ACTIVE',
    'COMPLETED',
    'ERROR',
    'DELETED',
    'PARTIALLY_DELETED'
  );


  proto_checkpoints_view := pg_get_viewdef('proto_checkpoints_view');
  checkpoints_view := pg_get_viewdef('checkpoints_view');
  checkpoints := pg_get_viewdef('checkpoints');
  checkpoints_old_view := pg_get_viewdef('checkpoints_old_view');
  checkpoints_new_view := pg_get_viewdef('checkpoints_new_view');

  DROP VIEW proto_checkpoints_view;
  DROP VIEW checkpoints_view;
  DROP VIEW checkpoints;
  DROP VIEW checkpoints_old_view;
  DROP VIEW checkpoints_new_view;


  ALTER TABLE public.raw_checkpoints ALTER COLUMN state
    SET DATA TYPE public.checkpoint_state USING (state::text::checkpoint_state);
  ALTER TABLE public.checkpoints_v2 ALTER COLUMN state
    SET DATA TYPE public.checkpoint_state USING (state::text::checkpoint_state);

  exec_text := format('CREATE VIEW checkpoints_old_view AS %s', checkpoints_old_view);
  execute exec_text;
  exec_text := format('CREATE VIEW checkpoints_new_view AS %s', checkpoints_new_view);
  execute exec_text;
  exec_text := format('CREATE VIEW checkpoints AS %s', checkpoints);
  execute exec_text;
  exec_text := format('CREATE VIEW checkpoints_view AS %s', checkpoints_view);
  execute exec_text;
  exec_text := format('CREATE VIEW proto_checkpoints_view AS %s', proto_checkpoints_view);
  execute exec_text;

  DROP TYPE public._checkpoint_state;
end $$;
