package logpattern

import (
	"context"

	log "github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/master/pkg/set"
)

// SetDefault sets the package level default for log pattern policies.
func SetDefault(p *logPatternPolicies) {
	defaultSingleton = p
}

// Monitor checks for logs against any log_pattern_policies and takes action according to the policy.
func Monitor(ctx context.Context,
	taskID model.TaskID, logs []*model.TaskLog, policies expconf.LogPoliciesConfig,
) error {
	if defaultSingleton == nil {
		log.Error("uninitialized log pattern policies")
		return nil
	}

	return defaultSingleton.monitor(ctx, taskID, logs, policies)
}

// DisallowedNodes returns a list of nodes that should be blocklisted for the given allocation.
func DisallowedNodes(taskID model.TaskID) *set.Set[string] {
	if defaultSingleton == nil {
		log.Error("uninitialized log pattern policies")
		return ptrs.Ptr(set.New[string]())
	}

	return defaultSingleton.disallowedNodes(taskID)
}

// ReportTaskDone cleans up taskID to disallowed nodes cache.
// This is safe to call multiple times and on tasks without disallowed nodes.
func ReportTaskDone(taskID model.TaskID) {
	if defaultSingleton == nil {
		log.Error("uninitialized log pattern policies")
		return
	}

	defaultSingleton.reportTaskDone(taskID)
}
