package logpattern

import (
	"context"

	log "github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
)

// SetDefault sets the package level default for log pattern policies.
func SetDefault(p *LogPatternPolicies) {
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
