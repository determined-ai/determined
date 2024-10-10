package logpattern

import (
	"context"
	"fmt"
	"regexp"

	lru "github.com/hashicorp/golang-lru/v2"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/task/tasklogger"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"

	"github.com/uptrace/bun"
)

const regexCacheSize = 256

var (
	defaultSingleton *LogPatternPolicies
	// ExpconfigCompiledRegex is used to identify user submitted config.
	ExpconfigCompiledRegex = regexp.MustCompile("(.*)(\\\"log_policies\\\":)(.*)")
)

// LogPatternPolicies performs log pattern checks.
type LogPatternPolicies struct {
	regexCache *lru.Cache[string, *regexp.Regexp]
}

// New create the log pattern policies singleton.
func New(ctx context.Context) (*LogPatternPolicies, error) {
	regexCache, err := lru.New[string, *regexp.Regexp](regexCacheSize)
	if err != nil {
		return nil, fmt.Errorf("creating LRU cache: %w", err)
	}

	return &LogPatternPolicies{
		regexCache: regexCache,
	}, nil
}

func (l *LogPatternPolicies) getCompiledRegex(regex string) (*regexp.Regexp, error) {
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

func (l *LogPatternPolicies) monitor(ctx context.Context,
	taskID model.TaskID, logs []*model.TaskLog, policies expconf.LogPoliciesConfig,
) error {
	// TODO when we add rm specific log grabbing we will need to also monitor them.
	for _, policy := range policies {
		compiledRegex, err := l.getCompiledRegex(policy.Pattern())
		if err != nil {
			return err
		}

		for _, log := range logs {
			if log.AgentID == nil {
				return fmt.Errorf("agentID must be non nil to monitor logs")
			}

			// One of the trial logs prints expconf which has the regex pattern.
			// We skip monitoring this line.
			if ExpconfigCompiledRegex.MatchString(log.Log) {
				continue
			}

			if compiledRegex.MatchString(log.Log) {
				if actions := policy.Actions(); len(actions) > 0 {
					for _, a := range actions {
						switch a.GetUnionMember().(type) {
						case expconf.LogActionCancelRetries:
							if err := addDontRetry(
								ctx, model.TaskID(log.TaskID), *log.AgentID, policy.Pattern(), log.Log,
							); err != nil {
								return fmt.Errorf("adding don't retry: %w", err)
							}

						case expconf.LogActionExcludeNode:
							if err := addRetryOnDifferentNode(
								ctx, model.TaskID(log.TaskID), *log.AgentID, policy.Pattern(), log.Log,
							); err != nil {
								return fmt.Errorf("adding retry on different node: %w", err)
							}

						default:
							return fmt.Errorf("unrecognized log pattern policy type")
						}
					}
				}

				if policy.Signal() != nil {
					signal := policy.Signal()

					err = db.Bun().RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
						if _, err := tx.NewUpdate().Model(&model.Task{}).
							Set("log_signal = ?", signal).
							Where("task_id = ?", log.TaskID).
							Exec(ctx); err != nil {
							return fmt.Errorf("updating log signal of task %s: %w", log.TaskID, err)
						}
						if _, err := tx.NewUpdate().Model(&model.Run{}).
							Table("run_id_task_id").
							Set("log_signal = ?", signal).
							Where("run.id = run_id_task_id.run_id").
							Where("run_id_task_id.task_id = ?", log.TaskID).
							Exec(ctx); err != nil {
							return fmt.Errorf("updating log signal of task %s: %w", log.TaskID, err)
						}

						return nil
					})
					if err != nil {
						return fmt.Errorf("updating log signal: %w", err)
					}
				}
			}
		}
	}

	return nil
}

type retryOnDifferentNode struct {
	bun.BaseModel `bun:"table:log_policy_retry_on_different_node"`

	ID            int          `bun:"id,pk,autoincrement"`
	TaskID        model.TaskID `bun:"task_id"`
	NodeName      string       `bun:"node_name"`
	Regex         string       `bun:"regex"`
	TriggeringLog string       `bun:"triggering_log"`
}

// GetBlockedNodes returns nodes you can't schedule on due to log pattern policies.
func GetBlockedNodes(ctx context.Context, taskID model.TaskID) ([]string, error) {
	var resp []retryOnDifferentNode
	if err := db.Bun().NewSelect().Model(&resp).
		Where("task_id = ?", taskID).
		Column("node_name").
		Distinct().
		Scan(ctx, &resp); err != nil {
		return nil, fmt.Errorf("getting nodes for taskID %s: %w", taskID, err)
	}

	var o []string
	for _, r := range resp {
		o = append(o, r.NodeName)
	}
	return o, nil
}

func addRetryOnDifferentNode(
	ctx context.Context, taskID model.TaskID, nodeName, regex, triggeringLog string,
) error {
	m := &retryOnDifferentNode{
		TaskID:        taskID,
		NodeName:      nodeName,
		Regex:         regex,
		TriggeringLog: triggeringLog,
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

// ClearSignal resets the log_signal.
func ClearSignal(ctx context.Context, taskID model.TaskID) error {
	if err := db.Bun().RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.NewUpdate().Model(&model.Task{}).
			Set("log_signal = null").
			Where("task_id = ?", taskID).
			Exec(ctx); err != nil {
			return fmt.Errorf("resetting log signal of task %s: %w", taskID, err)
		}
		if _, err := tx.NewUpdate().Model(&model.Run{}).
			Table("run_id_task_id").
			Set("log_signal = null").
			Where("run.id = run_id_task_id.run_id").
			Where("run_id_task_id.task_id = ?", taskID).
			Exec(ctx); err != nil {
			return fmt.Errorf("resetting log signal of task %s: %w", taskID, err)
		}

		return nil
	}); err != nil {
		return fmt.Errorf("resetting log signal: %w", err)
	}

	return nil
}
