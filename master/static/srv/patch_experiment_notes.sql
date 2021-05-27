UPDATE experiments e
SET notes = $2
WHERE e.id = $1
RETURNING e.id
