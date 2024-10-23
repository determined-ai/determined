package configpolicy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"github.com/labstack/gommon/log"

	"github.com/determined-ai/determined/master/internal/rm"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/schemas"
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
	errPriorityImmutable         = errors.New("priority cannot be modified")
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
		if err = checkSlotsConstraint(*constraints.ResourceConstraints.MaxSlots, &workloadConfig.Resources.Slots,
			workloadConfig.Resources.MaxSlots); err != nil {
			return err
		}
	}

	// For each submitted constraint, check if the workload config is within allowed values.
	// rm.SmallerValueIsHigherPriority only returns an error if task priority is not implemented for that resource manager.
	// In that case, there is no need to check if requested priority is within limits.
	smallerHigher, err := resourceManager.SmallerValueIsHigherPriority()
	if err == nil {
		if err = checkPriorityConstraint(smallerHigher, constraints.PriorityLimit,
			workloadConfig.Resources.Priority); err != nil {
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
		if workloadConfig.RawResources != nil {
			slotsRequest := workloadConfig.RawResources.RawSlotsPerTrial
			if err = checkSlotsConstraint(*constraints.ResourceConstraints.MaxSlots,
				slotsRequest,
				workloadConfig.Resources().MaxSlots()); err != nil {
				return err
			}
			slotsRequest = workloadConfig.RawResources.RawMaxSlots
			if err = checkSlotsConstraint(*constraints.ResourceConstraints.MaxSlots,
				slotsRequest,
				workloadConfig.Resources().MaxSlots()); err != nil {
				return err
			}
		}
	}

	// For each submitted constraint, check if the workload config is within allowed values.
	// rm.SmallerValueIsHigherPriority only returns an error if task priority is not implemented for that resource manager.
	// In that case, there is no need to check if requested priority is within limits.
	smallerHigher, err := resourceManager.SmallerValueIsHigherPriority()
	if err == nil {
		if err = checkPriorityConstraint(smallerHigher, constraints.PriorityLimit,
			workloadConfig.Resources().Priority()); err != nil {
			return err
		}
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

func checkSlotsConstraint(slotsLimit int, slotsRequest *int, maxSlotsRequest *int) error {
	if slotsRequest != nil {
		if slotsLimit < *slotsRequest {
			return fmt.Errorf("requested resources.slots [%d] exceeds limit set by admin [%d]: %w",
				slotsRequest, slotsLimit, errResourceConstraintFailure)
		}
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

// MergeWithInvariantExperimentConfigs merges the config with workspace and global invariant
// configs, where a global invariant config takes precedence over a workspace-level invariant
// config.
func MergeWithInvariantExperimentConfigs(ctx context.Context, workspaceID int,
	config expconf.ExperimentConfigV0,
) (*expconf.ExperimentConfigV0, error) {
	originalConfig := config
	var wkspOverride, globalOverride bool
	wkspConfigPolicies, err := GetTaskConfigPolicies(ctx, &workspaceID, model.ExperimentType)
	if err != nil {
		return nil, err
	}
	if wkspConfigPolicies.InvariantConfig != nil {
		var tempConfig expconf.ExperimentConfigV0
		if err := json.Unmarshal([]byte(*wkspConfigPolicies.InvariantConfig), &tempConfig); err != nil {
			return nil, fmt.Errorf("error unmarshaling workspace invariant config: %w", err)
		}
		// Merge arrays and maps with those specified in the user-submitted config.
		config = schemas.Merge(tempConfig, config)
		wkspOverride = true
	}

	globalConfigPolicies, err := GetTaskConfigPolicies(ctx, nil, model.ExperimentType)
	if err != nil {
		return nil, err
	}
	if globalConfigPolicies.InvariantConfig != nil {
		var tempConfig expconf.ExperimentConfigV0
		err = json.Unmarshal([]byte(*globalConfigPolicies.InvariantConfig), &tempConfig)
		if err != nil {
			return nil, fmt.Errorf("error unmarshaling global invariant config: %w", err)
		}
		// Merge arrays and maps with those specified in the current (user-submitted combined with
		// optionally set workspace invariant) config.
		config = schemas.Merge(tempConfig, config)
		globalOverride = true
	}

	scope := ""
	if wkspOverride {
		if globalOverride {
			scope += "workspace and global"
		} else {
			scope += "workspace"
		}
	} else if globalOverride {
		scope += "global"
	}

	if !reflect.DeepEqual(originalConfig, config) {
		log.Warnf("some fields were overridden by admin %s config policies", scope)
	}
	return &config, nil
}

// FindAllowedPriority finds the optionally set priority limit in scope's invariant config
// policies. Returns the invariant config priority if that's set, and otherwise returns the
// the priority_limit constraint. If neither of the two is set, returns nil limit.
func FindAllowedPriority(scope *int, workloadType string) (limit *int, constraintExists bool,
	err error) {
	configPolicies, err := GetTaskConfigPolicies(context.TODO(), scope, workloadType)
	if err != nil {
		return nil, false, fmt.Errorf("unable to fetch task config policies: %w", err)
	}

	// Cannot update priority if priority set in invariant config.
	if configPolicies.InvariantConfig != nil {
		switch workloadType {
		case model.NTSCType:
			var configs model.CommandConfig
			err = json.Unmarshal([]byte(*configPolicies.InvariantConfig), &configs)
			if err != nil {
				return nil, false, fmt.Errorf("unable to unmarshal task config policies: %w", err)
			}
			if configs.Resources.Priority != nil {
				adminPriority := *configs.Resources.Priority
				return &adminPriority, false,
					fmt.Errorf("priority set by invariant config: %w", errPriorityImmutable)
			}
		case model.ExperimentType:
			var configs expconf.ExperimentConfigV0
			err = json.Unmarshal([]byte(*configPolicies.InvariantConfig), &configs)
			if err != nil {
				return nil, false, fmt.Errorf("unable to unmarshal task config policies: %w", err)
			}
			if configs.RawResources != nil && configs.RawResources.RawPriority != nil {
				adminPriority := *configs.RawResources.RawPriority
				return &adminPriority, false,
					fmt.Errorf("priority set by invariant config: %w", errPriorityImmutable)
			}
		default:
			return nil, false, fmt.Errorf("workload type %s not supported", workloadType)
		}
	}

	// Find priority constraint, if set.
	var constraints model.Constraints
	if configPolicies.Constraints != nil {
		if err = json.Unmarshal([]byte(*configPolicies.Constraints), &constraints); err != nil {
			return nil, false, fmt.Errorf("unable to unmarshal task config policies: %w", err)
		}
		if constraints.PriorityLimit != nil {
			return constraints.PriorityLimit, true, nil
		}
	}

	return nil, false, nil
}

// PriorityUpdateAllowed returns true if the desired priority is within the task config policy limit.
func PriorityUpdateAllowed(wkspID int, workloadType string, priority int, smallerHigher bool) (bool, error) {
	// Check if a priority limit has been set with a constraint policy.
	// Global policies have highest precedence.
	globalEnforcedPriority, globalExists, err := FindAllowedPriority(nil, workloadType)

	if errors.Is(err, errPriorityImmutable) && globalEnforcedPriority != nil &&
		*globalEnforcedPriority == priority {
		// If task config policies have updated since the workload was originally scheduled, allow users
		// to update the priority to the new priority set by invariant config.
		return true, nil
	}
	if err != nil {
		return false, err
	}

	// TODO use COALESCE instead once postgres updates are complete.
	// Workspace policies have second precedence.
	wkspEnforcedPriority, wkspExists, err := FindAllowedPriority(&wkspID, workloadType)
	if errors.Is(err, errPriorityImmutable) && wkspEnforcedPriority != nil &&
		*wkspEnforcedPriority == priority {
		// If task config policies have updated since the workload was originally scheduled, allow users
		// to update the priority to the new priority set by invariant config.
		return true, nil
	}
	if err != nil {
		return false, err
	}

	// No invariant configs. Check for constraints.
	if globalExists {
		return priorityWithinLimit(priority, *wkspEnforcedPriority, smallerHigher), nil
	}
	if wkspExists {
		return priorityWithinLimit(priority, *wkspEnforcedPriority, smallerHigher), nil
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
