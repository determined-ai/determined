package expconf

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/schemas"
	"github.com/determined-ai/determined/master/pkg/set"
)

// LogPoliciesConfigV0 is a list of log policies.
//
//go:generate ../gen.sh
type LogPoliciesConfigV0 []LogPolicyV0

// Merge implements the Mergable pseudo-interface.
// We appends all LogPolicyV0s to the output slice, but if there are any with the same pattern, we merge
// their actions and save them as one LogPolicyV0.
func (b LogPoliciesConfigV0) Merge(
	src LogPoliciesConfigV0,
) LogPoliciesConfigV0 {
	// src is nil and b is an empty slice.
	if src == nil && b != nil && len(b) == 0 {
		return make(LogPoliciesConfigV0, 0)
	}
	var out LogPoliciesConfigV0

	patternTosrcLp := make(map[string]LogPolicyV0)
	for _, lp := range src {
		patternTosrcLp[lp.RawPattern] = lp
	}

	for _, otherLp := range b {
		pattern := otherLp.RawPattern
		if srcLp, ok := patternTosrcLp[pattern]; ok {
			// Merge actions of two LogPolicies if they have the same pattern.
			patternTosrcLp[pattern] = LogPolicyV0{
				RawPattern: pattern,
				RawActions: otherLp.RawActions.merge(srcLp.RawActions),
			}
		} else {
			// Source LogPoliciesConfig doesn't have this pattern.
			patternTosrcLp[pattern] = otherLp
		}
	}

	for _, lp := range patternTosrcLp {
		out = append(out, lp)
	}
	out.sort()
	return out
}

// UnmarshalJSON implements the json.Unmarshaler interface.
// The legacy log policies config allows entries with duplicated pattern because only single action
// can be associated with a pattern. For example:
//
//   - pattern: a
//     action:
//       type: cancel_retries
//   - pattern: a
//     action:
//       type: exclude_node
//
// All legacy policies become mordern policies after shimming:
//   - pattern: a
//     actions:
//      - cancel_retries
//   - pattern: a
//     actions:
//      - exclude_node
//
// The modern log policy has an actions field. Multiple entires can be associated with a pattern. No more
// duplicated pattern in log policies config. For example the policies above will be combined into:
//
//   - pattern: a
//     actions:
//      - cancel_retries
//      - exclude_node
func (b *LogPoliciesConfigV0) UnmarshalJSON(data []byte) error {
	// jsonItems may have duplicated patterns after applying shim to the legacy policy.
	type DefaultParser LogPoliciesConfigV0
	var jsonItems DefaultParser
	if err := json.Unmarshal(data, &jsonItems); err != nil {
		return errors.Wrapf(err, "failed to parse runtime items")
	}
	if jsonItems == nil {
		return nil
	}

	// Merge LogPolicyV0s with the same pattern into one.
	patternToLp := make(map[string]LogPolicyV0)
	for _, jsonItem := range jsonItems {
		pattern := jsonItem.RawPattern
		if _, ok := patternToLp[pattern]; !ok {
			patternToLp[pattern] = jsonItem
			continue
		}

		mergedActions := patternToLp[pattern].RawActions.merge(jsonItem.RawActions)
		patternToLp[pattern] = LogPolicyV0{RawPattern: pattern, RawActions: mergedActions}
	}

	// Can't use `var temp LogPolicies`. If the input data is [], function will return nil
	temp := make(LogPoliciesConfigV0, 0)
	for _, lp := range patternToLp {
		temp = append(temp, lp)
	}
	temp.sort()
	*b = temp

	return nil
}

// Sort LogPoliciesConfigV0 by pattern so the output is in deterministic state. Testing will be easier.
func (b LogPoliciesConfigV0) sort() {
	sort.Slice(b, func(i, j int) bool {
		return b[i].RawPattern < b[j].RawPattern
	})
}

// LogActionsV0 is a list of log actions.
type LogActionsV0 []LogActionV0

// LogPolicyV0 is an action to take if we match against trial logs.
//
//go:generate ../gen.sh
type LogPolicyV0 struct {
	RawPattern string       `json:"pattern"`
	RawActions LogActionsV0 `json:"actions,omitempty"`
}

