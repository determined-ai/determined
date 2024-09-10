package configpolicy

import (
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
)

// ExperimentConfigPolicies is the invariant config and constraints for an experiment.
// Submitted experiments whose config fields vary from the respective InvariantConfig fields set
// within a given scope are silently overridden.
// Submitted experiments whose constraint fields vary from the respective Constraint fields set
// within a given scope are rejected.
type ExperimentConfigPolicies struct {
	InvariantConfig *expconf.ExperimentConfig `json:"config"`
	Constraints     *model.Constraints        `json:"constraints"`
}

// NTSCConfigPolicies is the invariant config and constraints for an NTSC task.
// Submitted NTSC tasks whose config fields vary from the respective InvariantConfig fields set
// within a given scope are silently overridden.
// Submitted NTSC tasks whose constraint fields vary from the respective Constraint fields set
// within a given scope are rejected.
type NTSCConfigPolicies struct {
	InvariantConfig *model.CommandConfig `json:"config"`
	Constraints     *model.Constraints   `json:"constraints"`
}
