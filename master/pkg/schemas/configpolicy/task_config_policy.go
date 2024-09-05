package configpolicy

// ResourceConstraints are non-overridable resource constraints.
// Submitted workloads that request resource quanities exceeding defined resource constraints in a
// given scope are rejected.
type ResourceConstraints struct {
	MaxSlots *int `json:"max_slots"`
}

// Constraints are non-overridable workload constraints.
// Submitted workloads whose config's respective field(s) exceed defined constraints within a given
// scope are rejected.
type Constraints struct {
	ResourceConstraints *ResourceConstraints `json:"resources"`
	PriorityLimit       *int                 `json:"priority_limit"`
}