// UnmarshalJSON implements the json.Unmarshaler interface.
// It applies shim to the legacy policy.
// For the modern policy, if a user provides multiple LogActionTypeSignal, only
// the last one will be stored. As for the other action types, the unique ones
// will be stored. For example:
//
//	actions:
//	  - cancel_retries
//	  - exclude_node
//	  - cancel_retries
//	  - signal: a
//	  - signal: b
//
// We store:
//
//	[]RawActionV0{
//	    {Type: LogActionTypeCancelRetries},
//	    {Type: LogActionTypeExcludeNode},
//	    {Type: LogActionTypeSignal, Signal: "b"}
//	}
func (b *LogPolicyV0) UnmarshalJSON(data []byte) error {
	// First, check if we can unmarshal the log policy item as a legacy item.
	type LegacyAction struct {
		Type LogActionType `json:"type"`
	}

	type LegacyPolicy struct {
		Pattern string        `json:"pattern"`
		Action  *LegacyAction `json:"action"`
	}

	var legacy LegacyPolicy
	err := json.Unmarshal(data, &legacy)

	// For this to be a valid LegacyPolicy, the Action must have been provided.
	if err == nil && legacy.Action != nil {
		// Apply shim to bring legacy policy into the current format.
		var action LogActionV0
		switch legacy.Action.Type {
		case LogActionTypeCancelRetries:
			action = LogActionV0{Type: LogActionTypeCancelRetries}
		case LogActionTypeExcludeNode:
			action = LogActionV0{Type: LogActionTypeExcludeNode}
		default:
			return fmt.Errorf("unregonized legacy action type: %s, data: %q", legacy.Action.Type, string(data))
		}
		*b = LogPolicyV0{
			RawPattern: legacy.Pattern,
			RawActions: []LogActionV0{action},
		}
		return nil
	}
	// Modern policy doesn't need shimming.
	type DefaultParser *LogPolicyV0
	var lp LogPolicyV0
	if err := json.Unmarshal(data, DefaultParser(&lp)); err != nil {
		return fmt.Errorf("failed to parse LogPolicyV0: %w, data: %q", err, string(data))
	}

	// Get the last LogActionTypeSignal, and get the unique values of the other types.
	var signal *LogActionV0
	otherActions := set.New[LogActionV0]()
	for _, a := range lp.RawActions {
		if a.Type == LogActionTypeSignal {
			signal = &a
		} else {
			otherActions.Insert(a)
		}
	}

	// Prepare output.
	if signal != nil {
		b.RawActions = []LogActionV0{*signal}
	}
	b.RawActions = append(b.RawActions, otherActions.ToSlice()...)
	b.RawActions.sort()
	b.RawPattern = lp.RawPattern

	return nil
}

// Merge LogActionsV0. The value of LogActionTypeSignal from l takes precedence.
// Union merge other LogAction types from l and src.
func (l LogActionsV0) merge(src LogActionsV0) LogActionsV0 {
	// Store unique actions except signal, and find source signal.
	actions := set.New[LogActionV0]()
	var srcSignal *LogActionV0
	for _, a := range src {
		if a.Type == LogActionTypeSignal {
			srcSignal = &a
			continue
		}
		actions.Insert(a)
	}

	// Store unique actions except signal, and find other signal.
	var otherSignal *LogActionV0
	for _, a := range l {
		if a.Type == LogActionTypeSignal {
			otherSignal = &a
			continue
		}
		actions.Insert(a)
	}
	// Other signal takes precedence.
	if otherSignal != nil || srcSignal != nil {
		actions.Insert(*schemas.Merge(otherSignal, srcSignal))
	}

	out := LogActionsV0(actions.ToSlice())
	out.sort()
	return out
}

// Sort LogActionsV0 by type so the output is in deterministic state. Testing will be easier.
func (l LogActionsV0) sort() {
	sort.Slice(l, func(i, j int) bool {
		return l[i].Type < l[j].Type
	})
}

// LogActionType is the type of an action.
type LogActionType string

// LogActionType refers to the action user can take when a pattern is detected in the log.
const (
	LogActionTypeCancelRetries LogActionType = "cancel_retries"
	LogActionTypeExcludeNode   LogActionType = "exclude_node"
	LogActionTypeSignal        LogActionType = "signal"
)

// LogActionV0 is a policy to take after matching.
type LogActionV0 struct {
	Type LogActionType

	// Only used by the "signal" action.
	Signal *string
}

// MarshalJSON implements the json.Marshaler interface.
func (s LogActionV0) MarshalJSON() ([]byte, error) {
	switch s.Type {
	case LogActionTypeSignal:
		return json.Marshal(struct {
			Signal *string `json:"signal"`
		}{Signal: s.Signal})
	case LogActionTypeCancelRetries:
		return json.Marshal(LogActionTypeCancelRetries)
	case LogActionTypeExcludeNode:
		return json.Marshal(LogActionTypeExcludeNode)
	}
	return nil, fmt.Errorf("failed to marshal LogActionV0: %+v", s)
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (s *LogActionV0) UnmarshalJSON(data []byte) error {
	var action string
	// err is not nil means input data is not cancel_retries or exclude_node.
	if err := json.Unmarshal(data, &action); err == nil {
		// Handle all the types beside signal
		switch LogActionType(action) {
		case LogActionTypeCancelRetries:
			*s = LogActionV0{Type: LogActionTypeCancelRetries}
			return nil
		case LogActionTypeExcludeNode:
			*s = LogActionV0{Type: LogActionTypeExcludeNode}
			return nil
		}
	}

	// Handle Signal
	temp := struct {
		Signal *string `json:"signal"`
	}{}
	if err := json.Unmarshal(data, &temp); err != nil || temp.Signal == nil {
		return fmt.Errorf("failed to unmarshal log action: %w, data: %q", err, string(data))
	}
	*s = LogActionV0{Type: LogActionTypeSignal, Signal: temp.Signal}

	return nil
}
