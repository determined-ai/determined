DROP VIEW trials_augmented_view;

DROP VIEW steps;
DROP VIEW validations;
DROP VIEW checkpoints;

CREATE TYPE public.step_state AS ENUM (
    'ACTIVE',
    'COMPLETED',
    'ERROR'
);

CREATE TYPE public.validation_state AS ENUM (
    'ACTIVE',
    'COMPLETED',
    'ERROR'
);

ALTER TABLE raw_steps
	ADD COLUMN total_records integer NOT NULL DEFAULT 0,
	ADD COLUMN total_epochs real NOT NULL DEFAULT 0,
	ADD COLUMN computed_records integer NULL,
	ADD COLUMN state public.step_state NOT NULL DEFAULT 'COMPLETED';

ALTER TABLE raw_validations
	ADD COLUMN total_records integer NOT NULL DEFAULT 0,
	ADD COLUMN total_epochs real NOT NULL DEFAULT 0,
	ADD COLUMN computed_records integer NULL,
	ADD COLUMN state public.validation_state NOT NULL DEFAULT 'COMPLETED';

ALTER TABLE raw_checkpoints
	ADD COLUMN total_records integer NOT NULL DEFAULT 0,
	ADD COLUMN total_epochs real NOT NULL DEFAULT 0;

CREATE VIEW steps AS
	SELECT * FROM raw_steps WHERE NOT archived;
CREATE VIEW validations AS
	SELECT * FROM raw_validations WHERE NOT archived;
CREATE VIEW checkpoints AS
	SELECT * FROM raw_checkpoints WHERE NOT archived;

-- Copy from static/migrations/20220922114430_trials-collection.tx.up.sql
CREATE VIEW public.trials_augmented_view AS
  WITH b AS (
	select trial_id, max(total_batches) total_batches from steps group by trial_id
  )
  SELECT
	  t.id AS trial_id,
	  t.state AS state,
	  t.hparams AS hparams,
	  jsonb_collect(s.metrics->'avg_metrics') AS training_metrics,
	  jsonb_collect(v.metrics->'validation_metrics') AS validation_metrics,
	  t.tags AS tags,
	  t.start_time AS start_time,
	  t.end_time AS end_time,
	  max(e.config->'searcher'->>'name') as searcher_type,
	  max(e.id) AS experiment_id,
	  max(e.config->>'name') AS experiment_name,
	  max(e.config->>'description') AS experiment_description,
	  -- there's only one
	  jsonb_agg(e.config ->> 'labels'::text) AS experiment_labels,
	  max(e.owner_id) AS user_id,
	  max(e.project_id) AS project_id,
	  max(p.workspace_id) AS workspace_id,
	  -- temporary
	  max(b.total_batches) as total_batches,
	  max(e.config->'searcher'->>'metric') AS searcher_metric,
	  max(v.metrics->'validation_metrics'->>(e.config->'searcher'->>'metric'))::float8 AS searcher_metric_value,
	  max(CASE
		  WHEN coalesce((config->'searcher'->>'smaller_is_better')::boolean, true)
			THEN (v.metrics->'validation_metrics'->>(e.config->'searcher'->>'metric'))::float8
			ELSE -1.0 * (v.metrics->'validation_metrics'->>(e.config->'searcher'->>'metric'))::float8
	  END) AS searcher_metric_loss
  FROM trials t
  LEFT JOIN experiments e ON t.experiment_id = e.id
  LEFT JOIN projects p ON e.project_id = p.id
  LEFT JOIN validations v ON t.id = v.trial_id AND v.id = t.best_validation_id
  LEFT JOIN steps s on t.id = s.trial_id AND v.total_batches = s.total_batches
  LEFT JOIN b on t.id = b.trial_id
  GROUP BY t.id;
