package db

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/uptrace/bun"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/determined-ai/determined/master/pkg/model"
)

// SetExperimentConfigPolicies adds the experiment invariant config and constraints config policies to
// the database.
func SetExperimentConfigPolicies(ctx context.Context,
	experimentTCP *model.ExperimentTaskConfigPolicies,
) error {
	return Bun().RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		return SetExperimentConfigPoliciesTx(ctx, &tx, experimentTCP)
	})
}

// SetExperimentConfigPoliciesTx adds the experiment invariant config and constraints config
// policies to the database.
func SetExperimentConfigPoliciesTx(ctx context.Context, tx *bun.Tx,
	expTCPs *model.ExperimentTaskConfigPolicies,
) error {
	// Validate experiment invariant config and constraints.
	expInvariantConfig, err := json.Marshal(expTCPs.InvariantConfig)
	if err != nil {
		return errors.Wrapf(err, "error marshaling experiment invariant config %v",
			expInvariantConfig)
	}

	invariantConfig := string(expInvariantConfig)
	_, err = Bun().NewRaw(
		`
		INSERT INTO task_config_policies (workspace_id, last_updated_by, last_updated_time, 
			workload_type, invariant_config, constraints) VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT ON CONSTRAINT pk_wksp_id_wkld_type DO UPDATE 
			SET last_updated_by = ?, last_updated_time = ?, invariant_config = ?, constraints = ?
		`, expTCPs.WorkspaceID, expTCPs.LastUpdatedBy, expTCPs.LastUpdatedTime,
		expTCPs.WorkloadType, invariantConfig, expTCPs.Constraints, expTCPs.LastUpdatedBy,
		expTCPs.LastUpdatedTime, invariantConfig, expTCPs.Constraints).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("error setting experiment task config policies: %w", err)
	}

	return nil
}

// SetNTSCConfigPolicies adds the NTSC invariant config and constraints config policies to
// the database.
func SetNTSCConfigPolicies(ctx context.Context,
	experimentTCP *model.NTSCTaskConfigPolicies,
) error {
	return Bun().RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		return SetNTSCConfigPoliciesTx(ctx, &tx, experimentTCP)
	})
}

// SetNTSCConfigPoliciesTx adds the NTSC invariant config and constraints config policies to
// the database.
func SetNTSCConfigPoliciesTx(ctx context.Context, tx *bun.Tx,
	ntscTCPs *model.NTSCTaskConfigPolicies,
) error {
	_, err := Bun().NewRaw(
		`
		INSERT INTO task_config_policies (workspace_id, last_updated_by, last_updated_time, 
			workload_type, invariant_config, constraints) VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT ON CONSTRAINT pk_wksp_id_wkld_type DO UPDATE 
			SET last_updated_by = ?, last_updated_time = ?, invariant_config = ?, constraints = ?
		`, ntscTCPs.WorkspaceID, ntscTCPs.LastUpdatedBy, ntscTCPs.LastUpdatedTime,
		ntscTCPs.WorkloadType, ntscTCPs.InvariantConfig, ntscTCPs.Constraints, ntscTCPs.LastUpdatedBy,
		ntscTCPs.LastUpdatedTime, ntscTCPs.InvariantConfig, ntscTCPs.Constraints).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("error setting NTSC task config policies: %w", err)
	}

	return nil
}

// GetExperimentConfigPolicies retrieves the invariant experiment config and constraints for the
// given scope (global or workspace-level).
func GetExperimentConfigPolicies(ctx context.Context,
	scope int,
) (*model.ExperimentTaskConfigPolicies, error) {
	var experimentTCP model.ExperimentTaskConfigPolicies
	err := Bun().NewSelect().
		Model(&experimentTCP).
		Where("workspace_id = ? AND workload_type = ?", scope, model.ExperimentType).
		Scan(ctx)
	if err != nil {
		if scope == 0 {
			return nil, fmt.Errorf("error retrieving global experiment task config "+
				"policies: %w", err)
		}
		return nil, fmt.Errorf("error retrieving experiment task config policies for "+
			"workspace with ID %d: %w", scope, err)
	}
	return &experimentTCP, nil
}

// GetNTSCConfigPolicies retrieves the invariant NTSC config and constraints for the
// given scope (global or workspace-level).
func GetNTSCConfigPolicies(ctx context.Context,
	scope int,
) (*model.NTSCTaskConfigPolicies, error) {
	var ntscTCP model.NTSCTaskConfigPolicies
	err := Bun().NewSelect().
		Model(&ntscTCP).
		Where("workspace_id = ? AND workload_type = ?", scope, model.NTSCType).
		Scan(ctx)
	if err != nil {
		if scope == 0 {
			return nil, fmt.Errorf("error retrieving global NTSC task config "+
				"policies: %w", err)
		}
		return nil, fmt.Errorf("error retrieving NTSC task config policies for "+
			"workspace with ID %d: %w", scope, err)
	}
	return &ntscTCP, nil
}

// DeleteConfigPolicies deletes the invariant experiment config and constraints for the
// given scope (global or workspace-level) and workload type.
func DeleteConfigPolicies(ctx context.Context,
	scope int, workloadType model.WorkloadType,
) error {
	if workloadType == model.UnknownType {
		return status.Error(codes.InvalidArgument,
			"invalid workload type for config policies: "+workloadType.String())
	}
	_, err := Bun().NewDelete().
		Table("task_config_policies").
		Where("workspace_id = ? AND workload_type = ?", scope, workloadType.String()).
		Exec(ctx)
	if err != nil {
		if scope == 0 {
			return fmt.Errorf("error deleting global %s config policies:%w",
				strings.ToLower(workloadType.String()), err)
		}
		return fmt.Errorf("error deleting %s config policies for workspace with ID %d: %w",
			strings.ToLower(workloadType.String()), scope, err)
	}
	return nil
}
