CREATE TABLE IF NOT EXISTS allocation_workspace_info (
    allocation_id TEXT PRIMARY KEY, 
    workspace_id integer,
    workspace_name text,
    experiment_id INT
);

WITH 
alloc_experiments AS (
    SELECT
        allocations.allocation_id,
        rtrim(substring(task_id, '\d+?\.'), '.')::int AS experiment_id
    FROM
        allocations
),
allocation_workspace AS (
    SELECT
        allocations.allocation_id,
        alloc_experiments.experiment_id,
        COALESCE(projects.workspace_id, (c.generic_command_spec->'Metadata'->>'workspace_id')::int) AS workspace_id
    FROM
        allocations
    LEFT JOIN 
        command_state c ON allocations.task_id = c.task_id
    LEFT JOIN
        alloc_experiments ON allocations.allocation_id = alloc_experiments.allocation_id
    LEFT JOIN
        experiments ON alloc_experiments.experiment_id = experiments.id
    LEFT JOIN
        projects ON experiments.project_id = projects.id
)
-- Populate the new table with existing data based on the previous method of resolving the workspace info & experiment_id
INSERT INTO allocation_workspace_info (allocation_id, workspace_id, workspace_name, experiment_id)
SELECT
	allocation_workspace.allocation_id,
    allocation_workspace.workspace_id,
    workspaces.name,
    allocation_workspace.experiment_id
FROM 
    allocation_workspace
    LEFT JOIN
        workspaces ON allocation_workspace.workspace_id = workspaces.id
;
