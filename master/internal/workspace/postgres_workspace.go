package workspace

import (
	"context"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/model"
)

// WorkspaceByName returns a workspace given it's name.
func WorkspaceByName(ctx context.Context, workspaceName string) (*model.Workspace, error) {
	w := model.Workspace{}
	err := db.Bun().NewSelect().Model(&w).Where("name = ?", workspaceName).Scan(ctx)
	if err != nil {
		return nil, err
	}
	return &w, nil
}

// ProjectIdByName returns a project's ID if it exists in the given workspace.
func ProjectIdByName(ctx context.Context, workspaceId int, projectName string) (*int, error) {
	var pId int
	err := db.Bun().NewSelect().Model(&pId).Table("projects").Where("name = ?", projectName).Where("workspaceId = ?", workspaceId).Scan(ctx)
	if err != nil {
		return nil, err
	}
	return &pId, nil
}
