WITH ss AS (
  SELECT trial_id, state, end_time, metrics, total_batches, total_records, total_epochs, trial_run_id, archived, computed_records
  FROM steps
  WHERE trial_id=$1
  ORDER BY total_batches DESC
  LIMIT 1)
INSERT INTO steps
  (trial_id, state, end_time, metrics, total_batches, total_records, total_epochs, trial_run_id, archived, computed_records)
  SELECT trial_id, state, end_time, metrics, total_batches+g, total_records, total_epochs, trial_run_id, archived, computed_records
  FROM ss, generate_series(1, $2) g
RETURNING false;
