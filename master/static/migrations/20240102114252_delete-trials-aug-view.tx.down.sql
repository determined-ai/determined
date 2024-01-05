CREATE VIEW trials_augmented_view AS
 WITH b AS (
         SELECT steps.trial_id,
            max(steps.total_batches) AS total_batches
           FROM steps
          GROUP BY steps.trial_id
        )
 SELECT t.id AS trial_id,
    t.state,
    t.hparams,
    jsonb_collect(s.metrics -> 'avg_metrics'::text) AS training_metrics,
    jsonb_collect(v.metrics -> 'validation_metrics'::text) AS validation_metrics,
    t.tags,
    t.start_time,
    t.end_time,
    max((e.config -> 'searcher'::text) ->> 'name'::text) AS searcher_type,
    max(e.id) AS experiment_id,
    max(e.config ->> 'name'::text) AS experiment_name,
    max(e.config ->> 'description'::text) AS experiment_description,
    jsonb_agg(e.config ->> 'labels'::text) AS experiment_labels,
    max(e.owner_id) AS user_id,
    max(e.project_id) AS project_id,
    max(p.workspace_id) AS workspace_id,
    max(b.total_batches) AS total_batches,
    max((e.config -> 'searcher'::text) ->> 'metric'::text) AS searcher_metric,
    max((v.metrics -> 'validation_metrics'::text) ->> ((e.config -> 'searcher'::text) ->> 'metric'::text))::double precision AS searcher_metric_value,
    max(
        CASE
            WHEN COALESCE(((e.config -> 'searcher'::text) ->> 'smaller_is_better'::text)::boolean, true) THEN ((v.metrics -> 'validation_metrics'::text) ->> ((e.config -> 'searcher'::text) ->> 'metric'::text))::double precision
            ELSE '-1.0'::numeric::double precision * (((v.metrics -> 'validation_metrics'::text) ->> ((e.config -> 'searcher'::text) ->> 'metric'::text))::double precision)
        END) AS searcher_metric_loss
   FROM runs t
     LEFT JOIN experiments e ON t.experiment_id = e.id
     LEFT JOIN projects p ON e.project_id = p.id
     LEFT JOIN validations v ON t.id = v.trial_id AND v.id = t.best_validation_id
     LEFT JOIN steps s ON t.id = s.trial_id AND v.total_batches = s.total_batches
     LEFT JOIN b ON t.id = b.trial_id
  GROUP BY t.id;
