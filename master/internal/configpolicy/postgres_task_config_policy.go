package configpolicy

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/model"
)

const (
	wkspIDQuery       = "workspace_id = ?"
	wkspIDGlobalQuery = "workspace_id IS ?"
	// DefaultInvariantConfigStr is the default invariant config val used for tests.
	DefaultInvariantConfigStr = `{
	"description": "random description", 
	"resources": {"slots": 4, "max_slots": 8},
	"log_policies": [
		{
			"pattern": ".*CUDA out of memory.*",
			"actions": [
				{
					"signal": "CUDA OOM"
				}
			]
		},
		{
			"pattern": ".*uncorrectable ECC error encountered.*",
			"actions": [
				{
					"signal": "ECC Error"
				}
			]
		}
	]
	}`
	// DefaultConstraintsStr is the default constraints val used for tests.
	DefaultConstraintsStr = `{"priority_limit": 10, "resources": {"max_slots": 8}}`
)

// SetTaskConfigPolicies adds the task invariant config and constraints config policies to
// the database.
func SetTaskConfigPolicies(ctx context.Context,
	tcp *model.TaskConfigPolicies,
) error {
	return db.Bun().RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		return SetTaskConfigPoliciesTx(ctx, &tx, tcp)
	})
}

// SetTaskConfigPoliciesTx adds the task invariant config and constraints policies to
// the database.
func SetTaskConfigPoliciesTx(ctx context.Context, tx *bun.Tx,
	tcp *model.TaskConfigPolicies,
) error {
	q := db.Bun().NewInsert().Model(tcp)

	q = q.Set("last_updated_by = ?, last_updated_time = ?", tcp.LastUpdatedBy, tcp.LastUpdatedTime)
	q = q.Set("invariant_config = ?", tcp.InvariantConfig)
	q = q.Set("constraints = ?", tcp.Constraints)

	if tcp.WorkspaceID == nil {
		q = q.On("CONFLICT (workload_type) WHERE workspace_id IS NULL DO UPDATE")
	} else {
		q = q.On("CONFLICT (workspace_id, workload_type) WHERE workspace_id IS NOT NULL DO UPDATE")
	}

	_, err := q.Exec(ctx)
	if err != nil {
		return fmt.Errorf("error setting task config policies: %w", err)
	}
	return nil
}

// GetTaskConfigPolicies retrieves the invariant config and constraints for the
// given scope (global or workspace-level) and workload Type.
func GetTaskConfigPolicies(
	ctx context.Context, scope *int, workloadType string,
) (*model.TaskConfigPolicies, error) {
	var tcp model.TaskConfigPolicies
	wkspQuery := wkspIDQuery
	if scope == nil {
		wkspQuery = wkspIDGlobalQuery
	}
	err := db.Bun().NewSelect().
		Model(&tcp).
		Where(wkspQuery, scope).
		Where("workload_type = ?", workloadType).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) || errors.Cause(err) == sql.ErrNoRows {
			return &model.TaskConfigPolicies{}, nil
		}
		return nil, fmt.Errorf("error retrieving %v task config policies for "+
			"workspace with ID %d: %w", workloadType, scope, err)
	}
	return &tcp, nil
}

// DeleteConfigPolicies deletes the invariant experiment config and constraints for the
// given scope (global or workspace-level) and workload type.
func DeleteConfigPolicies(ctx context.Context,
	scope *int, workloadType string,
) error {
	wkspQuery := wkspIDQuery
	if scope == nil {
		wkspQuery = wkspIDGlobalQuery
	}

	_, err := db.Bun().NewDelete().
		Table("task_config_policies").
		Where(wkspQuery, scope).
		Where("workload_type = ?", workloadType).
		Exec(ctx)
	if err != nil {
		if scope == nil {
			return fmt.Errorf("error deleting global %s config policies:%w",
				strings.ToLower(workloadType), err)
		}
		return fmt.Errorf("error deleting %s config policies for workspace with ID %d: %w",
			strings.ToLower(workloadType), scope, err)
	}
	return nil
}
