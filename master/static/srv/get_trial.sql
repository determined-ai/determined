SELECT Row_to_json(r1)
FROM
  (SELECT t.id,
          t.experiment_id,
          t.state,
          t.start_time,
          t.end_time,
          t.hparams,
          t.seed,
          t.warm_start_checkpoint_id,
          t.runner_state,

     (SELECT COALESCE(jsonb_agg(r2
                                ORDER BY r2.id ASC), '[]'::JSONB)
      FROM
        (SELECT s.id,
                s.trial_id,
                s.state,
                s.start_time,
                s.end_time,
                s.total_batches,

           (SELECT row_to_json(r3)
            FROM
              (SELECT c.id,
                      c.trial_id,
                      c.total_batches,
                      c.state,
                      c.start_time,
                      c.end_time,
                      c.uuid,
                      c.resources,
                      c.metadata
               FROM checkpoints c
               WHERE c.trial_id = t.id
                 AND c.total_batches = s.total_batches ) r3) AS CHECKPOINT,

           (SELECT row_to_json(r4)
            FROM
              (SELECT v.id,
                      v.trial_id,
                      v.total_batches,
                      v.state,
                      v.start_time,
                      v.end_time,
                      v.metrics
               FROM validations v
               WHERE v.trial_id = t.id
                 AND v.total_batches = s.total_batches ) r4) AS validation
         FROM steps s
         WHERE s.trial_id = t.id ) r2) AS steps
   FROM trials t
   WHERE t.id = $1 ) r1
