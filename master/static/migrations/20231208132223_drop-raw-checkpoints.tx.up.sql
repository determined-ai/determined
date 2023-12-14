ALTER TABLE checkpoints_v2 
    ALTER COLUMN id DROP DEFAULT;

-- We need to add `IF EXISTS` to these data drops in order to safely perform down migrations. 
-- Since this migration does not have a corresponding down migration, we need to verify existence
-- of the following functions, table, and sequence in order to migrate up and down from this 
-- schema version.

DROP FUNCTION IF EXISTS best_checkpoint_by_metric;
DROP FUNCTION IF EXISTS experiments_best_checkpoints_by_metric;
DROP TABLE IF EXISTS raw_checkpoints;
DROP SEQUENCE IF EXISTS checkpoints_id_seq;

CREATE SEQUENCE checkpoints_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

SELECT setval('checkpoints_id_seq', (SELECT MAX(id) FROM checkpoints_v2));

ALTER TABLE checkpoints_v2
    ALTER COLUMN id SET DEFAULT nextval('checkpoints_id_seq');
