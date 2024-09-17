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

// SetNTSCConfigPolicies adds the NTSC invariant config and constraints config policies to
// the database.
func SetNTSCConfigPolicies(ctx context.Context,
	ntscTCPs *model.NTSCTaskConfigPolicies,
) error {
	return db.Bun().RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		return SetNTSCConfigPoliciesTx(ctx, &tx, ntscTCPs)
	})
}

// SetNTSCConfigPoliciesTx adds the NTSC invariant config and constraints config policies to
// the database.
func SetNTSCConfigPoliciesTx(ctx context.Context, tx *bun.Tx,
	ntscTCPs *model.NTSCTaskConfigPolicies,
) error {
	if ntscTCPs.WorkloadType != model.NTSCType {
		return status.Error(codes.InvalidArgument,
			"invalid workload type for config policies: "+ntscTCPs.WorkloadType)
	}

	q := `
		INSERT INTO task_config_policies (workspace_id, workload_type, last_updated_by,
			last_updated_time,  invariant_config, constraints) VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT (workspace_id, workload_type) WHERE workspace_id IS NOT NULL
			DO UPDATE SET last_updated_by = ?, last_updated_time = ?, invariant_config = ?, 
			constraints = ?
		`
	if ntscTCPs.WorkspaceID == nil {
		q = `
			INSERT INTO task_config_policies (workspace_id, workload_type, last_updated_by,
				last_updated_time, invariant_config, constraints) VALUES (?, ?, ?, ?, ?, ?)
			ON CONFLICT (workload_type) WHERE workspace_id IS NULL
				DO UPDATE SET last_updated_by = ?, last_updated_time = ?, invariant_config = ?, 
				constraints = ?
			`
	}
	_, err := db.Bun().NewRaw(q, ntscTCPs.WorkspaceID, model.NTSCType,
		ntscTCPs.LastUpdatedBy, ntscTCPs.LastUpdatedTime, ntscTCPs.InvariantConfig,
		ntscTCPs.Constraints, ntscTCPs.LastUpdatedBy, ntscTCPs.LastUpdatedTime,
		ntscTCPs.InvariantConfig, ntscTCPs.Constraints).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("error setting NTSC task config policies: %w", err)
	}

	return nil
}

// GetNTSCConfigPolicies retrieves the invariant NTSC config and constraints for the
// given scope (global or workspace-level).
func GetNTSCConfigPolicies(ctx context.Context,
	scope *int,
) (*model.NTSCTaskConfigPolicies, error) {
	var ntscTCP model.NTSCTaskConfigPolicies
	wkspQuery := wkspIDQuery
	if scope == nil {
		wkspQuery = wkspIDGlobalQuery
	}
	err := db.Bun().NewSelect().
		Model(&ntscTCP).
		Where(wkspQuery, scope).
		Where("workload_type = ?", model.NTSCType).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("error retrieving NTSC task config policies for "+
			"workspace with ID %d: %w", scope, err)
	}
	return &ntscTCP, nil
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
