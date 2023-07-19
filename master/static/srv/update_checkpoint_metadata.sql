UPDATE checkpoints_v2 SET metadata = $2
WHERE uuid = $1
RETURNING metadata
