UPDATE checkpoints SET  metadata = $2
WHERE uuid = $1
RETURNING metadata
