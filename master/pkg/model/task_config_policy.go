package model

import (
	"strings"
	"time"

	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/proto/pkg/configpolicyv1"
)

// WorkloadType is the type of workload (experiment or NTSC) that the task config policy applies to.
type WorkloadType string

// Constants.

const (
	// UnknownType constant.
	UnknownType WorkloadType = "UNSPECIFIED"
	// ExperimentType constant.
	ExperimentType WorkloadType = "EXPERIMENT"
	// NTSCType constant.
	NTSCType WorkloadType = "NTSC"
)

// ExperimentTaskConfigPolicies is the bun model of a task config policy.
type ExperimentTaskConfigPolicies struct {
	bun.BaseModel   `bun:"table:task_config_policies"`
	WorkspaceID     *int                     `bun:"workspace_id"`
	LastUpdatedBy   UserID                   `bun:"last_updated_by,notnull"`
	LastUpdatedTime time.Time                `bun:"last_updated_time,notnull"`
	WorkloadType    WorkloadType             `bun:"workload_type,notnull"`
	InvariantConfig expconf.ExperimentConfig `bun:"invariant_config"`
	Constraints     Constraints              `bun:"constraints"`
}

// NTSCTaskConfigPolicies is the bun model of a task config policy.
type NTSCTaskConfigPolicies struct {
	bun.BaseModel   `bun:"table:task_config_policies"`
	WorkspaceID     *int          `bun:"workspace_id"`
	LastUpdatedBy   UserID        `bun:"last_updated_by,notnull"`
	LastUpdatedTime time.Time     `bun:"last_updated_time,notnull"`
	WorkloadType    WorkloadType  `bun:"workload_type,notnull"`
	InvariantConfig CommandConfig `bun:"invariant_config"`
	Constraints     Constraints   `bun:"constraints"`
}

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

// WorkloadTypeFromProto maps taskconfigpolicyv1.WorkloadType to WorkloadType.
func WorkloadTypeFromProto(workloadType configpolicyv1.WorkloadType) WorkloadType {
	str := workloadType.String()
	return WorkloadType(strings.TrimPrefix(str, "WORKLOAD_TYPE_"))
}

// WorkloadTypeToProto maps WorkloadType to taskconfigpolicyv1.WorkloadType.
func WorkloadTypeToProto(workloadType WorkloadType) configpolicyv1.WorkloadType {
	protoWorkloadType := configpolicyv1.WorkloadType_value["WORKLOAD_TYPE_"+string(workloadType)]
	return configpolicyv1.WorkloadType(protoWorkloadType)
}

func (w WorkloadType) String() string {
	return string(w)
}
