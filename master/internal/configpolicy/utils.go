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
	var expConfigPolicy ExperimentConfigPolicies
	var err error

	dec := json.NewDecoder(bytes.NewReader([]byte(str)))
	dec.DisallowUnknownFields()
	if err = dec.Decode(&expConfigPolicy); err == nil {
		// valid JSON input
		if reflect.DeepEqual(expConfigPolicy, ExperimentConfigPolicies{}) {
			return nil, fmt.Errorf(EmptyInvariantConfigErr)
		}
		return &expConfigPolicy, nil
	}

	if err = yaml.Unmarshal([]byte(str), &expConfigPolicy, yaml.DisallowUnknownFields); err == nil {
		// valid Yaml input
		if reflect.DeepEqual(expConfigPolicy, ExperimentConfigPolicies{}) {
			return nil, fmt.Errorf(EmptyInvariantConfigErr)
		}
		return &expConfigPolicy, nil
	}

	// invalid JSON & invalid Yaml input
	return nil, fmt.Errorf("invalid experiment config policy: %w", err)
}

// UnmarshalNTSCConfigPolicy unpacks a string into NTSCConfigPolicy struct.
func UnmarshalNTSCConfigPolicy(str string) (*NTSCConfigPolicies, error) {
	var ntscConfigPolicy NTSCConfigPolicies
	var err error

	dec := json.NewDecoder(bytes.NewReader([]byte(str)))
	dec.DisallowUnknownFields()
	if err = dec.Decode(&ntscConfigPolicy); err == nil {
		// valid JSON input
		if reflect.DeepEqual(ntscConfigPolicy, NTSCConfigPolicies{}) {
			return nil, fmt.Errorf(EmptyInvariantConfigErr)
		}
		return &ntscConfigPolicy, nil
	}

	if err = yaml.Unmarshal([]byte(str), &ntscConfigPolicy, yaml.DisallowUnknownFields); err == nil {
		// valid Yaml input
		if reflect.DeepEqual(ntscConfigPolicy, NTSCConfigPolicies{}) {
			return nil, fmt.Errorf(EmptyInvariantConfigErr)
		}
		return &ntscConfigPolicy, nil
	}

	// invalid JSON & Yaml input
	return nil, fmt.Errorf("invalid NTSC config policy: %w", err)
}

// MarshalConfigPolicy packs a config policy into a proto struct.
func MarshalConfigPolicy(configPolicy interface{}) *structpb.Struct {
	return protoutils.ToStruct(configPolicy)
}
