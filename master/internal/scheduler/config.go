package scheduler

import (
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/check"
	"github.com/determined-ai/determined/master/pkg/union"
)

// DefaultConfig is the default configuration of the scheduler.
func DefaultConfig() *Config {
	return &Config{
		Type: "fair_share",
		Fit:  "best",
		ResourceProvider: &ResourceProviderConfig{
			DefaultRPConfig: &DefaultResourceProviderConfig{},
		},
	}
}

// Config hosts configuration fields of the scheduler.
type Config struct {
	Type             string                  `json:"type"`
	Fit              string                  `json:"fit"`
	ResourceProvider *ResourceProviderConfig `json:"resource_provider"`
}

// Validate implements the check.Validatable interface.
func (c Config) Validate() []error {
	return []error{
		check.Contains(c.Type, []interface{}{"priority", "fair_share"}, "invalid scheduler type"),
		check.Contains(c.Fit, []interface{}{"best", "worst"}, "invalid scheduler fitting method"),
		check.True(c.ResourceProvider != nil, "resource provider not set"),
	}
}

// FitFunction returns the corresponding function for the Fit field in Config.
func (c Config) FitFunction() func(*AddTask, *agentState) float64 {
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

// ResourceProviderConfig hosts configuration fields for the resource provider.
type ResourceProviderConfig struct {
	DefaultRPConfig    *DefaultResourceProviderConfig    `union:"type,default" json:"-"`
	KubernetesRPConfig *KubernetesResourceProviderConfig `union:"type,kubernetes" json:"-"`
}

// MarshalJSON implements the json.Marshaler interface.
func (r ResourceProviderConfig) MarshalJSON() ([]byte, error) {
	return union.Marshal(r)
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (r *ResourceProviderConfig) UnmarshalJSON(data []byte) error {
	if err := union.Unmarshal(data, r); err != nil {
		return err
	}
	type DefaultParser *ResourceProviderConfig
	return errors.Wrap(json.Unmarshal(data, DefaultParser(r)), "failed to parse resource provider")
}

// DefaultResourceProviderConfig hosts configuration fields for the default resource provider.
type DefaultResourceProviderConfig struct{}

// KubernetesResourceProviderConfig hosts configuration fields for the kubernetes resource provider.
type KubernetesResourceProviderConfig struct {
	Namespace                string `json:"namespace"`
	MaxSlotsPerPod           int    `json:"max_slots_per_pod"`
	MasterServiceName        string `json:"master_service_name"`
	LeaveKubernetesResources bool   `json:"leave_kubernetes_resources"`
}

// Validate implements the check.Validatable interface.
func (k *KubernetesResourceProviderConfig) Validate() []error {
	return []error{
		check.GreaterThanOrEqualTo(k.MaxSlotsPerPod, 0, "max_slots_per_pod must be >= 0"),
	}
}
