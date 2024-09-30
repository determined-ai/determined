package expconf

import (
	"encoding/json"
	"fmt"

	log "github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/pkg/schemas"
	"github.com/determined-ai/determined/master/pkg/union"
)

// LogPoliciesConfigV0 is a list of log policies.
type LogPoliciesConfigV0 []LogPolicyV0

// WithDefaults implements the Defaultable psuedointerface.
func (l *LogPoliciesConfigV0) WithDefaults() *LogPoliciesConfigV0 {
	eccErrorPattern := ECCErrorPattern
	eccErrorSignal := ECCErrorSignal
	cudaOomPattern := CUDAOOMPattern
	cudaOomSignal := CUDAOOMSignal

	if l != nil && len(*l) == 0 {
		return &LogPoliciesConfigV0{
			LogPolicyV0{RawPattern: eccErrorPattern, RawSignal: &eccErrorSignal},
			LogPolicyV0{RawPattern: cudaOomPattern, RawSignal: &cudaOomSignal},
		}
	}
	return l
}

// Merge implemenets the mergable interface.
func (b LogPoliciesConfigV0) Merge(
	other LogPoliciesConfigV0,
) LogPoliciesConfigV0 {
	var out LogPoliciesConfigV0
	seen := make(map[string]bool)
	for _, p := range append(other, b...) {
		json, err := json.Marshal(p)
		if err != nil {
			log.Errorf("marshaling error %+v %v", p, err)
		}
		fmt.Printf("\njson: %#v\n", json)
		if seen[string(json)] {
			continue
		}
		seen[string(json)] = true

		out = append(out, p)
	}
	return out
}

// LogPolicyV0 is an action to take if we match against trial logs.
//
//go:generate ../gen.sh
type LogPolicyV0 struct {
	RawPattern string `json:"pattern"`

	RawAction *LogActionV0 `json:"action,omitempty"`
	RawSignal *string      `json:"signal,omitempty"`
}

// LogActionV0 is a policy to take after matching.
//
//go:generate ../gen.sh
type LogActionV0 struct {
	RawCancelRetries *LogActionCancelRetriesV0 `union:"type,cancel_retries" json:"-"`
	RawExcludeNode   *LogActionExcludeNodeV0   `union:"type,exclude_node" json:"-"`
}

// Merge implements schemas.Mergeable.
func (s LogActionV0) Merge(other LogActionV0) LogActionV0 {
	return schemas.UnionMerge(s, other)
}

// MarshalJSON implements the json.Marshaler interface.
func (s LogActionV0) MarshalJSON() ([]byte, error) {
	return union.MarshalEx(s, true)
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (s *LogActionV0) UnmarshalJSON(data []byte) error {
	if err := union.Unmarshal(data, s); err != nil {
		return err
	}
	type DefaultParser *LogActionV0
	if err := json.Unmarshal(data, DefaultParser(s)); err != nil {
		return fmt.Errorf("failed to parse searcher config: %w", err)
	}
	return nil
}

// LogActionCancelRetriesV0 doesn't retry the trial if it fails.
//
//go:generate ../gen.sh
type LogActionCancelRetriesV0 struct {
	// This comment is needed to stop ../gen.sh from complaining.
}

// LogActionExcludeNodeV0 will exclude the node the log was seen on
// (only for that trial) and reschedule.
//
//go:generate ../gen.sh
type LogActionExcludeNodeV0 struct {
	// This comment is needed to stop ../gen.sh from complaining.
}
