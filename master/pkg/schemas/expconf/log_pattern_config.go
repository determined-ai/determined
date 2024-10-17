package expconf

import (
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/schemas"
	"github.com/determined-ai/determined/master/pkg/set"
	"github.com/determined-ai/determined/master/pkg/union"
)

const (
	cancelRetries = "cancel_retries"
	excludeNode   = "exclude_node"
)

// LogPoliciesConfigV0 is a list of log policies.
//
//go:generate ../gen.sh
type LogPoliciesConfigV0 []LogPolicyV0

// WithDefaults implements the Defaultable psuedointerface.
func (b LogPoliciesConfigV0) WithDefaults() LogPoliciesConfigV0 {
	eccErrorPattern := ECCErrorPattern
	eccErrorSignal := ECCErrorSignal
	cudaOomPattern := CUDAOOMPattern
	cudaOomSignal := CUDAOOMSignal

	if b != nil && len(b) == 0 {
		return LogPoliciesConfigV0{
			LogPolicyV0{RawPattern: eccErrorPattern, RawActions: []LogActionV0{{Signal: &eccErrorSignal}}},
			LogPolicyV0{RawPattern: cudaOomPattern, RawActions: []LogActionV0{{Signal: &cudaOomSignal}}},
		}
	}
	return b
}

// Merge implements the Mergable psuedo-interface.
func (b LogPoliciesConfigV0) Merge(
	other LogPoliciesConfigV0,
) LogPoliciesConfigV0 {
	var out LogPoliciesConfigV0

	patternToLp := make(map[string]LogPolicyV0)
	for _, lp := range other {
		patternToLp[lp.RawPattern] = lp
	}

	for _, lp := range b {
		if v, ok := patternToLp[lp.RawPattern]; ok {
			// Union merge all actions except signal
			actions := set.New[LogActionV0]()
			var signal *LogActionV0
			for _, a := range patternToLp[lp.RawPattern].RawActions {
				if a.Signal != nil {
					signal = &a
					continue
				}
				actions.Insert(a)
			}
			var otherSignal *LogActionV0
			for _, a := range lp.RawActions {
				if a.Signal != nil {
					otherSignal = &a
					continue
				}
				if !actions.Contains(a) {
					v.RawActions = append(v.RawActions, a)
				}
			}

			// Other signal takes precedence
			if otherSignal != nil || signal != nil {
				v.RawActions = append(v.RawActions, *schemas.Merge(otherSignal, signal))
			}

			patternToLp[lp.RawPattern] = v
		} else {
			patternToLp[lp.RawPattern] = lp
		}
	}

	for _, p := range patternToLp {
		out = append(out, p)
	}
	return out
}

// UnmarshalJSON implements the json.Unmarshaler interface.
// The legacy log policy config allows entries with duplicated pattern because only single action
// can be associated with a pattern. For example:
//
//   - pattern: a
//     action: cancel_retries
//   - pattern: a
//     action: exclude_node
//
// An actions field is available now. Multiple entires can be associated with a pattern. No more
// duplicated pattern in log policies config. For example:
//
//   - pattern: a
//     actions:
//   - cancel_retries
//   - exclude_node
func (b *LogPoliciesConfigV0) UnmarshalJSON(data []byte) error {
	type DefaultParser LogPoliciesConfigV0
	var jsonItems DefaultParser
	if err := json.Unmarshal(data, &jsonItems); err != nil {
		return errors.Wrapf(err, "failed to parse runtime items")
	}

	// Merge LogPolicyV0s with the same pattern into one
	patternToActions := make(map[string]set.Set[LogActionV0])
	for _, jsonItem := range jsonItems {
		if jsonItem.RawLegacyAction != nil {
			return fmt.Errorf("legacy action field expected to be nil: %+v", jsonItem)
		}

		if _, ok := patternToActions[jsonItem.RawPattern]; !ok {
			patternToActions[jsonItem.RawPattern] = set.New[LogActionV0]()
		}

		for _, a := range jsonItem.RawActions {
			actions := patternToActions[jsonItem.RawPattern]
			actions.Insert(a)
		}
	}

	for p, actions := range patternToActions {
		lp := LogPolicyV0{
			RawPattern: p,
			RawActions: actions.ToSlice(),
		}
		*b = append(*b, lp)
	}

	return nil
}

