package workspace

import (
	"context"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/model"
)

// WorkspaceByName returns a workspace given it's name.
func WorkspaceByName(ctx context.Context, workspaceName string) (*model.Workspace, error) {
	var w model.Workspace
	err := db.Bun().NewSelect().Model(&w).Where("name = ?", workspaceName).Scan(ctx)
	if err != nil {
		return nil, err
	}
	return &w, nil
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
