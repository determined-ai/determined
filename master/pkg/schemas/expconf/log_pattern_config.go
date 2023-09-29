package expconf

import (
	"encoding/json"
	"fmt"

	"github.com/determined-ai/determined/master/pkg/schemas"
	"github.com/determined-ai/determined/master/pkg/union"
)

// LogPatternPoliciesConfigV0 is a list of log pattern actions.
//
//go:generate ../gen.sh
type LogPatternPoliciesConfigV0 []LogPatternPolicyV0

// TODO test merging in task container defaults.
/*
// Merge implemenets the mergable interface.
func (b LogPatternPoliciesConfigV0) Merge(
	other LogPatternPoliciesConfigV0,
) LogPatternPoliciesConfigV0 {
	// TODO add both together...

	// Yeah like don't define cluster level logs?
	return append(b, other...)
}
*/

// LogPatternPolicyV0 is an action to take if we match against trial logs.
//
//go:generate ../gen.sh
type LogPatternPolicyV0 struct {
	RawPattern string `json:"pattern"`

	RawPolicy *LogPolicyV0 `json:"policy"`
}

// LogPatternPoliciesPolicyV0 is a policy to take after matching.
//
//go:generate ../gen.sh
type LogPolicyV0 struct {
	RawOnFailureDontRetry   *DontRetryPolicyV0            `union:"type,on_failure_dont_retry" json:"-"`
	RawOnFailureExcludeNode *OnFailureExcludeNodePolicyV0 `union:"type,on_failure_exclude_node" json:"-"`
	RawSendWebhook          *SendWebhookPolicyV0          `union:"type,send_webhook" json:"-"`
}

// Merge implements schemas.Mergeable.
func (s LogPolicyV0) Merge(other LogPolicyV0) LogPolicyV0 {
	return schemas.UnionMerge(s, other)
}

// MarshalJSON implements the json.Marshaler interface.
func (s LogPolicyV0) MarshalJSON() ([]byte, error) {
	return union.MarshalEx(s, true)
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (s *LogPolicyV0) UnmarshalJSON(data []byte) error {
	if err := union.Unmarshal(data, s); err != nil {
		return err
	}
	type DefaultParser *LogPolicyV0
	if err := json.Unmarshal(data, DefaultParser(s)); err != nil {
		return fmt.Errorf("failed to parse searcher config: %w", err)
	}
	return nil
}

// DontRetryPolicyV0 doesn't retry the trial if it fails.
//
//go:generate ../gen.sh
type DontRetryPolicyV0 struct {
	// This comment is needed to stop ../gen.sh from complaining.
}

// OnFailureExcludeNodePolicyV0 will exclude the node the log was seen on
// (only for that trial) and reschedule.
//
//go:generate ../gen.sh
type OnFailureExcludeNodePolicyV0 struct {
	RawRestarts *int `json:"restarts"`
}

// SendWebhookPolicyV0
//
//go:generate ../gen.sh
type SendWebhookPolicyV0 struct {
	RawWebhookName string `json:"webhook_name"`
}
