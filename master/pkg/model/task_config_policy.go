package model

import (
	"time"

	"github.com/uptrace/bun"
)

// Constants.

const (
	// UnknownType constant.
	UnknownType string = "UNSPECIFIED"
	// ExperimentType constant.
	ExperimentType string = "EXPERIMENT"
	// NTSCType constant.
	NTSCType string = "NTSC"
)

// TaskConfigPolicies is the bun model of a task config policy.
type TaskConfigPolicies struct {
	bun.BaseModel   `bun:"table:task_config_policies"`
	WorkspaceID     *int      `bun:"workspace_id"`
	WorkloadType    string    `bun:"workload_type,notnull"`
	LastUpdatedBy   UserID    `bun:"last_updated_by,notnull"`
	LastUpdatedTime time.Time `bun:"last_updated_time,notnull"`
	InvariantConfig *string   `bun:"invariant_config"`
	Constraints     *string   `bun:"constraints"`
}

// ResourceConstraints are non-overridable resource constraints.
// Submitted workloads that request resource quanities exceeding defined resource constraints in a
// given scope are rejected.
type ResourceConstraints struct {
	MaxSlots      *int `json:"max_slots"`
	PriorityLimit *int `json:"priority_limit"`
}

// Constraints are non-overridable workload constraints.
// Submitted workloads whose config's respective field(s) exceed defined constraints within a given
// scope are rejected.
type Constraints struct {
	ResourceConstraints *ResourceConstraints `json:"resources"`
}
