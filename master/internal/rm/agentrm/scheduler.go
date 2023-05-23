package agentrm

import (
	"fmt"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/model"
)

// Scheduler schedules tasks on agents.  Its only function Schedule is called
// to determine which pending requests can be fulfilled and which scheduled tasks
// can be terminated. Schedule is expected to ba called every time there is a change
// to the cluster status, for example, new agents being connected, devices being disabled,
// and etc,. Schedule should avoid unnecessary shuffling tasks on agents to avoid
// the overhead of restarting a preempted task.
type Scheduler interface {
	Schedule(rp *resourcePool) ([]*sproto.AllocateRequest, []model.AllocationID)
	JobQInfo(rp *resourcePool) map[model.JobID]*sproto.RMJobInfo
}

// MakeScheduler returns the corresponding scheduler implementation.
func MakeScheduler(conf *config.SchedulerConfig) Scheduler {
	switch conf.GetType() {
	case config.PriorityScheduling:
		return NewPriorityScheduler(conf)
	case config.FairShareScheduling:
		return NewFairShareScheduler()
	case config.RoundRobinScheduling:
		return NewRoundRobinScheduler()
	default:
		panic(fmt.Sprintf("invalid scheduler: %s", conf.GetType()))
	}
}
