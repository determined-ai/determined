package ft // rename ft

import (
	"fmt"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/set"
)

/*
write through cache the alerts. or nvm just use the db or in memory for first milestone.
*/

// DisallowedNodes returns a list of nodes that should be blacklisted for the given allocation
func DisallowedNodes(taskID model.TaskID) *set.Set[string] {
	fmt.Println(taskID)

	s := set.New[string]()
	s.Insert("agent1")
	s.Insert("gke-nblaskey-test2-node-pool-name-bae40778-jq79")
	return &s

	// maybe just taskid is enough. go off of task id and GetAlerts
	return nil
}

// log_policy_webhook
// id | task_id | regex | log | alert_report_time
func AddWebhookAlert(taskID model.TaskID, regex string, log string) error {
	return nil
}

// log_policy_dont_retry
// id | task_id | regex | log | alert_report_time
func AddDontRetry(taskID model.TaskID, regex string, log string) error {
	return nil
}

// log_policy_retry_on_different_node
// id | task_id | allocation_id | regex | log | restarts_left | alert_report_time
//
// To be explicit 1 restarts left means we have exactly one restart left.
func AddRetryOnDifferentNode(
	taskID model.TaskID, allocID model.AllocationID, regex string, log string,
) error {
	return nil
}

type RetryInfo struct {
	Regex string
	Log   string // TODO this could be a model.Log but just the string I think is fine for now.
}

func ShouldRetry(taskID model.TaskID) ([]RetryInfo, error) {
	return []RetryInfo{}, err
}

/*
type RetryInfo struct {
	Regex string
	Log   string // TODO this could be a model.Log but just the string I think is fine for now.
}

func ShouldRetryOnDifferentNode(taskID model.TaskID) ([]RetryDifferentNodeInfo, error) {
}
*/
