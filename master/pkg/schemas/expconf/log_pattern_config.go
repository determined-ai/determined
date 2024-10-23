package expconf

import (
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas"
	"github.com/determined-ai/determined/master/pkg/set"
)

// LogPoliciesConfigV0 is a list of log policies.
//
//go:generate ../gen.sh
type LogPoliciesConfigV0 []LogPolicyV0

// WithDefaults implements the Defaultable pseudo-interface.
func (b LogPoliciesConfigV0) WithDefaults() LogPoliciesConfigV0 {
	defaultPolicies := LogPoliciesConfigV0{
		LogPolicyV0{
			RawName:    ptrs.Ptr(CUDAOOM),
			RawPattern: ptrs.Ptr(CUDAOOMPattern),
		},
		LogPolicyV0{
			RawName:    ptrs.Ptr(ECCError),
			RawPattern: ptrs.Ptr(ECCErrorPattern),
		},
	}

	return schemas.Merge(b, defaultPolicies)
}

// Merge implements the Mergable pseudo-interface.
// Union merge log policies.
// Special cases:
// 1. We may see policies with different names same patterns. We keep both of them, but which name
// will be shown in the UI is undefined.
// 2. There could be policies with the same name different patterns. We save the one with the higher priority.
// Unsetting default values depends on this behavior.
// 2.1 Polices that don't have name but have same patterns is a special case. These are legacy
// policies.
func (b LogPoliciesConfigV0) Merge(
	src LogPoliciesConfigV0,
) LogPoliciesConfigV0 {
	if src == nil && b == nil {
		return nil
	}

	// Keep everything in b unconditionally.
	out := append(LogPoliciesConfigV0{}, b...)

	names := set.New[string]()
	unnamedPolicies := set.New[string]()
	for _, lp := range b {
		if lp.RawName == nil {
			// Not checking nil because we've enforced action and pattern must be set for the legacy policy
			// in the json schema.
			s := fmt.Sprintf("%v:%v", *lp.RawAction, *lp.RawPattern)
			unnamedPolicies.Insert(s)
		} else {
			names.Insert(*lp.RawName)
		}
	}

	// Add policies in src that don't exist in b.
	for _, lp := range src {
		if lp.RawName == nil {
			s := fmt.Sprintf("%v:%v", *lp.RawAction, *lp.RawPattern)
			if !unnamedPolicies.Contains(s) {
				out = append(out, lp)
				unnamedPolicies.Insert(s)
			}
		} else if !names.Contains(*lp.RawName) {
			out = append(out, lp)
			names.Insert(*lp.RawName)
		}
	}

	return out
}

// UnmarshalJSON implements the json.Unmarshaler interface.
// Special cases:
// 1. User may submit policies that have different names but same patterns. We keep both of them, but which name
// will be shown in the UI is undefined.
// 2. User can't submit policies that have the same names.
func (b *LogPoliciesConfigV0) UnmarshalJSON(data []byte) error {
	type DefaultParser LogPoliciesConfigV0
	var jsonItems DefaultParser
	if err := json.Unmarshal(data, &jsonItems); err != nil {
		return errors.Wrapf(err, "failed to parse runtime items")
	}

	// Detect policies that have the same names but different patterns
	names := set.New[string]()
	for _, jsonItem := range jsonItems {
		n := jsonItem.RawName
		if n == nil {
			continue
		}

		if names.Contains(*n) {
			return fmt.Errorf("log_policies have duplicated names %q", *n)
		}
		names.Insert(*n)
	}

	*b = LogPoliciesConfig(jsonItems)
	return nil
}

// LogPolicyV0 is an action to take if we match against trial logs.
//
//go:generate ../gen.sh
type LogPolicyV0 struct {
	// Legacy log policy doesn't have a name. Legacy log policy will be deprecated.
	RawName *string `json:"name,omitempty"`
	// Pattern can be nil. So user can override it to disable the default log polices.
	RawPattern *string      `json:"pattern,omitempty"`
	RawAction  *LogActionV0 `json:"action,omitempty"`
}

// LogActionType is the type of an action.
type LogActionType string

// LogActionType refers to the action user can take when a pattern is detected in the log.
const (
	LogActionTypeCancelRetries LogActionType = "cancel_retries"
	LogActionTypeExcludeNode   LogActionType = "exclude_node"
)

// LogActionV0 is a policy to take after matching.
type LogActionV0 struct {
	Type LogActionType
}

// UnmarshalJSON implements the json.Unmarshaler interface.
// It applies a shim to the legacy actions. For example we have a legacy action:
//
//	action:
//	  type: cancel_retries
//
// It will become:
//
//	action: cancel_retries
func (s *LogActionV0) UnmarshalJSON(data []byte) error {
	// First, check if we can unmarshal the log policy item as a legacy item.
	type LegacyAction struct {
		Type LogActionType `json:"type"`
	}

	var legacy LegacyAction
	err := json.Unmarshal(data, &legacy)
	if err == nil {
		// Apply shim to bring legacy policy into the current format.
		switch legacy.Type {
		case LogActionTypeCancelRetries, LogActionTypeExcludeNode:
			*s = LogActionV0(legacy)
		default:
			return fmt.Errorf("unrecognized legacy action type: %s, data: %q", legacy.Type, string(data))
		}
		return nil
	}

	// It is not a legacy item. Try to unmarshal it as a modern item.
	var lat LogActionType
	if err := json.Unmarshal(data, &lat); err == nil {
		switch lat {
		case LogActionTypeCancelRetries, LogActionTypeExcludeNode:
			*s = LogActionV0{Type: lat}
			return nil
		}
	}

	return fmt.Errorf("failed to unmarshal log action: %w, data: %q", err, string(data))
}

// MarshalJSON implements the json.Marshaler interface.
func (s LogActionV0) MarshalJSON() ([]byte, error) {
	if s.Type == LogActionTypeCancelRetries || s.Type == LogActionTypeExcludeNode {
		return json.Marshal(s.Type)
	}
	return nil, fmt.Errorf("failed to marshal LogActionV0: %+v", s)
}
