DO $$
    BEGIN
        IF NOT EXISTS (SELECT * FROM INFORMATION_SCHEMA.COLUMNS
                       WHERE TABLE_NAME = 'trials'
                       AND COLUMN_NAME = 'total_batches') THEN

            ALTER TABLE trials ADD COLUMN total_batches INTEGER NOT NULL DEFAULT 0;

            UPDATE public.trials SET total_batches=sub.max_total_batches
            FROM (
                SELECT trial_id, max(total_batches) AS max_total_batches
                FROM (
                    SELECT trial_id, coalesce(max(s.total_batches), 0) AS total_batches
                    FROM steps s GROUP by trial_id 
                    UNION ALL
                    SELECT trial_id, coalesce(max(v.total_batches), 0) AS total_batches
                    FROM validations v  GROUP by trial_id
                ) AS q GROUP by trial_id 
            ) AS sub 
            WHERE public.trials.id = sub.trial_id;
        END IF;
    END;
$$
