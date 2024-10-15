package configpolicy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/ghodss/yaml"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/protoutils"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
)

const (
	// GlobalConfigConflictErr is the error reported when an invariant config has a conflict
	// with a value already set in the global config.
	GlobalConfigConflictErr = "conflict between global and workspace config policy"
	// InvalidExperimentConfigPolicyErr is the error reported by an invalid experiment config policy.
	InvalidExperimentConfigPolicyErr = "invalid experiment config policy"
	// InvalidNTSCConfigPolicyErr is the error reported by an invalid NTSC config policy.
	InvalidNTSCConfigPolicyErr = "invalid ntsc config policy"
)

// ConfigPolicyWarning logs a warning for the configuration policy component.
func ConfigPolicyWarning(msg string) {
	logrus.WithField("component", "task configuration & constraints policy").Warn(msg)
}

// ValidWorkloadType checks if the string is an accepted WorkloadType.
func ValidWorkloadType(val string) bool {
	switch val {
	case model.ExperimentType, model.NTSCType:
		return true
	default:
		return false
	}
}

// UnmarshalConfigPolicy is a generic helper function to unmarshal both JSON and YAML strings.
func UnmarshalConfigPolicy[T any](str string, errString string) (*T, error) {
	var configPolicy T
	dec := json.NewDecoder(bytes.NewReader([]byte(str)))
	dec.DisallowUnknownFields()

	var err error
	// Attempt to decode JSON.
	if err = dec.Decode(&configPolicy); err == nil {
		return &configPolicy, nil
	}

	// Attempt to decode YAML if JSON fails.
	if err = yaml.Unmarshal([]byte(str), &configPolicy, yaml.DisallowUnknownFields); err == nil {
		return &configPolicy, nil
	}

	// Return error if both JSON and YAML parsing fail.
	return nil, fmt.Errorf("%s: %w", errString, err)
}

// MarshalConfigPolicy packs a config policy into a proto struct.
func MarshalConfigPolicy(configPolicy interface{}) *structpb.Struct {
	return protoutils.ToStruct(configPolicy)
}

// ValidateExperimentConfig validates a model.ExperimentType config & constraints.
func ValidateExperimentConfig(
	globalConfigPolicies *model.TaskConfigPolicies,
	configPolicies string,
	priorityEnabledErr error,
) error {
	cp, err := UnmarshalConfigPolicy[ExperimentConfigPolicies](configPolicies, InvalidExperimentConfigPolicyErr)
	if err != nil || cp == nil {
		return err
	}

	if globalConfigPolicies != nil {
		checkAgainstGlobalConfig[model.Constraints](globalConfigPolicies.Constraints, cp.Constraints, "invalid constraints")
		checkAgainstGlobalConfig[expconf.ExperimentConfig](
			globalConfigPolicies.InvariantConfig, cp.InvariantConfig, InvalidExperimentConfigPolicyErr,
		)
	}

	if cp.Constraints != nil {
		checkAgainstGlobalPriority(priorityEnabledErr, cp.Constraints.PriorityLimit)
	}

	if cp.InvariantConfig != nil {
		if cp.InvariantConfig.RawResources != nil {
			checkAgainstGlobalPriority(priorityEnabledErr, cp.InvariantConfig.RawResources.RawPriority)
			if err := checkConstraintConflicts(cp.Constraints, cp.InvariantConfig.RawResources.RawMaxSlots,
				cp.InvariantConfig.RawResources.RawSlotsPerTrial, cp.InvariantConfig.RawResources.RawPriority); err != nil {
				return status.Errorf(codes.InvalidArgument, fmt.Sprintf(InvalidExperimentConfigPolicyErr+": %s.", err))
			}
		}
	}

	return nil
}

