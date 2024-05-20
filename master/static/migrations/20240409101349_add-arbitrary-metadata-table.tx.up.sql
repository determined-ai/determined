-- CREATE RUN METADATA TABLE
CREATE TABLE runs_metadata (
    run_id INTEGER PRIMARY KEY,
    metadata JSONB,
    FOREIGN KEY (run_id) REFERENCES runs(id) ON DELETE CASCADE
);

-- CREATE INDEX TABLE
CREATE TABLE runs_metadata_index (
    id SERIAL PRIMARY KEY,
    run_id INTEGER,
    flat_key VARCHAR,
    value TEXT,
    data_type VARCHAR, -- 'string', 'number', 'boolean', 'timestamp', null
    project_id INTEGER,
    FOREIGN KEY (run_id) REFERENCES runs(id) ON DELETE CASCADE,
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
);
-- CREATE FILTER-OPTIMIZING INDEXES
CREATE INDEX idx_flat_key ON runs_metadata_index (flat_key);
CREATE INDEX idx_flat_key_value ON runs_metadata_index (flat_key, value);

-- Keep the run metadata indexes up to date with the project_id
CREATE OR REPLACE FUNCTION func_update_run_metadata_index_project_id()
RETURNS TRIGGER AS $$
BEGIN
UPDATE runs_metadata_index
    SET project_id = NEW.project_id
    WHERE run_id = NEW.id;
RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trigger_update_run_metadata_index_project_id ON runs;
CREATE TRIGGER trigger_update_run_metadata_index_project_id
AFTER UPDATE OF project_id ON runs
FOR EACH ROW EXECUTE PROCEDURE func_update_run_metadata_index_project_id();
