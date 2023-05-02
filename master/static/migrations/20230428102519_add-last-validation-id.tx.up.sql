ALTER TABLE public.trials ADD COLUMN latest_validation_id int
    REFERENCES public.raw_validations(id)
    ON DELETE SET NULL
    DEFAULT NULL;

UPDATE trials SET latest_validation_id = sub.id
FROM (
    SELECT validations.id, trial_id, ROW_NUMBER() OVER(
        PARTITION BY trial_id
        ORDER BY validations.end_time DESC
    ) AS rank
    FROM validations
    JOIN trials t on validations.trial_id = t.id
    JOIN experiments e on t.experiment_id = e.id
    WHERE (
        validations.metrics->'validation_metrics'->>(e.config->'searcher'->>'metric')
    ) IS NOT NULL
) sub
WHERE
  rank = 1 AND
  sub.trial_id = trials.id;
