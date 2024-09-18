package configpolicy

import (
	"context"
	"fmt"

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
