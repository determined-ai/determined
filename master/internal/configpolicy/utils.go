package configpolicy

import (
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
	case string(model.ExperimentType), string(model.NTSCType):
		return true
	default:
		return false
	}
}

// UnmarshalExperimentConfigPolicy unpacks a string into ExperimentConfigPolicy struct.
func UnmarshalExperimentConfigPolicy(str string) (*ExperimentConfigPolicy, error) {
	var expConfigPolicy ExperimentConfigPolicy
	var err error

	if err = json.Unmarshal([]byte(str), &expConfigPolicy); err == nil {
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
func UnmarshalNTSCConfigPolicy(str string) (*NTSCConfigPolicy, error) {
	var ntscConfigPolicy NTSCConfigPolicy
	var err error

	if err = json.Unmarshal([]byte(str), &ntscConfigPolicy); err == nil {
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
