package expconf

import (
	"encoding/json"
	"fmt"

	"github.com/determined-ai/determined/master/pkg/schemas"
	"github.com/determined-ai/determined/master/pkg/union"
	log "github.com/sirupsen/logrus"
)

// LogPatternPoliciesConfigV0 is a list of log pattern actions.
//
//go:generate ../gen.sh
type LogPatternPoliciesConfigV0 []LogPatternPolicyV0

// Merge implemenets the mergable interface.
func (b LogPatternPoliciesConfigV0) Merge(
	other LogPatternPoliciesConfigV0,
) LogPatternPoliciesConfigV0 {
	var out LogPatternPoliciesConfigV0
	seen := make(map[string]bool)
	for _, p := range append(other, b...) {
		json, err := json.Marshal(p)
		if err != nil {
			log.Errorf("marshaling error %+v %v", p, err)
		}
		if seen[string(json)] {
			continue
		}
		seen[string(json)] = true

		out = append(out, p)
	}
	return out
}

// LogPatternPolicyV0 is an action to take if we match against trial logs.
//
//go:generate ../gen.sh
type LogPatternPolicyV0 struct {
	RawPattern string `json:"pattern"`

	RawPolicy *LogPolicyV0 `json:"policy"`
}

// LogPolicyV0 is a policy to take after matching.
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
	// This comment is needed to stop ../gen.sh from complaining.
}

// SendWebhookPolicyV0 will send a webhook.
//
//go:generate ../gen.sh
type SendWebhookPolicyV0 struct {
	RawWebhookType string `json:"webhook_type"`
	RawWebhookURL  string `json:"webhook_url"`
}
