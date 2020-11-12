package resourcemanagers

import (
	"github.com/determined-ai/determined/master/pkg/check"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/union"
)

const (
	defaultSchedulingPriority = 42

	fairShareScheduling = "fair_share"
	priorityScheduling  = "priority"

	best             = "best"
	worst            = "worst"
	defaultFitPolicy = best
)

func defaultSchedulerConfig() *SchedulerConfig {
	return &SchedulerConfig{
		FairShare:     &FairShareSchedulerConfig{},
		FittingPolicy: defaultFitPolicy,
	}
}

// SchedulerConfig holds the configurations for scheduling policies.
type SchedulerConfig struct {
	FairShare     *FairShareSchedulerConfig `union:"type,fair_share" json:"-"`
	Priority      *PrioritySchedulerConfig  `union:"type,priority" json:"-"`
	FittingPolicy string                    `json:"fitting_policy"`
}

// MarshalJSON implements the json.Marshaler interface.
func (s SchedulerConfig) MarshalJSON() ([]byte, error) {
	return union.Marshal(s)
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (s *SchedulerConfig) UnmarshalJSON(data []byte) error {
	return union.Unmarshal(data, s)
}

// Validate implements the check.Validatable interface.
func (s SchedulerConfig) Validate() []error {
	return []error{
		check.Contains(
			s.FittingPolicy, []interface{}{best, worst}, "invalid fitting policy",
		),
	}
}

func fillInSchedulerDefaults(s *SchedulerConfig) {
	if s.FittingPolicy == "" {
		s.FittingPolicy = defaultFitPolicy
	}

	if s.Priority != nil && s.Priority.DefaultPriority == nil {
		defaultPriority := defaultSchedulingPriority
		s.Priority.DefaultPriority = &defaultPriority
	}
}

func (s *SchedulerConfig) getType() string {
	switch {
	case s.FairShare != nil:
		return fairShareScheduling
	case s.Priority != nil:
		return priorityScheduling
	default:
		panic("neither scheduler type configured")
	}
}

// FairShareSchedulerConfig holds configurations for the fair share scheduler.
type FairShareSchedulerConfig struct{}

// PrioritySchedulerConfig holds the configurations for the priority scheduler.
type PrioritySchedulerConfig struct {
	Preemption      bool `json:"preemption"`
	DefaultPriority *int `json:"default_priority"`
}

// Validate implements the check.Validatable interface.
func (p PrioritySchedulerConfig) Validate() []error {
	return model.ValidatePrioritySetting(p.DefaultPriority)
}
