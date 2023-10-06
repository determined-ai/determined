package ft // rename ft

import (
	"fmt"
	"sync"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/set"

	"github.com/uptrace/bun"
)

// START TODO (maybe seperate file? PROB NOT)

var (
	blockListCache map[model.TaskID]*set.Set[string]
	mu             sync.RWMutex
)

func InitializeLogPatternActions() error {
	mu.Lock()
	defer mu.Unlock()

	// TODO load non done tasks into memory for block list.
	// TODO is this ever going to be an performance issue?
	// I mean worse case it is like a scan of tasks on restart?
	//
	// The alternative is to change like endTasks to add a column as invalid here.
	// I think that might be reasonable just to make this not evena  question.
	return nil
}

// DisallowedNodes returns a list of nodes that should be blacklisted for the given allocation
func DisallowedNodes(taskID model.TaskID) *set.Set[string] {
	mu.RLock()
	defer mu.RUnlock()

	return s[taskID] // TODO copy the map returning.
}

// ReportTaskDone cleans up taskID to disallowed nodes cache.
// This is safe to call multiple times and on tasks without disallowed nodes.
func ReportTaskDone(taskID model.TaskID) {
	mu.Lock()
	defer mu.Unlock()

	delete(blockListCache, taskID)
}

type retryOnDifferentNode struct {
	bun.BaseModel `bun:"table:log_policy_retry_on_different_node"`

	ID            int          `bun:"id,pk,autoincrement"`
	TaskID        model.TaskID `bun:"task_id"`
	Regex         string       `bun:"regex"`
	NodeName      string       `bun:"node_name"`
	TriggeringLog string       `bun:"triggering_log"`
}

// AddRetryOnDifferentNode comment.
func AddRetryOnDifferentNode(
	context context.Context, taskID model.TaskID, nodeName, regex, log string,
) error {
	m := &retryOnDifferentNode{
		TaskID:        taskID,
		Regex:         regex,
		NodeName:      nodeName,
		TriggeringLog: log,
	}
	_, err := db.Bun().NewInsert().Model(m).Exec(ctx)
	if err != nil {
		return fmt.Errorf("inserting log policy retry on different node alert %+v: %w", m, err)
	}

	mu.Lock()
	defer mu.Unlock()

	if _, ok := blockListCache[taskID]; !ok {
		blockListCache[taskID] = set.New[string]()
	}
	blockListCache[taskID].Insert(nodeName)
	return nil
}

// END TODO

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
