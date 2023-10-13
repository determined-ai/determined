package logpattern

import (
	"context"
	"fmt"
	"regexp"
	"sync"
	"testing"
	"time"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/task/tasklogger"
	"github.com/determined-ai/determined/master/internal/webhooks"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/master/pkg/set"
	lru "github.com/hashicorp/golang-lru/v2"

	"github.com/uptrace/bun"
)

var (
	blockListCache = make(map[model.TaskID]*set.Set[string])
	mu             sync.RWMutex
	regexCache     *lru.Cache[string, *regexp.Regexp]
	regexCacheSize = 128
)

func webhookTypeFromPolicy(p expconf.SendWebhookPolicy) webhooks.WebhookType {
	switch p.WebhookType() {
	case "default":
		return webhooks.WebhookTypeDefault
	case "slack":
		return webhooks.WebhookTypeSlack
	default:
		return webhooks.WebhookTypeDefault
	}
}

func Monitor(ctx context.Context,
	taskID model.TaskID, logs []*model.TaskLog, policies expconf.LogPatternPoliciesConfig,
) error {
	if len(policies) == 0 {
		return nil
	}

	// TODO when we add rm specific log grabbing we will need to also monitor them.
	for _, l := range logs {
		if l.AgentID == nil {
			return fmt.Errorf("agentID must be non nil to monitor logs")
		}

		for _, lpp := range policies {
			regex := fmt.Sprintf("(.*)%s(.*)", lpp.Pattern())
			compiledRegex, err := getCompiledRegex(regex, l.Log)
			if err != nil {
				return err
			}

			if compiledRegex.MatchString(l.Log) {
				switch policy := lpp.Policy().GetUnionMember().(type) {
				case expconf.DontRetryPolicyV0:
					if err := addDontRetry(
						ctx, model.TaskID(l.TaskID), *l.AgentID, lpp.Pattern(), l.Log,
					); err != nil {
						return fmt.Errorf("adding don't retry: %w", err)
					}
				case expconf.OnFailureExcludeNodePolicy:
					if err := addRetryOnDifferentNode(
						ctx, model.TaskID(l.TaskID), *l.AgentID, lpp.Pattern(), l.Log,
					); err != nil {
						return fmt.Errorf("adding retry on different node: %w", err)
					}
				case expconf.SendWebhookPolicy:
					if err := addWebhookAlert(
						ctx, model.TaskID(l.TaskID), *l.AgentID, lpp.Pattern(), l.Log,
						policy.WebhookURL(), webhookTypeFromPolicy(policy),
					); err != nil {
						return fmt.Errorf("adding webhook alert: %w", err)
					}
				default:
					return fmt.Errorf("unrecognized log pattern policy type")
				}
			}
		}
	}

	return nil
}

// There are two reasons for this using a cache
//  1. Avoid the possibility this feature causes a major slowdown to Scheduler
//     that won't be obvious till it run at scale.
//  2. Avoid putting possible transient db errors in the path of the Scheduler.
//
// I think there is going to be a decent chance this cache approach will somehow leak tasks
// in the future but I think even if we never removed items from the cache
// we would still probably be okay.
// Initialize the blocked node list.
func Initialize(ctx context.Context) error {
	mu.Lock()
	defer mu.Unlock()

	var blockedNodes []*retryOnDifferentNode
	if err := db.Bun().NewSelect().Model(&blockedNodes).
		Where("task_ended = false").
		Scan(ctx, &blockedNodes); err != nil {
		return fmt.Errorf("getting blocked nodes: %w", err)
	}

	blockListCache = make(map[model.TaskID]*set.Set[string])
	for _, b := range blockedNodes {
		if _, ok := blockListCache[b.TaskID]; !ok {
			blockListCache[b.TaskID] = ptrs.Ptr(set.New[string]())
		}
		blockListCache[b.TaskID].Insert(b.NodeName)
	}

	var err error
	regexCache, err = lru.New[string, *regexp.Regexp](regexCacheSize)
	if err != nil {
		return fmt.Errorf("creating LRU cache for compiled regex: %w", err)
	}

	return nil
}

// DisallowedNodes returns a list of nodes that should be blacklisted for the given allocation
func DisallowedNodes(taskID model.TaskID) *set.Set[string] {
	mu.RLock()
	defer mu.RUnlock()

	disallowedNodes := blockListCache[taskID]
	if disallowedNodes != nil {
		return disallowedNodes
	}

	return ptrs.Ptr(set.New[string]())
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
	NodeName      string       `bun:"node_name"`
	Regex         string       `bun:"regex"`
	TriggeringLog string       `bun:"triggering_log"`
	TaskEnded     bool         `bun:"task_ended"`
}

func addRetryOnDifferentNode(
	ctx context.Context, taskID model.TaskID, nodeName, regex, triggeringLog string,
) error {
	mu.Lock()
	defer mu.Unlock()

	m := &retryOnDifferentNode{
		TaskID:        taskID,
		NodeName:      nodeName,
		Regex:         regex,
		TriggeringLog: triggeringLog,
		TaskEnded:     false,
	}
	res, err := db.Bun().NewInsert().Model(m).
		On("CONFLICT (task_id, node_name, regex) DO NOTHING"). // Only care about the first log.
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("inserting log policy retry on different node alert %+v: %w", m, err)
	}
	if num, err := res.RowsAffected(); err != nil {
		return fmt.Errorf("retry different node rows affected: %w", err)
	} else if num == 0 {
		return nil
	}

	// TODO make master log function in tasklogger
	tasklogger.Insert(&model.TaskLog{
		TaskID:    string(taskID),
		Timestamp: ptrs.Ptr(time.Now().UTC()),
		Level:     ptrs.Ptr(model.LogLevelError),
		Source:    ptrs.Ptr("master"),
		StdType:   ptrs.Ptr("stdout"),
		Log: fmt.Sprintf("(log '%q' matched regex %s) therefore will not schedule on %s\n",
			triggeringLog, regex, nodeName),
	})

	// TODO actually maybe here we should do the cap check on the taskID.
	// Like getAgents and decide this should be killed?

	if _, ok := blockListCache[taskID]; !ok {
		blockListCache[taskID] = ptrs.Ptr(set.New[string]())
	}
	blockListCache[taskID].Insert(nodeName)
	return nil
}

