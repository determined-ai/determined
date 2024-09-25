package configpolicy

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/determined-ai/determined/master/internal/rm"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
)

// ExperimentConfigPolicies is the invariant config and constraints for an experiment.
// Submitted experiments whose config fields vary from the respective InvariantConfig fields set
// within a given scope are silently overridden.
// Submitted experiments whose constraint fields vary from the respective Constraint fields set
// within a given scope are rejected.
type ExperimentConfigPolicies struct {
	InvariantConfig *expconf.ExperimentConfig `json:"invariant_config"`
	Constraints     *model.Constraints        `json:"constraints"`
}

// NTSCConfigPolicies is the invariant config and constraints for an NTSC task.
// Submitted NTSC tasks whose config fields vary from the respective InvariantConfig fields set
// within a given scope are silently overridden.
// Submitted NTSC tasks whose constraint fields vary from the respective Constraint fields set
// within a given scope are rejected.
type NTSCConfigPolicies struct {
	InvariantConfig *model.CommandConfig `json:"invariant_config"`
	Constraints     *model.Constraints   `json:"constraints"`
}

// ValidateNTSCConstraints returns true if the NTSC config passes constraint checks.
func ValidateNTSCConstraints(
	ctx context.Context,
	workspaceID int,
	workloadConfig model.CommandConfig,
	resourceManager rm.ResourceManager,
) (bool, error) {
	constraints, err := GetMergedConstraints(ctx, workspaceID, model.NTSCType)
	if err != nil {
		return false, err
	}

	// For each submitted constraint, check if the workload config is within allowed values.
	smallerHigher, err := resourceManager.SmallerValueIsHigherPriority()
	if err == nil && constraints.PriorityLimit != nil && workloadConfig.Resources.Priority != nil {
		if !priorityWithinLimit(*workloadConfig.Resources.Priority, *constraints.PriorityLimit, smallerHigher) {
			return false, fmt.Errorf("requested priority [%d] exceeds limit set by admin [%d]",
				*constraints.PriorityLimit, *workloadConfig.Resources.Priority)
		}
	}

	if constraints.ResourceConstraints != nil && constraints.ResourceConstraints.MaxSlots != nil &&
		workloadConfig.Resources.MaxSlots != nil {
		if *constraints.ResourceConstraints.MaxSlots < *workloadConfig.Resources.MaxSlots {
			return false, fmt.Errorf("requested resources.max_slots [%d] exceeds limit set by admin [%d]",
				*constraints.ResourceConstraints.MaxSlots, *workloadConfig.Resources.MaxSlots)
		}
	}

	return true, nil
}

// GetMergedConstraints retrieves Workspace and Global constraints and returns a merged result.
func GetMergedConstraints(ctx context.Context, workspaceID int, workloadType string) (model.Constraints, error) {
	// Workspace-level constraints should be over-ridden by global contraints, if set.
	var constraints model.Constraints
	wkspConfigPolicies, err := GetTaskConfigPolicies(ctx, &workspaceID, workloadType)
	if err != nil {
		return constraints, err
	}
	if wkspConfigPolicies.Constraints != nil {
		if err = json.Unmarshal([]byte(*wkspConfigPolicies.Constraints), &constraints); err != nil {
			return constraints, err
		}
	}

	globalConfigPolicies, err := GetTaskConfigPolicies(ctx, nil, workloadType)
	if err != nil {
		return constraints, err
	}
	if globalConfigPolicies.Constraints != nil {
		if err = json.Unmarshal([]byte(*globalConfigPolicies.Constraints), &constraints); err != nil {
			return constraints, err
		}
	}

	return constraints, nil
}

// PriorityAllowed returns true if the desired priority is within the task config policy limit.
func PriorityAllowed(wkspID int, workloadType string, priority int, smallerHigher bool) (bool, error) {
	// Check if a priority limit has been set in task config policies.
	// Global policies have highest precedence.
	limit, found, err := GetPriorityLimit(context.TODO(), nil, workloadType)
	if err != nil {
		// TODO do we really want to block on this?
		return false, fmt.Errorf("unable to fetch task config policy priority limit")
	}
	if found {
		return priorityWithinLimit(priority, limit, smallerHigher), nil
	}

	// TODO use COALESCE instead once postgres updates are complete.
	// Workspace policies have second precedence.
	limit, found, err = GetPriorityLimit(context.TODO(), &wkspID, workloadType)
	if err != nil {
		// TODO do we really want to block on this?
		return false, fmt.Errorf("unable to fetch task config policy priority limit")
	}
	if found {
		return priorityWithinLimit(priority, limit, smallerHigher), nil
	}

	// No priority limit has been set.
	return true, nil
}

func priorityWithinLimit(userPriority int, adminLimit int, smallerHigher bool) bool {
	if smallerHigher {
		return userPriority >= adminLimit
	}

	return userPriority <= adminLimit
}
