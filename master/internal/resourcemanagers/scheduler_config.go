package resourcemanagers

import (
	"github.com/determined-ai/determined/master/pkg/check"
	"github.com/determined-ai/determined/master/pkg/union"
)

const (
	// MinUserSchedulingPriority is the smallest priority users may specify.
	MinUserSchedulingPriority = 1
	// MaxUserSchedulingPriority is the largest priority users may specify.
	MaxUserSchedulingPriority = 99
	// DefaultSchedulingPriority is the default scheduling policy if users do not specify one.
	DefaultSchedulingPriority = 42

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
			s.FittingPolicy, []interface{}{"best", "worst"}, "invalid fitting policy",
		),
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
	errors := make([]error, 0)

	if p.DefaultPriority != nil {
		errors = append(errors, check.GreaterThanOrEqualTo(
			*p.DefaultPriority, MinUserSchedulingPriority,
			"scheduling priority must be greater than 0 and less than 100"))
		errors = append(errors, check.LessThanOrEqualTo(
			*p.DefaultPriority, MaxUserSchedulingPriority,
			"scheduling priority must be greater than 0 and less than 100"))
	}
	return errors
}
