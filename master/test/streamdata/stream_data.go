//go:build integration
// +build integration

package streamdata

import (
	"context"
	"database/sql"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/model"
)

// ExecutableQuery an interface that requires queries of this type to have an exec function.
type ExecutableQuery interface {
	Exec(ctx context.Context, dest ...interface{}) (sql.Result, error)
}

// GetAddProjectQuery constructs a query to create a new project in the db.
func GetAddProjectQuery(proj model.Project) ExecutableQuery {
	return db.Bun().NewInsert().Model(&proj).ExcludeColumn(
		"workspace_name",
		"username",
		"num_active_experiments",
		"num_experiments",
		"last_experiment_started_at",
	)
}

// GetUpdateProjectQuery constructs a query to update a project.
func GetUpdateProjectQuery(proj model.Project) ExecutableQuery {
	return db.Bun().NewUpdate().Model(&proj).OmitZero().Where("id = ?", proj.ID)
}

// GetDeleteProjectQuery constructs a query to delete a project.
func GetDeleteProjectQuery(proj model.Project) ExecutableQuery {
	return db.Bun().NewDelete().Model(&proj).Where("id = ?", proj.ID)
}
