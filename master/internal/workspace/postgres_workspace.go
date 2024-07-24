package workspace

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/pkg/errors"
	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/set"
)

// AddWorkspace adds the given workspace to the database.
func AddWorkspace(ctx context.Context, workspace *model.Workspace, tx *bun.Tx) error {
	if tx != nil {
		_, err := tx.NewInsert().Model(workspace).Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed adding workspace %s to the database: %w: %w", workspace.Name,
				db.ErrDuplicateRecord, err)
		}
	} else {
		_, err := db.Bun().NewInsert().Model(workspace).Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed adding workspace %s to the database: %w: %w", workspace.Name,
				db.ErrDuplicateRecord, err)
		}
	}
	return nil
}

// WorkspaceByName returns a workspace given it's name.
func WorkspaceByName(ctx context.Context, workspaceName string) (*model.Workspace, error) {
	var w model.Workspace
	err := db.Bun().NewSelect().Model(&w).Where("name = ?", workspaceName).Scan(ctx)
	if err == sql.ErrNoRows {
		return nil, db.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("getting workspace: %w", err)
	}
	return &w, nil
}

// Exists returns if the workspace exists and is not archived.
func Exists(ctx context.Context, id int) (bool, error) {
	return db.Bun().NewSelect().Table("workspaces").
		Where("id = ?", id).
		Where("archived = false").
		Limit(1).
		Exists(ctx)
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

// WorkspaceByProjectID returns a workspace given a project ID.
func WorkspaceByProjectID(ctx context.Context, projectID int) (*model.Workspace, error) {
	var w model.Workspace
	err := db.Bun().NewSelect().Model(&w).Where(
		"id = (SELECT workspace_id FROM projects WHERE id = ?)",
		projectID).Scan(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get workspace for project %d", projectID)
	}
	return &w, nil
}

// WorkspacesIDsByExperimentIDs gets workspace IDs associated with each experiment.
func WorkspacesIDsByExperimentIDs(ctx context.Context, expIDs []int) ([]int, error) {
	if len(expIDs) == 0 {
		return nil, nil
	}

	res := []struct {
		ExperimentID int
		WorkspaceID  int
	}{}
	err := db.Bun().NewRaw(`
SELECT e.id AS experiment_id, p.workspace_id AS workspace_id
FROM experiments e
JOIN projects p ON e.project_id = p.id
WHERE e.id IN (?)
`, bun.In(expIDs)).Scan(ctx, &res)
	if err != nil {
		return nil, fmt.Errorf("getting experiment's %v workspace IDs: %w", expIDs, err)
	}

	if len(expIDs) != len(res) {
		return nil, fmt.Errorf("expected %d results from expIDs %v got %d instead %v",
			len(expIDs), expIDs, len(res), res)
	}

	expIDToWorkspace := make(map[int]int)
	for _, r := range res {
		expIDToWorkspace[r.ExperimentID] = r.WorkspaceID
	}

	var output []int
	for _, expID := range expIDs {
		output = append(output, expIDToWorkspace[expID])
	}
	return output, nil
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

// GetNamespaceFromWorkspace returns the namespace for the given workspace and kubernetes cluster.
func GetNamespaceFromWorkspace(ctx context.Context, workspaceName string, clusterName string) (string, error) {
	var ns string
	err := db.Bun().
		NewSelect().
		TableExpr("workspaces as w").
		ColumnExpr("n.namespace").
		Join("JOIN workspace_namespace_bindings AS n ON  n.workspace_id = w.id").
		Where("w.name = ? and n.cluster_name = ?", workspaceName, clusterName).
		Scan(ctx, &ns)
	if err != nil {
		return "", fmt.Errorf("failed to get namespace: %w", err)
	}
	return ns, nil
}

// GetAllNamespacesForRM gets all namespaces associated with a particular kubernetes cluster. defaultNs is an optional
// parameter, if there is no defaultNs provided, the "default" namespace will be added to the list instead.
func GetAllNamespacesForRM(ctx context.Context, rmName string) ([]string, error) {
	var ns []string
	err := db.Bun().
		NewSelect().
		Table("workspace_namespace_bindings").
		ColumnExpr("DISTINCT namespace").
		Where("cluster_name = ?", rmName).
		Scan(ctx, &ns)
	if err != nil {
		return ns, fmt.Errorf("failed to get all namespaces for %v: %w", rmName, err)
	}
	return ns, nil
}

// AddWorkspaceNamespaceBinding adds a workspace-namespace binding.
func AddWorkspaceNamespaceBinding(ctx context.Context, wkspNmsp *model.WorkspaceNamespace,
	tx *bun.Tx,
) error {
	if tx != nil {
		_, err := tx.NewInsert().Model(wkspNmsp).Exec(ctx)
		if err != nil {
			return fmt.Errorf("error adding workspace-namespace binding to database: %w", err)
		}
	} else {
		_, err := db.Bun().NewInsert().Model(wkspNmsp).Exec(ctx)
		if err != nil {
			return fmt.Errorf("error adding workspace-namespace binding to database: %w", err)
		}
	}
	return nil
}

// GetWorkspaceNamespaceBindings gets the workspace-namespace bindings for a given workspace.
func GetWorkspaceNamespaceBindings(ctx context.Context,
	wkspID int,
) ([]model.WorkspaceNamespace, error) {
	var workspaceNamespaceBindings []model.WorkspaceNamespace
	err := db.Bun().NewSelect().
		Model(&model.WorkspaceNamespace{}).
		Where("workspace_id = ?", wkspID).
		Scan(ctx, &workspaceNamespaceBindings)
	if err != nil {
		return nil, err
	}
	return workspaceNamespaceBindings, nil
}

// DeleteWorkspaceNamespaceBindings deletes the workspace-namespace binding.
func DeleteWorkspaceNamespaceBindings(ctx context.Context, wkspID int,
	clusterNames []string, tx *bun.Tx,
) ([]model.WorkspaceNamespace, error) {
	var deletedBindings []model.WorkspaceNamespace

	_, err := tx.NewDelete().Model(&model.WorkspaceNamespace{}).
		Where("workspace_id = ?", wkspID).
		Where("cluster_name in (?)", bun.In(clusterNames)).
		Returning("*").
		Exec(ctx, &deletedBindings)
	if err != nil {
		return nil, fmt.Errorf(`error deleting workspace-namespace binding with workspace-id %d`,
			wkspID)
	}
	return deletedBindings, nil
}

// GetNumWorkspacesUsingNamespaceInCluster gets the number of Workspaces that are
// using a particular namespace for the given cluster.
func GetNumWorkspacesUsingNamespaceInCluster(ctx context.Context, clusterName string,
	namespaceName string,
) (int, error) {
	return db.Bun().NewSelect().
		Table("workspace_namespace_bindings").
		ColumnExpr("count(*)").
		Where("cluster_name = ? and namespace = ?", clusterName, namespaceName).
		Count(ctx)
}
