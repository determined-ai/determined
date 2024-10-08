package configpolicy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/ghodss/yaml"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/protoutils"
)

const (
	// EmptyInvariantConfigErr is the error reported when an invariant config is specified with no
	// fields.
	EmptyInvariantConfigErr = "empty invariant config"
	// EmptyConstraintsErr is the error reported when a constraints policy is specified with no
	// fields.
	EmptyConstraintsErr = "empty constraints policy"
	// GlobalConfigConflictErr is the error reported when an invariant config has a conflict
	// with a value already set in the global config.
	GlobalConfigConflictErr = "conflict between global and task config policy"
	// InvalidExperimentConfigPolicyErr is the error reported by an invalid experiment config policy.
	InvalidExperimentConfigPolicyErr = "invalid experiment config policy"
	// InvalidNTSCConfigPolicyErr is the error reported by an invalid NTSC config policy.
	InvalidNTSCConfigPolicyErr = "invalid NTSC config policy"
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

// UnmarshalExperimentConfigPolicy unpacks a string into ExperimentConfigPolicy struct.
func UnmarshalExperimentConfigPolicy(str string) (*ExperimentConfigPolicies, error) {
	return UnmarshalConfigPolicy[ExperimentConfigPolicies](str, InvalidExperimentConfigPolicyErr)
}

// UnmarshalNTSCConfigPolicy unpacks a string into NTSCConfigPolicy struct.
func UnmarshalNTSCConfigPolicy(str string) (*NTSCConfigPolicies, error) {
	return UnmarshalConfigPolicy[NTSCConfigPolicies](str, InvalidNTSCConfigPolicyErr)
}

// UnmarshalConfigPolicy is a generic helper function to unmarshal both JSON and YAML strings.
func UnmarshalConfigPolicy[T any](str string, errString string) (*T, error) {
	var configPolicy T
	dec := json.NewDecoder(bytes.NewReader([]byte(str)))
	dec.DisallowUnknownFields()

	var err error
	// Attempt to decode JSON.
	if err = dec.Decode(&configPolicy); err == nil {
		// valid JSON input
		if reflect.ValueOf(configPolicy).IsZero() {
			return nil, fmt.Errorf(EmptyInvariantConfigErr)
		}
		return &configPolicy, nil
	}

	// Attempt to decode YAML if JSON fails.
	if err = yaml.Unmarshal([]byte(str), &configPolicy, yaml.DisallowUnknownFields); err == nil {
		// valid YAML input
		if reflect.ValueOf(configPolicy).IsZero() {
			return nil, fmt.Errorf(EmptyInvariantConfigErr)
		}
		return &configPolicy, nil
	}

	// Return error if both JSON and YAML parsing fail.
	return nil, fmt.Errorf("%s: %w", errString, err)
}

// MarshalConfigPolicy packs a config policy into a proto struct.
func MarshalConfigPolicy(configPolicy interface{}) *structpb.Struct {
	return protoutils.ToStruct(configPolicy)
}

// HaveAtLeastOneSharedDefinedField compares two different configurations and
// returns an error if both try to define the same field.
func HaveAtLeastOneSharedDefinedField(config1, config2 interface{}) error {
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
		return fmt.Errorf("both inputs must be structs")
	}

	// Iterate over the fields in the struct
	for i := 0; i < v1.NumField(); i++ {
		field1 := v1.Field(i)
		field2 := v2.Field(i)

		if field1.IsValid() && field2.IsValid() && !field1.IsZero() && !field2.IsZero() {
			// For non-pointer fields, compare directly if both are non-zero
			if !reflect.DeepEqual(field1.Interface(), field2.Interface()) {
				return fmt.Errorf("%s: %v, %v", GlobalConfigConflictErr, field1.Interface(), field2.Interface())
			}
		}
	}

	// Configs are equal in shared non-null fields, or don't share any non-null fields
	return nil
}