// LogPolicyV0 is an action to take if we match against trial logs.
//
//go:generate ../gen.sh
type LogPolicyV0 struct {
	RawPattern      string             `json:"pattern"`
	RawLegacyAction *LogLegacyActionV0 `json:"action,omitempty"`
	RawActions      []LogActionV0      `json:"actions,omitempty"`
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (b *LogPolicyV0) UnmarshalJSON(data []byte) error {
	// First, check if we can unmarshal the log policy item as a legacy item.
	type LegacyCancelRetries struct {}

	type LegacyExcludeNode struct {}

	type LegacyAction struct {
		CancelRetries *LegacyCancelRetries `union:"type,cancel_retries" json:"-"`
		ExcludeNode   *LegacyExcludeNode   `union:"type,exclude_node" json:"-"`
	}

	type LegacyPolicy struct {
		Pattern string        `json:"pattern"`
		Action  *LegacyAction  `json:"action"`
	}

	var legacy LegacyPolicy
	err := json.Unmarshal(data, &legacy)
	// For this to be a valid LegacyPolicy, the Action must have been provided.
	if err == nil && legacy.Action != nil {
		// Apply shim to bring legacy policy into the current format.
		var action LogActionV0
		if legacy.Action.CancelRetries {
			action = LogActionV0{Type: LogActionTypeCancelRetries}
		} else {
			action = LogActionV0{Type: LogActionTypeExcludeNode}
		}
		*b = LogPolicyV0{
			RawPattern: legacy.Pattern,
			RawActions: []LogActionV0{action},
		}
		return nil
	}

	// Shimming is not necessary.
	type DefaultParser *LogPolicyV0
	if err := json.Unmarshal(data, DefaultParser(b)); err != nil {
		return fmt.Errorf("failed to parse LogPolicyV0: %w", err)
	}
	return nil
}

type LogActionType string

const LogActionTypeCancelRetries LogActionType = "cancel_retries"
const LogActionTypeExcludeNodes LogActionType = "exclude_nodes"
const LogActionTypeSignal LogActionType = "signal"

// LogActionV0 is a policy to take after matching.
type LogActionV0 struct {
	Type LogActionType

	// Only used by the "signal" action.
	Signal string
}

// MarshalJSON implements the json.Marshaler interface.
func (s LogActionV0) MarshalJSON() ([]byte, error) {
	// XXX case statement here
	if s.Signal != nil {
		return json.Marshal(struct {
			Signal string `json:"signal"`
		}{Signal: *s.Signal})
	} else if s.CancelRetries != nil {
		return json.Marshal(cancelRetries)
	} else if s.ExcludeNode != nil {
		return json.Marshal(excludeNode)
	}
	return nil, fmt.Errorf("failed to marshal LogActionV0: %+v", s)
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (s *LogActionV0) UnmarshalJSON(data []byte) error {
	var action string
	if err := json.Unmarshal(data, &action); err == nil {
		switch action {
		case LogActionTypeCancelRetries:
			*s = LogActionV0{Type: LogActionTypeCancelRetries}
			return nil
		case LogActionTypeExcludeNode:
			*s = LogActionV0{Type: LogActionTypeExcludeNode}
			return nil
		default:
			return fmt.Errorf("invalid log action: %v", action)
		}
	}

	// Handle Signal
	temp := struct {
		Signal *string `json:"signal"`
	}{}
	if err := json.Unmarshal(data, &temp); err != nil {
		return fmt.Errorf("failed to unmarshal log action: %q", string(data))
	}
	*s = LogActionV0{Type: LogActionTypeSignal, Signal: out.Signal}
	return nil
}
