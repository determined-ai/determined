CREATE OR REPLACE FUNCTION function_generate_project_key(
    max_key_length INTEGER,
    max_prefix_length INTEGER,
    input_string TEXT
)
RETURNS TEXT AS $$
DECLARE
    prefix TEXT;
    prefix_length INT;
    suffix TEXT;
BEGIN
    -- Take the first 3 characters of the input string
    prefix := UPPER(LEFT(input_string, max_prefix_length));
    prefix_length := LENGTH(prefix);
    -- Generate a random suffix
    suffix := UPPER(LEFT(md5(random()::text), max_key_length - prefix_length));
    
    -- Check if the key already exists and loop until we find a unique key
    WHILE EXISTS(SELECT 1 FROM projects WHERE key = prefix || suffix) LOOP
       suffix := UPPER(LEFT(md5(random()::text), max_key_length - prefix_length));
    END LOOP;
    
    RETURN prefix || suffix;
END;
$$ LANGUAGE plpgsql;

ALTER TABLE projects ADD COLUMN IF NOT EXISTS key VARCHAR(5) UNIQUE;

UPDATE 
    projects 
SET 
    key = function_generate_project_key(5, 3, name)
WHERE key IS NULL;
