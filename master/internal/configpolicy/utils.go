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
	// NotSupportedConfigPolicyErr is the error reported when admins attempt to set NTSC invariant config.
	NotSupportedConfigPolicyErr = "not supported"
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

// UnmarshalConfigPolicies unmarshals optionally specified invariant config and constraint
// configurations presented as YAML or JSON strings.
func UnmarshalConfigPolicies[T any](errMsg string, constraintsStr,
	configStr *string) (*model.Constraints, *T,
	error,
) {
	var constraints *model.Constraints
	var config *T

	if constraintsStr != nil {
		unmarshaledConstraints, err := UnmarshalConfigPolicy[model.Constraints](
			*constraintsStr,
			errMsg,
		)
		if err != nil {
			ConfigPolicyWarning(err.Error())
			return nil, nil, err
		}
		constraints = unmarshaledConstraints
	}

	if configStr != nil {
		unmarshaledConfig, err := UnmarshalConfigPolicy[T](
			*configStr,
			errMsg,
		)
		if err != nil {
			ConfigPolicyWarning(err.Error())
			return nil, nil, err
		}
		config = unmarshaledConfig
	}
	return constraints, config, nil
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

	// Warn the user when fields specified in workspace config policies overlap with global config
	// policies (since these fields will be overridden by the respective fields in the global
	// policies).
	var globalConstraints *model.Constraints
	var globalConfig *expconf.ExperimentConfig
	if globalConfigPolicies != nil {
		globalConstraints, globalConfig, err = UnmarshalConfigPolicies[expconf.ExperimentConfig](
			InvalidExperimentConfigPolicyErr,
			globalConfigPolicies.Constraints,
			globalConfigPolicies.InvariantConfig)
		if err != nil {
			return err
		}

		configPolicyOverlap(globalConstraints, cp.Constraints)
		configPolicyOverlap(globalConfig, cp.InvariantConfig)
	}

	if cp.Constraints != nil {
		checkAgainstGlobalPriority(priorityEnabledErr, cp.Constraints.PriorityLimit)
	}

	if cp.InvariantConfig != nil {
		if cp.InvariantConfig.RawResources != nil {
			checkAgainstGlobalPriority(priorityEnabledErr, cp.InvariantConfig.RawResources.RawPriority)

			// Verify the workspace invariant config doesn't conflict with workspace constraints.
			if err := checkConstraintConflicts(cp.Constraints, cp.InvariantConfig.RawResources.RawMaxSlots,
				cp.InvariantConfig.RawResources.RawSlotsPerTrial, cp.InvariantConfig.RawResources.RawPriority); err != nil {
				return status.Errorf(codes.InvalidArgument, fmt.Sprintf(InvalidExperimentConfigPolicyErr+": %s.", err))
			}

			// Verify the workspace invariant config doesn't conflict with global constraints.
			if err := checkConstraintConflicts(globalConstraints, cp.InvariantConfig.RawResources.RawMaxSlots,
				cp.InvariantConfig.RawResources.RawSlotsPerTrial, cp.InvariantConfig.RawResources.RawPriority); err != nil {
				return status.Errorf(codes.InvalidArgument, fmt.Sprintf(InvalidExperimentConfigPolicyErr+
					": workspace invariant_config conflicts with global constraints: %s.", err))
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
		msg := `invariant config policies for tasks is not yet supported, 
		  please remove "invariant_config" section and try again`
		return status.Errorf(codes.InvalidArgument, fmt.Sprintf(NotSupportedConfigPolicyErr+": %s.", msg))
	}

	// Warn the user when fields specified in workspace config policies overlap with global config
	// policies (since these fields will be overridden by the respective fields in the global
	// policies).
	var globalConstraints *model.Constraints
	var globalConfig *model.CommandConfig
	if globalConfigPolicies != nil {
		if globalConfigPolicies.Constraints != nil {
			globalConstraints, globalConfig, err = UnmarshalConfigPolicies[model.CommandConfig](
				InvalidNTSCConfigPolicyErr,
				globalConfigPolicies.Constraints,
				globalConfigPolicies.InvariantConfig)
			if err != nil {
				return err
			}
		}

		configPolicyOverlap(globalConstraints, cp.Constraints)
		configPolicyOverlap(globalConfig, cp.InvariantConfig)
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

		// Verify the workspace invariant config doesn't conflict with workspace constraints.
		if err := checkConstraintConflicts(cp.Constraints, cp.InvariantConfig.Resources.MaxSlots,
			slots, cp.InvariantConfig.Resources.Priority); err != nil {
			return status.Errorf(codes.InvalidArgument, fmt.Sprintf(InvalidNTSCConfigPolicyErr+": %s.", err))
		}

		// Verify the workspace invariant config conflict with global constraints.
		if err := checkConstraintConflicts(globalConstraints,
			cp.InvariantConfig.Resources.MaxSlots, slots,
			cp.InvariantConfig.Resources.Priority); err != nil {
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
			return fmt.Errorf("invariant config has %v slots per trial. violates constraints max slots of %v",
				*slots, *constraints.ResourceConstraints.MaxSlots)
		}
	}

	return nil
}

// configPolicyOverlap compares two different configurations and warns the user when both
// configurations define the same field.
func configPolicyOverlap(config1, config2 interface{}) {
	if reflect.ValueOf(config1).Type() != reflect.ValueOf(config2).Type() &&
		reflect.ValueOf(config1).Type() != reflect.ValueOf(&model.Constraints{}).Type() &&
		reflect.ValueOf(config1).Type() != reflect.ValueOf(&model.CommandConfig{}).Type() &&
		reflect.ValueOf(config1).Type() != reflect.ValueOf(&expconf.ExperimentConfig{}).Type() {
		return
	}

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
