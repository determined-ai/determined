package configpolicies

import (
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
)

// ResourceConstraints are non-overridable resource constraints.
// Submitted workloads that request resource quanities exceeding defined resource constraints in a
// given scope are rejected.
type ResourceConstraints struct {
	MaxSlots    *int `json:"max_slots"`
	MaxPriority *int `json:"max_priority"`
}

// Constraints are non-overridable workload constraints.
// Submitted workloads whose config's respective field(s) exceed defined constraints within a given
// scope are rejected.
type Constraints struct {
	ResourceConstraints ResourceConstraints `json:"resources"`
}

// ExperimentTaskConfigPolicy is the invariant config and constraints for an experiment.
// Submitted experiments whose config fields vary from the respective InvariantConfig fields set
// within a given scope are silently overriden.
// Submitted experiments whose constraint fields vary from the respective Constraint fields set
// within a given scope are rejected.
type ExperimentConfigPolicy struct {
	InvariantConfig expconf.ExperimentConfig `json:"config"`
	Constraints     Constraints              `json:"constraints"`
}

// NTSCTaskConfigPolicy is the invariant config and constraints for an NTSC task.
// Submitted NTSC tasks whose config fields vary from the respective InvariantConfig fields set
// within a given scope are silently overriden.
// Submitted NTSC tasks whose constraint fields vary from the respective Constraint fields set
// within a given scope are rejected.
type NTSCConfigPolicy struct {
	InvariantConfig model.CommandConfig `json:"config"`
	Constraints     Constraints         `json:"constraints"`
}
