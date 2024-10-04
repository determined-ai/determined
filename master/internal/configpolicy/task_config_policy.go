package configpolicy

import (
	"context"
	"encoding/json"
	"errors"
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

var (
	errPriorityConstraintFailure = errors.New("submitted workload failed priority constraint")
	errResourceConstraintFailure = errors.New("submitted workload failed a resource constraint")
)

// CheckNTSCConstraints returns an error if the NTSC config fails constraint checks.
func CheckNTSCConstraints(
	ctx context.Context,
	workspaceID int,
	workloadConfig model.CommandConfig,
	resourceManager rm.ResourceManager,
) error {
	constraints, err := GetMergedConstraints(ctx, workspaceID, model.NTSCType)
	if err != nil {
		return err
	}

	if constraints.ResourceConstraints != nil && constraints.ResourceConstraints.MaxSlots != nil {
		if err = checkSlotsConstraint(*constraints.ResourceConstraints.MaxSlots, workloadConfig.Resources.Slots, workloadConfig.Resources.MaxSlots); err != nil {
			return err
		}
	}

	// For each submitted constraint, check if the workload config is within allowed values.
	// rm.SmallerValueIsHigherPriority only returns an error if task priority is not implemented for that resource manager.
	// In that case, there is no need to check if requested priority is within limits.
	smallerHigher, err := resourceManager.SmallerValueIsHigherPriority()
	if err != nil {
		if err = checkPriorityConstraint(smallerHigher, constraints.PriorityLimit, workloadConfig.Resources.Priority); err != nil {
			return err
		}
	}

	return nil
}

// CheckExperimentConstraints returns an error if the NTSC config fails constraint checks.
func CheckExperimentConstraints(
	ctx context.Context,
	workspaceID int,
	workloadConfig expconf.ExperimentConfigV0,
	resourceManager rm.ResourceManager,
) error {
	constraints, err := GetMergedConstraints(ctx, workspaceID, model.ExperimentType)
	if err != nil {
		return err
	}

	if constraints.ResourceConstraints != nil && constraints.ResourceConstraints.MaxSlots != nil {
		// users cannot specify number of slots for an experiment
		slotsRequest := 0
		if err = checkSlotsConstraint(*constraints.ResourceConstraints.MaxSlots, slotsRequest, workloadConfig.Resources().MaxSlots()); err != nil {
			return err
		}
	}

	// For each submitted constraint, check if the workload config is within allowed values.
	// rm.SmallerValueIsHigherPriority only returns an error if task priority is not implemented for that resource manager.
	// In that case, there is no need to check if requested priority is within limits.
	smallerHigher, err := resourceManager.SmallerValueIsHigherPriority()
	if err != nil {
		return nil
	}
	if err = checkPriorityConstraint(smallerHigher, constraints.PriorityLimit, workloadConfig.Resources().Priority()); err != nil {
		return err
	}

	return nil
}

func checkPriorityConstraint(smallerHigher bool, priorityLimit *int, priorityRequest *int) error {
	if priorityLimit == nil || priorityRequest == nil {
		return nil
	}

	if !priorityWithinLimit(*priorityRequest, *priorityLimit, smallerHigher) {
		return fmt.Errorf("requested priority [%d] exceeds limit set by admin [%d]: %w",
			*priorityRequest, *priorityLimit, errPriorityConstraintFailure)
	}
	return nil
}

func checkSlotsConstraint(slotsLimit int, slotsRequest int, maxSlotsRequest *int) error {
	if slotsLimit < slotsRequest {
		return fmt.Errorf("requested resources.slots [%d] exceeds limit set by admin [%d]: %w",
			slotsRequest, slotsLimit, errResourceConstraintFailure)
	}

	if maxSlotsRequest != nil {
		if slotsLimit < *maxSlotsRequest {
			return fmt.Errorf("requested resources.max_slots [%d] exceeds limit set by admin [%d]: %w",
				*maxSlotsRequest, slotsLimit, errResourceConstraintFailure)
		}
	}

	return nil
}

// GetMergedConstraints retrieves Workspace and Global constraints and returns a merged result.
// workloadType is expected to be model.ExperimentType or model.NTSCType.
func GetMergedConstraints(ctx context.Context, workspaceID int, workloadType string) (*model.Constraints, error) {
	// Workspace-level constraints should be over-ridden by global contraints, if set.
	var constraints model.Constraints
	wkspConfigPolicies, err := GetTaskConfigPolicies(ctx, &workspaceID, workloadType)
	if err != nil {
		return nil, err
	}
	if wkspConfigPolicies.Constraints != nil {
		if err = json.Unmarshal([]byte(*wkspConfigPolicies.Constraints), &constraints); err != nil {
			return nil, fmt.Errorf("unable to merge workspace and global constraints: %w", err)
		}
	}

	globalConfigPolicies, err := GetTaskConfigPolicies(ctx, nil, workloadType)
	if err != nil {
		return nil, err
	}
	if globalConfigPolicies.Constraints != nil {
		if err = json.Unmarshal([]byte(*globalConfigPolicies.Constraints), &constraints); err != nil {
			return nil, fmt.Errorf("unable to merge workspace and global constraints: %w", err)
		}
	}

	return &constraints, nil
}

// PriorityAllowed returns true if the desired priority is within the task config policy limit.
func PriorityAllowed(wkspID int, workloadType string, priority int, smallerHigher bool) (bool, error) {
	// Check if a priority limit has been set with a constraint policy.
	// Global policies have highest precedence.
	limit, found, err := GetPriorityLimit(context.TODO(), nil, workloadType)
	if err != nil {
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
