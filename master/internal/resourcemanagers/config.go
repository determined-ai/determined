package resourcemanagers

import (
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/union"
)

// Config hosts configuration fields of the scheduler.
type Config struct {
	Type             string                  `json:"type"`
	Fit              string                  `json:"fit"`
	ResourceProvider *ResourceProviderConfig `json:"resource_provider"`
}

// ResourceProviderConfig hosts configuration fields for the resource provider.
type ResourceProviderConfig struct {
	DefaultRPConfig    *DefaultResourceProviderConfig   `union:"type,default" json:"-"`
	KubernetesRPConfig *KubernetesResourceManagerConfig `union:"type,kubernetes" json:"-"`
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

// MakeFitFunction returns the corresponding fitting function.
func MakeFitFunction(fittingPolicy string) func(*AllocateRequest, *agentState) float64 {
	switch fittingPolicy {
	case "worst":
		return WorstFit
	case "best":
		return BestFit
	default:
		panic(fmt.Sprintf("invalid scheduler fit: %s", fittingPolicy))
	}
}

// MakeScheduler returns the corresponding scheduler implementation.
func MakeScheduler(schedulingPolicy string) Scheduler {
	switch schedulingPolicy {
	case "priority":
		return NewPriorityScheduler()
	case "fair_share":
		return NewFairShareScheduler()
	default:
		panic(fmt.Sprintf("invalid scheduler: %s", schedulingPolicy))
	}
}