// ValidateNTSCConfig validates a model.NTSCType config & constraints.
func ValidateNTSCConfig(
	globalConfigPolicies *model.TaskConfigPolicies,
	configPolicies string,
	priorityEnabledErr error,
) error {
	cp, err := UnmarshalConfigPolicy[NTSCConfigPolicies](configPolicies, InvalidNTSCConfigPolicyErr)
	if err != nil || cp == nil {
		return err // Handle error for nil cp or unmarshalling error.
	}
	if cp.InvariantConfig != nil {
		return fmt.Errorf("not supported: invariant config policies for tasks is not yet supported; please remove `invariant_config` section and try again")
	}
	if globalConfigPolicies != nil {
		checkAgainstGlobalConfig[model.Constraints](globalConfigPolicies.Constraints, cp.Constraints, "invalid constraints")
		checkAgainstGlobalConfig[model.CommandConfig](
			globalConfigPolicies.InvariantConfig, cp.InvariantConfig, InvalidNTSCConfigPolicyErr,
		)
	}

	if cp.Constraints != nil {
		checkAgainstGlobalPriority(priorityEnabledErr, cp.Constraints.PriorityLimit)
	}

	if cp.InvariantConfig != nil {
		if cp.InvariantConfig.Resources.Priority != nil {
			checkAgainstGlobalPriority(priorityEnabledErr, cp.InvariantConfig.Resources.Priority)
		}

		var slots *int
		if cp.InvariantConfig.Resources.Slots != 0 {
			slots = &cp.InvariantConfig.Resources.Slots
		}

		if err := checkConstraintConflicts(cp.Constraints, cp.InvariantConfig.Resources.MaxSlots,
			slots, cp.InvariantConfig.Resources.Priority); err != nil {
			return status.Errorf(codes.InvalidArgument, fmt.Sprintf(InvalidNTSCConfigPolicyErr+": %s.", err))
		}
	}

	return err
}

func checkAgainstGlobalPriority(priorityEnabledErr error, taskPriority *int) {
	if taskPriority != nil && priorityEnabledErr != nil {
		ConfigPolicyWarning("task priority is not supported in this cluster")
	}
}

func checkConstraintConflicts(constraints *model.Constraints, maxSlots, slots, priority *int) error {
	if constraints == nil {
		return nil
	}
	if priority != nil && constraints.PriorityLimit != nil {
		if *constraints.PriorityLimit != *priority {
			return fmt.Errorf("invariant config & constraints are trying to set the priority limit")
		}
	}
	if maxSlots != nil && constraints.ResourceConstraints != nil && constraints.ResourceConstraints.MaxSlots != nil {
		if *constraints.ResourceConstraints.MaxSlots != *maxSlots {
			return fmt.Errorf("invariant config & constraints are trying to set the max slots")
		}
	}
	if slots != nil && constraints.ResourceConstraints != nil && constraints.ResourceConstraints.MaxSlots != nil {
		if *constraints.ResourceConstraints.MaxSlots < *slots {
			return fmt.Errorf("invariant config & constraints are attempting to set an invalid max slot: %v vs %v",
				*constraints.ResourceConstraints.MaxSlots, *slots)
		}
	}

	return nil
}

// checkAgainstGlobalConfig is a generic to check constraints & invariant configs against the global config.
func checkAgainstGlobalConfig[T any](
	globalConfigPolicies *string,
	config *T,
	errorMsg string,
) {
	if globalConfigPolicies != nil && config != nil {
		global, err := UnmarshalConfigPolicy[T](*globalConfigPolicies, errorMsg)
		if err != nil {
			ConfigPolicyWarning(err.Error())
			return
		}
		configPolicyConflict(global, config)
	}
}

// configPolicyConflict compares two different configurations and
// returns an error if both try to define the same field.
func configPolicyConflict(config1, config2 interface{}) {
	v1 := reflect.ValueOf(config1)
	v2 := reflect.ValueOf(config2)

	// If the values are pointers, dereference them
	if v1.Kind() == reflect.Ptr {
		v1 = v1.Elem()
	}
	if v2.Kind() == reflect.Ptr {
		v2 = v2.Elem()
	}

	// Check if both values are valid structs
	if v1.Kind() != reflect.Struct || v2.Kind() != reflect.Struct {
		ConfigPolicyWarning("both inputs must be structs")
		return
	}

	// Iterate over the fields in the struct
	for i := 0; i < v1.NumField(); i++ {
		field1 := v1.Field(i)
		field2 := v2.Field(i)

		if field1.IsValid() && field2.IsValid() && !field1.IsZero() && !field2.IsZero() {
			// For non-pointer fields, compare directly if both are non-zero
			if !reflect.DeepEqual(field1.Interface(), field2.Interface()) {
				ConfigPolicyWarning(fmt.Sprintf("%s: field=%s", GlobalConfigConflictErr, v1.Type().Field(i).Name))
				return
			}
		}
	}
}