type sendWebhook struct {
	bun.BaseModel `bun:"table:log_policy_send_webhook"`

	ID            int                  `bun:"id,pk,autoincrement"`
	TaskID        model.TaskID         `bun:"task_id"`
	Regex         string               `bun:"regex"`
	NodeName      string               `bun:"node_name"`
	TriggeringLog string               `bun:"triggering_log"`
	WebhookType   webhooks.WebhookType `bun:"webhook_type"`
	WebhookURL    string               `bun:"webhook_url"`
}

func addWebhookAlert(ctx context.Context,
	taskID model.TaskID, nodeName, regex, triggeringLog, url string, wt webhooks.WebhookType,
) error {
	// The reason we persist this is to avoid sending dupes.
	// Maybe the webhook package could handle this but I think it makes sense for us to do it here.
	m := &sendWebhook{
		TaskID:        taskID,
		NodeName:      nodeName,
		Regex:         regex,
		TriggeringLog: triggeringLog,
		WebhookURL:    url,
		WebhookType:   wt,
	}
	res, err := db.Bun().NewInsert().Model(m).
		On("CONFLICT (task_id, regex, webhook_type, webhook_url) DO NOTHING").
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("adding send webhook policy %+v: %w", m, err)
	}
	if num, err := res.RowsAffected(); err != nil {
		return fmt.Errorf("retry different node rows affected: %w", err)
	} else if num == 0 {
		return nil
	}

	// TODO make master log function in tasklogger
	tasklogger.Insert(&model.TaskLog{
		TaskID:    string(taskID),
		Timestamp: ptrs.Ptr(time.Now().UTC()),
		Level:     ptrs.Ptr(model.LogLevelError),
		Source:    ptrs.Ptr("master"),
		StdType:   ptrs.Ptr("stdout"),
		Log: fmt.Sprintf("(log '%q' matched regex %s) therefore sent webhook\n",
			triggeringLog, regex),
	})

	if err := webhooks.ReportLogPatternAction(
		ctx, taskID, nodeName, regex, triggeringLog, url, wt,
	); err != nil {
		return err
	}

	return nil
}

// RetryInfo has information about don't retry policies that have been triggered.
type RetryInfo struct {
	Regex         string
	TriggeringLog string
}

type dontRetry struct {
	bun.BaseModel `bun:"table:log_policy_dont_retry"`

	ID            int          `bun:"id,pk,autoincrement"`
	TaskID        model.TaskID `bun:"task_id"`
	Regex         string       `bun:"regex"`
	NodeName      string       `bun:"node_name"`
	TriggeringLog string       `bun:"triggering_log"`
}

func addDontRetry(
	ctx context.Context, taskID model.TaskID, nodeName, regex, triggeringLog string,
) error {
	m := &dontRetry{
		TaskID:        taskID,
		NodeName:      nodeName,
		Regex:         regex,
		TriggeringLog: triggeringLog,
	}
	if _, err := db.Bun().NewInsert().Model(m).
		On("CONFLICT (task_id, regex) DO NOTHING"). // Only care about the first log.
		Exec(ctx); err != nil {
		return fmt.Errorf("adding don't retry policy %+v: %w", m, err)
	}

	// TODO should we send a log to task logs here?
	// I kinda like it being the last message of the log.
	// The others make most of sense to me (well webhook does)
	// Maybe we should put retry on different node to end also? Nah prob is fine

	return nil
}

// ShouldRetry returns a list of any triggered log policies that prevent retrying a trial.
// Returns an empty list if taskID doesn't exist. Order is not guaranteed.
// Only returns first log that triggered each regex. Multiple policies with the same regex
// will only have one RetryInfo.
func ShouldRetry(ctx context.Context, taskID model.TaskID) ([]RetryInfo, error) {
	var models []*dontRetry
	if err := db.Bun().NewSelect().Model(&models).
		Where("task_id = ?", taskID).
		Scan(ctx, &models); err != nil {
		return nil, fmt.Errorf("getting taskID %s should retry: %w", taskID, err)
	}

	var out []RetryInfo
	for _, m := range models {
		out = append(out, RetryInfo{
			Regex:         m.Regex,
			TriggeringLog: m.TriggeringLog,
		})
	}

	return out, nil
}

func getCompiledRegex(regex string, log string) (*regexp.Regexp, error) {
	compiledRegex, ok := regexCache.Get(regex)
	if !ok {
		var err error
		compiledRegex, err = regexp.Compile(regex)
		if err != nil {
			return nil, fmt.Errorf("matching %s with %s: %w", regex, log, err)
		}
		regexCache.Add(regex, compiledRegex)
	}
	return compiledRegex, nil
}

// SetDisallowedNodesCacheTest is used only in unit tests. export_test.go does not work as expected.
// t *testing.T should convince you to not use this.
func SetDisallowedNodesCacheTest(t *testing.T, c map[model.TaskID]*set.Set[string]) {
	mu.Lock()
	defer mu.Unlock()
	blockListCache = c
}
