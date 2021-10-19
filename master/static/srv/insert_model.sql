INSERT INTO models (name, description, metadata, creation_time, last_updated_time)
VALUES ($1, $2, $3, $4, $5)
RETURNING name, description, metadata, creation_time, last_updated_time, id
