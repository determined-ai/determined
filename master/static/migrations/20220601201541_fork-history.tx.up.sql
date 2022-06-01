ALTER TABLE experiments ADD COLUMN lineage integer[];
UPDATE experiments SET lineage = ARRAY[] WHERE parent_id IS NULL;
UPDATE experiments SET lineage = ARRAY[parent_id] WHERE parent_id IS NOT NULL;

CREATE OR REPLACE FUNCTION find_exp_parents (exp_id int)
RETURNS VOID
language plpgsql
AS $$
BEGIN
    WITH e AS (SELECT lineage[1] AS oldest_parent, lineage FROM experiments WHERE id = exp_id),
    p AS (SELECT parent_id FROM experiments, e WHERE id = e.oldest_parent)
    UPDATE experiments SET lineage = array_prepend((SELECT parent_id FROM p), lineage)
      WHERE id = exp_id
      AND (SELECT parent_id FROM p) IS NOT NULL;
END $$;

DO
$do$
BEGIN
   FOR i IN 1..25 LOOP
      PERFORM find_exp_parents(id)
      FROM experiments
      WHERE parent_id IS NOT NULL;
   END LOOP;
END
$do$;
