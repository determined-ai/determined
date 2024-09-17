package configpolicy

import (
	"context"
	"fmt"
	"strings"

	"github.com/uptrace/bun"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/model"
)

const (
	wkspIDQuery       = "workspace_id = ?"
	wkspIDGlobalQuery = "workspace_id IS ?"
)

// SetTaskConfigPolicies adds the task invariant config and constraints config policies to
// the database.
func SetTaskConfigPolicies(ctx context.Context,
	experimentTCP *model.TaskConfigPolicies,
) error {
	return db.Bun().RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		return SetTaskConfigPoliciesTx(ctx, &tx, experimentTCP)
	})
}

// SetTaskConfigPoliciesTx adds the task invariant config and constraints config policies to
// the database.
func SetTaskConfigPoliciesTx(ctx context.Context, tx *bun.Tx,
	tcp *model.TaskConfigPolicies,
) error {
	q := db.Bun().NewInsert().
		Model(tcp)

	if tcp.InvariantConfig == nil {
		q = q.ExcludeColumn("invariant_config")
	}
	if tcp.Constraints == nil {
		q = q.ExcludeColumn("constraints")
	}

	if tcp.WorkspaceID == nil {
		q = q.On("CONFLICT (workload_type) WHERE workspace_id IS NULL DO UPDATE")
	} else {
		q = q.On("CONFLICT (workspace_id, workload_type) WHERE workspace_id IS NOT NULL DO UPDATE")
	}

	q = q.Set("last_updated_by = ?, last_updated_time = ?", tcp.LastUpdatedBy, tcp.LastUpdatedTime)
	if tcp.InvariantConfig != nil {
		q = q.Set("invariant_config = ?", tcp.InvariantConfig)
	}
	if tcp.Constraints != nil {
		q = q.Set("constraints = ?", tcp.Constraints)
	}

	_, err := q.Exec(ctx)
	if err != nil {
		return fmt.Errorf("error setting task config policies: %w", err)
	}
	return nil
}

// GetTaskConfigPolicies retrieves the invariant config and constraints for the
// given scope (global or workspace-level) and workload Type.
func GetTaskConfigPolicies(ctx context.Context,
	scope *int, workloadType model.WorkloadType,
) (*model.TaskConfigPolicies, error) {
	var ntscTCP model.TaskConfigPolicies
	wkspQuery := wkspIDQuery
	if scope == nil {
		wkspQuery = wkspIDGlobalQuery
	}
	err := db.Bun().NewSelect().
		Model(&ntscTCP).
		Where(wkspQuery, scope).
		Where("workload_type = ?", workloadType.String()).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("error retrieving %v task config policies for "+
			"workspace with ID %d: %w", workloadType.String(), scope, err)
	}
	return &ntscTCP, nil
}

// DeleteConfigPolicies deletes the invariant experiment config and constraints for the
// given scope (global or workspace-level) and workload type.
func DeleteConfigPolicies(ctx context.Context,
	scope *int, workloadType model.WorkloadType,
) error {
	if workloadType == model.UnknownType {
		return status.Error(codes.InvalidArgument,
			"invalid workload type for config policies: "+workloadType.String())
	}
	wkspQuery := wkspIDQuery
	if scope == nil {
		wkspQuery = wkspIDGlobalQuery
	}

	_, err := db.Bun().NewDelete().
		Table("task_config_policies").
		Where(wkspQuery, scope).
		Where("workload_type = ?", workloadType.String()).
		Exec(ctx)
	if err != nil {
		if scope == nil {
			return fmt.Errorf("error deleting global %s config policies:%w",
				strings.ToLower(workloadType.String()), err)
		}
		return fmt.Errorf("error deleting %s config policies for workspace with ID %d: %w",
			strings.ToLower(workloadType.String()), scope, err)
	}
	return nil
}
