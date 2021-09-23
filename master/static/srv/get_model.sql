SELECT name, description, metadata, creation_time, last_updated_time, numVersions, labels, readme, username, archived FROM models WHERE id = $1;
