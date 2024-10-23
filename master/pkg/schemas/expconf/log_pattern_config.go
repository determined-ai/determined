package expconf

import (
	"encoding/json"
	"fmt"
	"sort"

	log "github.com/sirupsen/logrus"

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
	// src is nil and b is an empty slice.
	if src == nil && b != nil && len(b) == 0 {
		return make(LogPoliciesConfigV0, 0)
	}
	var out LogPoliciesConfigV0
	var policiesWoName LogPoliciesConfigV0
	seenPolicies := set.New[string]()
	nameToLp := make(map[string]LogPolicyV0)

	for _, lp := range src {
		if lp.RawName == nil {
			policiesWoName.appendLegacyPolicies(lp, seenPolicies)
			continue
		}
		nameToLp[*lp.RawName] = lp
	}

	for _, otherLp := range b {
		if otherLp.RawName == nil {
			policiesWoName.appendLegacyPolicies(otherLp, seenPolicies)
			continue
		}
		name := *otherLp.RawName
		if srcLp, ok := nameToLp[name]; ok {
			// Merge two LogPolicies if they have the same name.
			nameToLp[name] = schemas.Merge(otherLp, srcLp)
		} else {
			nameToLp[name] = otherLp
		}
	}

	for _, lp := range nameToLp {
		out = append(out, lp)
	}
	out = append(out, policiesWoName...)
	out.sort()
	return out
}

// Only keep unique legacy policies.
func (b *LogPoliciesConfigV0) appendLegacyPolicies(policy LogPolicyV0, seenPolicies set.Set[string]) {
	// policy without name is legacy policy.
	if policy.RawName == nil {
		json, err := json.Marshal(policy)
		if err != nil {
			log.Errorf("marshaling error %+v %v", policy, err)
		}
		if seenPolicies.Contains(string(json)) {
			return
		}
		seenPolicies.Insert(string(json))
		*b = append(*b, policy)
	}
}

// UnmarshalJSON implements the json.Unmarshaler interface.
// Special cases:
// 1. User may submit policies that have different names but same patterns. We keep both of them, but which name
// will be shown in the UI is undefined.
// 2. User can't submit policies that have the same names but different patterns.
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
			return fmt.Errorf("log_policies have duplicated names %q but different patterns", *n)
		}
		names.Insert(*n)
	}

	out := LogPoliciesConfig(jsonItems)
	out.sort()
	*b = out

	return nil
}

// Sort LogPoliciesConfigV0 so the output is in deterministic state. Testing will be easier.
func (b LogPoliciesConfigV0) sort() {
	sort.Slice(b, func(i, j int) bool {
		// Sort policies by name first.
		var iName string
		var jName string
		if b[i].RawName != nil {
			iName = *b[i].RawName
		}
		if b[j].RawName != nil {
			jName = *b[j].RawName
		}

		// Sort policies by pattern if their names are the same.
		if iName == jName {
			var iPattern string
			var jPattern string
			if b[i].RawPattern != nil {
				iPattern = *b[i].RawPattern
			}
			if b[j].RawPattern != nil {
				jPattern = *b[j].RawPattern
			}
			// Sort by action if patterns and names are the same.
			if iPattern == jPattern {
				var iAction string
				var jAction string
				if b[i].RawAction != nil {
					iAction = string(b[i].RawAction.Type)
				}
				if b[j].RawAction != nil {
					jAction = string(b[j].RawAction.Type)
				}
				return iAction < jAction
			}

			return iPattern < jPattern
		}

		return iName < jName
	})
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

// Merge LogPolicyV0.
func (l LogPolicyV0) Merge(src LogPolicyV0) LogPolicyV0 {
	// Merging only applies to the LogActionV0 with the same name.
	if src.RawName == nil || l.RawName == nil || *src.RawName != *l.RawName {
		log.Errorf("the names of %+v and %+v are not the same", l, src)
		return src
	}

	return l
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
