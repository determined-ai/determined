package logpattern

import (
	"context"
	"fmt"
	"regexp"
	"sync"

	lru "github.com/hashicorp/golang-lru/v2"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/task/tasklogger"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/master/pkg/set"

	"github.com/uptrace/bun"
)

const regexCacheSize = 256

var defaultSingleton *logPatternPolicies

type logPatternPolicies struct {
	blockListCache map[model.TaskID]*set.Set[string]
	mu             sync.RWMutex
	regexCache     *lru.Cache[string, *regexp.Regexp]
}

func (l *logPatternPolicies) getCompiledRegex(regex string) (*regexp.Regexp, error) {
	if compiledRegex, ok := l.regexCache.Get(regex); ok {
		return compiledRegex, nil
	}

	compiledRegex, err := regexp.Compile(regex)
	if err != nil {
		return nil, fmt.Errorf("compiling regex '%s': %w", regex, err)
	}
	l.regexCache.Add(regex, compiledRegex)

	return compiledRegex, nil
}

// New create the log pattern policies singleton.
// There are two reasons for this using a cache
//  1. Avoid the possibility this feature causes a major slowdown to Scheduler
//     that won't be obvious till it run at scale.
//  2. Avoid putting possible transient db errors in the path of the Scheduler.
//
// I think there is going to be a decent chance this cache approach will somehow leak tasks
// in the future but I think even if we never removed items from the cache
// we would still probably be okay.
func New(ctx context.Context) (*logPatternPolicies, error) { //nolint: revive
	var blockedNodes []*retryOnDifferentNode
	if err := db.Bun().NewSelect().Model(&blockedNodes).
		Where("task_ended = false").
		Scan(ctx, &blockedNodes); err != nil {
		return nil, fmt.Errorf("getting blocked nodes: %w", err)
	}

	blockListCache := make(map[model.TaskID]*set.Set[string])
	for _, b := range blockedNodes {
		if _, ok := blockListCache[b.TaskID]; !ok {
			blockListCache[b.TaskID] = ptrs.Ptr(set.New[string]())
		}
		blockListCache[b.TaskID].Insert(b.NodeName)
	}

	regexCache, err := lru.New[string, *regexp.Regexp](regexCacheSize)
	if err != nil {
		return nil, fmt.Errorf("creating LRU cache: %w", err)
	}

	return &logPatternPolicies{
		blockListCache: blockListCache,
		regexCache:     regexCache,
	}, nil
}

func (l *logPatternPolicies) monitor(ctx context.Context,
	taskID model.TaskID, logs []*model.TaskLog, policies expconf.LogPoliciesConfig,
) error {
	if len(policies) == 0 {
		return nil
	}

	// TODO when we add rm specific log grabbing we will need to also monitor them.
	for _, log := range logs {
		if log.AgentID == nil {
			return fmt.Errorf("agentID must be non nil to monitor logs")
		}

		for _, policy := range policies {
			compiledRegex, err := l.getCompiledRegex(policy.Pattern())
			if err != nil {
				return err
			}

			if compiledRegex.MatchString(log.Log) {
				switch policy.Action().GetUnionMember().(type) {
				case expconf.LogActionCancelRetries:
					if err := addDontRetry(
						ctx, model.TaskID(log.TaskID), *log.AgentID, policy.Pattern(), log.Log,
					); err != nil {
						return fmt.Errorf("adding don't retry: %w", err)
					}

				case expconf.LogActionExcludeNode:
					if err := l.addRetryOnDifferentNode(
						ctx, model.TaskID(log.TaskID), *log.AgentID, policy.Pattern(), log.Log,
					); err != nil {
						return fmt.Errorf("adding retry on different node: %w", err)
					}

				default:
					return fmt.Errorf("unrecognized log pattern policy type")
				}
			}
		}
	}

	return nil
}

func (l *logPatternPolicies) disallowedNodes(taskID model.TaskID) *set.Set[string] {
	l.mu.RLock()
	defer l.mu.RUnlock()

	disallowedNodes := l.blockListCache[taskID]
	if disallowedNodes != nil {
		return disallowedNodes
	}

	return ptrs.Ptr(set.New[string]())
}

func (l *logPatternPolicies) reportTaskDone(taskID model.TaskID) {
	l.mu.Lock()
	defer l.mu.Unlock()

	delete(l.blockListCache, taskID)
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

func (l *logPatternPolicies) addRetryOnDifferentNode(
	ctx context.Context, taskID model.TaskID, nodeName, regex, triggeringLog string,
) error {
	l.mu.Lock()
	defer l.mu.Unlock()

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

	tasklogger.Insert(tasklogger.CreateLogFromMaster(taskID, model.LogLevelError,
		fmt.Sprintf("(log '%q' matched regex %s) therefore will not schedule on %s\n",
			triggeringLog, regex, nodeName)))

	if _, ok := l.blockListCache[taskID]; !ok {
		l.blockListCache[taskID] = ptrs.Ptr(set.New[string]())
	}
	l.blockListCache[taskID].Insert(nodeName)
	return nil
}

// DontRetryTrigger has information about don't retry policies that have been triggered.
type DontRetryTrigger struct {
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

	// We don't send a log to the trial. The trial will do it if it failed.
	return nil
}

// ShouldRetry returns a list of any triggered log policies that prevent retrying a trial.
// Returns an empty list if taskID doesn't exist. Order is not guaranteed.
// Only returns first log that triggered each regex. Multiple policies with the same regex
// will only have one DontRetryTrigger.
func ShouldRetry(ctx context.Context, taskID model.TaskID) ([]DontRetryTrigger, error) {
	var models []*dontRetry
	if err := db.Bun().NewSelect().Model(&models).
		Where("task_id = ?", taskID).
		Scan(ctx, &models); err != nil {
		return nil, fmt.Errorf("getting taskID %s should retry: %w", taskID, err)
	}

	var out []DontRetryTrigger
	for _, m := range models {
		out = append(out, DontRetryTrigger{
			Regex:         m.Regex,
			TriggeringLog: m.TriggeringLog,
		})
	}

	return out, nil
}

// TaskLogsFromDontRetryTriggers returns informational task logs from dont retry triggers.
func TaskLogsFromDontRetryTriggers(taskID model.TaskID, t []DontRetryTrigger) []*model.TaskLog {
	var regexLogs []string
	for _, r := range t {
		regexLogs = append(regexLogs,
			fmt.Sprintf("(log %q matched regex %q)", r.TriggeringLog, r.Regex))
	}

	var taskLogs []*model.TaskLog
	for _, l := range append([]string{
		"trial failed and matched logs to a don't retry policy",
	}, regexLogs...) {
		taskLogs = append(taskLogs, tasklogger.CreateLogFromMaster(taskID, model.LogLevelError, l))
	}

	return taskLogs
}
