package configpolicy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/ghodss/yaml"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/protoutils"
)

// ValidWorkloadType checks if the string is an accepted WorkloadType.
func ValidWorkloadType(val string) bool {
	switch val {
	case model.ExperimentType, model.NTSCType:
		return true
	default:
		return false
	}
}

// GetPriorityLimitPrecedence retrieves the priority limit using order of precedence
func GetPriorityLimitPrecedence(ctx context.Context, workspace_id int, workload_type string) (limit int, found bool, err error) {
	// highest precedence: get global limit
	if limit, found, err = GetPriorityLimit(ctx, nil, workload_type); found {
		return limit, found, err
	}

	// second precedence: get workspace limit
	if limit, found, err = GetPriorityLimit(ctx, &workspace_id, workload_type); found {
		return limit, found, err
	}

	// default
	return 0, false, nil
}

// PriorityOK returns true if the current priority is acceptable given the priority limit and resource manager.
func PriorityOK(currPriority int, priorityLimit int, smallerValueIsHigherPriority bool) bool {

	if smallerValueIsHigherPriority {
		return currPriority >= priorityLimit
	}

	return currPriority <= priorityLimit
}

// UnmarshalExperimentConfigPolicy unpacks a string into ExperimentConfigPolicy struct.
func UnmarshalExperimentConfigPolicy(str string) (*ExperimentConfigPolicies, error) {
	var expConfigPolicy ExperimentConfigPolicies
	var err error

	dec := json.NewDecoder(bytes.NewReader([]byte(str)))
	dec.DisallowUnknownFields()
	if err = dec.Decode(&expConfigPolicy); err == nil {
		// valid JSON input
		return &expConfigPolicy, nil
	}

	if err = yaml.Unmarshal([]byte(str), &expConfigPolicy, yaml.DisallowUnknownFields); err == nil {
		// valid Yaml input
		return &expConfigPolicy, nil
	}
	// invalid JSON & invalid Yaml input
	return nil, fmt.Errorf("invalid ExperimentConfigPolicy input: %w", err)
}

// UnmarshalNTSCConfigPolicy unpacks a string into NTSCConfigPolicy struct.
func UnmarshalNTSCConfigPolicy(str string) (*NTSCConfigPolicies, error) {
	var ntscConfigPolicy NTSCConfigPolicies
	var err error

	dec := json.NewDecoder(bytes.NewReader([]byte(str)))
	dec.DisallowUnknownFields()
	if err = dec.Decode(&ntscConfigPolicy); err == nil {
		// valid JSON input
		return &ntscConfigPolicy, nil
	}

	if err = yaml.Unmarshal([]byte(str), &ntscConfigPolicy, yaml.DisallowUnknownFields); err == nil {
		// valid Yaml input
		return &ntscConfigPolicy, nil
	}
	// invalid JSON & Yaml input
	return nil, fmt.Errorf("invalid ExperimentConfigPolicy input: %w", err)
}

// MarshalConfigPolicy packs a config policy into a proto struct.
func MarshalConfigPolicy(configPolicy interface{}) *structpb.Struct {
	return protoutils.ToStruct(configPolicy)
}
