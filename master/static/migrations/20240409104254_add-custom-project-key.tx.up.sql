CREATE OR REPLACE FUNCTION function_generate_project_key(input_string TEXT)
RETURNS TEXT AS $$
DECLARE
    prefix TEXT;
    count_suffix TEXT;
	count_value INT;
BEGIN
    -- Take the first 3 characters of the input string
    prefix := LEFT(input_string, 3);
	
   	execute format('SELECT COUNT(*)+1 FROM projects WHERE key ILIKE $1') 
   		into count_value
   		using prefix || '%';
   	count_suffix := CAST(count_value as text);

    -- Concatenate prefix and count suffix
    RETURN lower(prefix || count_suffix);
END;
$$ LANGUAGE plpgsql;

ALTER TABLE projects ADD COLUMN key VARCHAR(5) UNIQUE;

UPDATE projects SET key = function_generate_project_key(name) WHERE key IS NULL;
