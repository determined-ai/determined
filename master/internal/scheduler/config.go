package scheduler

import (
	"fmt"

	"github.com/determined-ai/determined/master/pkg/check"
)

// DefaultConfig is the default configuration of the scheduler.
func DefaultConfig() *Config {
	return &Config{
		Type: "fair_share",
		Fit:  "best",
	}
}

// Config hosts configuration fields of the scheduler.
type Config struct {
	Type string `json:"type"`
	Fit  string `json:"fit"`
}

// Validate implements the check.Validatable interface.
func (c Config) Validate() []error {
	return []error{
		check.Contains(c.Type, []interface{}{"priority", "fair_share"}, "invalid scheduler type"),
		check.Contains(c.Fit, []interface{}{"best", "worst"}, "invalid scheduler fitting method"),
	}
}

// FitFunction returns the corresponding function for the Fit field in Config.
func (c Config) FitFunction() func(*Task, *agentState) float64 {
	switch c.Fit {
	case "worst":
		return WorstFit
	case "best":
		return BestFit
	default:
		panic(fmt.Sprintf("invalid scheduler fit: %s", c.Fit))
	}
}

// MakeScheduler returns the corresponding scheduler implementation for the Type field in Config.
func (c Config) MakeScheduler() Scheduler {
	switch c.Type {
	case "priority":
		return NewPriorityScheduler()
	case "fair_share":
		return NewFairShareScheduler()
	default:
		panic(fmt.Sprintf("invalid scheduler: %s", c.Type))
	}
}
