UPDATE experiments e
SET config = config || $2,
  name = $3,
  note = $4
WHERE e.id = $1
RETURNING e.id
