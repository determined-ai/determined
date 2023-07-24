package workspace

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/set"
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

// WorkspaceIDsFromNames returns an unordered slice of workspaceIDs that correlate with the given
// workspace names.
func WorkspaceIDsFromNames(ctx context.Context, workspaceNames []string) (
	[]int32, error,
) {
	if len(workspaceNames) == 0 {
		return []int32{}, nil
	}
	var workspaces []model.Workspace
	err := db.Bun().NewSelect().
		Model(&workspaces).
		Where("name IN (?)", bun.In(workspaceNames)).
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	if len(workspaces) != len(workspaceNames) {
		var missing []string
		namesFound := set.New[string]()
		for _, workspace := range workspaces {
			namesFound.Insert(workspace.Name)
		}

		for _, name := range workspaceNames {
			if !namesFound.Contains(name) {
				missing = append(missing, name)
			}
		}

		return nil, fmt.Errorf("the following workspaces do not exist: %s", missing)
	}

	var workspaceIDs []int32
	for _, workspace := range workspaces {
		workspaceIDs = append(workspaceIDs, int32(workspace.ID))
	}
	return workspaceIDs, nil
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

// WorkspaceByProjectID returns a workspace given a project ID.
func WorkspaceByProjectID(ctx context.Context, projectID int) (*model.Workspace, error) {
	var w model.Workspace
	err := db.Bun().NewSelect().Model(&w).Where(
		"id = (SELECT workspace_id FROM projects WHERE id = ?)",
		projectID).Scan(ctx)
	if err != nil {
		return nil, err
	}
	return &w, nil
}

// AllWorkspaces returns all the workspaces that exist.
func AllWorkspaces(ctx context.Context) ([]*model.Workspace, error) {
	var w []*model.Workspace
	err := db.Bun().NewSelect().Model(&w).Scan(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get all workspaces")
	}
	return w, nil
}
