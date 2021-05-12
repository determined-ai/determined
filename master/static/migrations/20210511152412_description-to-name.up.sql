UPDATE experiments
SET config = jsonb_set(config, '{name}',
  to_jsonb(coalesce(config->>'description', 'Experiment ' || id)::text)
)
WHERE config->>'name' IS NULL;
