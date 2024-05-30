ALTER TABLE projects ADD COLUMN IF NOT EXISTS key VARCHAR(5) UNIQUE;

DO $$
DECLARE
    project RECORD;
    prefix TEXT;
    prefix_length INT;
    suffix TEXT;
    max_key_length INT := 5;
    max_prefix_length INT := 3;
BEGIN
    FOR project IN SELECT * FROM projects WHERE key IS NULL LOOP
        -- Take the first 3 characters of the project name
        prefix := UPPER(LEFT(project.name, max_prefix_length));
        prefix_length := LENGTH(prefix);
        -- Generate a random suffix
        suffix := UPPER(LEFT(md5(random()::text), max_key_length - prefix_length));
        
        -- Check if the key already exists and loop until we find a unique key
        WHILE EXISTS(SELECT 1 FROM projects WHERE key = prefix || suffix) LOOP
           suffix := UPPER(LEFT(md5(random()::text), max_key_length - prefix_length));
        END LOOP;
        
        -- Update the project key
        UPDATE projects SET key = prefix || suffix WHERE id = project.id;
    END LOOP;
END;
$$ LANGUAGE plpgsql;
