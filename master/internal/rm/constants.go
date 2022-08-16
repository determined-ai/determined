package rm

import (
	"time"
)

const (
	// DefaultSchedulingPriority is the default resource manager priority.
	DefaultSchedulingPriority = 42

	actionCoolDown          = 500 * time.Millisecond
	defaultResourcePoolName = "default"
)

// schedulerTick periodically triggers the scheduler to act.
type schedulerTick struct{}
