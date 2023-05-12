DO $$
    BEGIN
        EXECUTE format('ALTER TABLE templates ADD workspace_id INT REFERENCES workspaces(id) NOT NULL DEFAULT %L'
            , (SELECT MIN(id) FROM workspaces WHERE name = 'Uncategorized'));
END $$;
