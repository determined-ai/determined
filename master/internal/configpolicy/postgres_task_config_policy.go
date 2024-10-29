package configpolicy

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/model"
)

const (
	wkspIDQuery          = "workspace_id = ?"
	wkspIDGlobalQuery    = "workspace_id IS ?"
	invalidPolicyTypeErr = "invalid policy type"
	// DefaultInvariantConfigStr is the default invariant config val used for tests.
	DefaultInvariantConfigStr = `{
	"description": "random description", 
	"resources": {"slots": 4, "max_slots": 8}
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

	q = q.Set("last_updated_by = ?, last_updated_time = ?, invariant_config = ?, constraints = ?",
		tcp.LastUpdatedBy, tcp.LastUpdatedTime, tcp.InvariantConfig, tcp.Constraints)

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

// GetConfigPolicyField fetches the field from an invariant_config or constraints policyType, in order
// of precedence. Global scope has highest precedence, then workspace. Returns nil if none is found.
// **NOTE** The field arguments are wrapped in bun.Safe, so you  must specify the "raw" string
// exactly as you wish for it to be accessed in the database. For example, if you want to access
// resources.max_slots, the field argument should be "'resources' -> 'max_slots'" NOT
// "resources -> max_slots".
// **NOTE**When using this function to retrieve an object of Kind Pointer, set T as the Type of
// object that the Pointer wraps. For example, if we want an object of type *int, set T to int, so
// that when its pointer is returned, you get an object of type *int.
func GetConfigPolicyField[T any](ctx context.Context, wkspID *int, policyType, field, workloadType string) (*T,
	error,
) {
	if policyType != "invariant_config" && policyType != "constraints" {
		return nil, fmt.Errorf("%s :%s", invalidPolicyTypeErr, policyType)
	}

	var confBytes []byte
	var conf T
	err := db.Bun().RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		var globalBytes []byte
		err := tx.NewSelect().Table("task_config_policies").
			ColumnExpr("? -> ?", bun.Safe(policyType), bun.Safe(field)).
			Where("workspace_id IS NULL").
			Where("workload_type = ?", workloadType).Scan(ctx, &globalBytes)
		if err == nil && len(globalBytes) > 0 {
			confBytes = globalBytes
		}
		if err != nil && err != sql.ErrNoRows {
			return err
		}

		var wkspBytes []byte
		err = tx.NewSelect().Table("task_config_policies").
			ColumnExpr("? -> ?", bun.Safe(policyType), bun.Safe(field)).
			Where("workspace_id = ?", wkspID).
			Where("workload_type = ?", workloadType).Scan(ctx, &wkspBytes)
		if err == nil && len(globalBytes) == 0 {
			confBytes = wkspBytes
		}
		return err
	})
	if err == sql.ErrNoRows || len(confBytes) == 0 {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("error getting config field %s: %w", field, err)
	}

	err = json.Unmarshal(confBytes, &conf)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling config field: %w", err)
	}

	return &conf, nil
}
