package project

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/workspace"
)

// ProjectByName returns a project's ID if it exists in the given workspace and is not archived.
func ProjectByName(ctx context.Context, workspaceName string, projectName string) (int, error) {
	workspace, err := workspace.WorkspaceByName(ctx, workspaceName)
	if err != nil {
		return 1, err
	}
	if workspace.Archived {
		return 1, fmt.Errorf("workspace is archived and cannot add new experiments")
	}

	var pID int
	var archived bool
	err = db.Bun().NewSelect().
		Table("projects").
		Column("id").
		Column("archived").
		Where("workspace_id = ?", workspace.ID).
		Where("name = ?", projectName).
		Scan(ctx, &pID, &archived)
	if err == sql.ErrNoRows {
		return 1, db.ErrNotFound
	}
	if err != nil {
		return 1, err
	}
	if archived {
		return 1, fmt.Errorf("project is archived and cannot add new experiments")
	}
	return pID, nil
}

// ProjectIDByName returns a project's ID if it exists in the given workspace.
func ProjectIDByName(ctx context.Context, workspaceID int, projectName string) (*int, error) {
	var pID int
	err := db.Bun().NewRaw("SELECT id FROM projects WHERE name = ? AND workspace_id = ?",
		projectName, workspaceID).Scan(ctx, &pID)
	if err != nil {
		return nil, err
	}
	return &pID, nil
}
