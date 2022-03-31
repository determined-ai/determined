WITH update_v1 AS (
    UPDATE raw_checkpoints SET metadata = $2
    WHERE uuid = $1
    RETURNING metadata
), update_v2 AS (
    UPDATE checkpoints_v2 SET metadata = $2
    WHERE uuid = $1
    RETURNING metadata
)
SELECT metadata
FROM update_v1
UNION ALL update_v2
WHERE metadata IS NOT NULL
